package main

import (
	"DocumentUpdatesConsumer/config"
	"DocumentUpdatesConsumer/database"
	"DocumentUpdatesConsumer/handler"
	"DocumentUpdatesConsumer/repository"
	"DocumentUpdatesConsumer/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const (
	kafkaBroker = "canvas-live-kafka:9092"
	topic       = "document-updates"
	groupID     = "document-updates-consumer-group"
)

// connectConsumerWithRetry loops until a broker connection is viable
func connectConsumerWithRetry(brokers, group string) *kafka.Consumer {
	var consumer *kafka.Consumer
	var err error
	retryInterval := 5 * time.Second

	for {
		fmt.Printf("Attempting to connect consumer to %s...\n", brokers)
		consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":        brokers,
			"group.id":                 group,
			"auto.offset.reset":        "earliest",
			"socket.timeout.ms":        10000,
			"session.timeout.ms":       30000,
			"heartbeat.interval.ms":    3000,
			"allow.auto.create.topics": true,
		})

		if err == nil {
			// Check metadata to verify broker is reachable
			_, err = consumer.GetMetadata(nil, false, 10000)
			if err == nil {
				fmt.Println("Successfully connected to Kafka Broker!")
				return consumer
			}
			consumer.Close()
		}

		fmt.Printf("Connection failed: %v. Retrying in %v...\n", err, retryInterval)
		time.Sleep(retryInterval)
	}
}

// subscribeWithRetry attempts to subscribe to the topic with retry logic
func subscribeWithRetry(consumer *kafka.Consumer, topic string) {
	retryInterval := 5 * time.Second
	maxRetries := 20
	retries := 0

	for retries < maxRetries {
		err := consumer.SubscribeTopics([]string{topic}, nil)
		if err == nil {
			fmt.Printf("Successfully subscribed to topic: %s\n", topic)
			return
		}

		fmt.Printf("Failed to subscribe to topic %s (attempt %d/%d): %v\n",
			topic, retries+1, maxRetries, err)

		retries++
		time.Sleep(retryInterval)
	}

	log.Fatalf("Failed to subscribe to topic after %d attempts", maxRetries)
}

// ensureTopicExists creates an admin client and ensures the topic exists
func ensureTopicExists(brokers, topicName string) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}
	defer adminClient.Close()

	// Check if topic already exists
	metadata, err := adminClient.GetMetadata(&topicName, false, 5000)
	if err == nil && len(metadata.Topics) > 0 {
		fmt.Printf("Topic %s already exists\n", topicName)
		return nil
	}

	// Create topic
	fmt.Printf("Creating topic %s...\n", topicName)
	topicSpec := kafka.TopicSpecification{
		Topic:             topicName,
		NumPartitions:     3,
		ReplicationFactor: 1,
	}

	results, err := adminClient.CreateTopics(
		context.Background(),
		[]kafka.TopicSpecification{topicSpec},
		kafka.SetAdminOperationTimeout(30*time.Second),
	)

	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError &&
			result.Error.Code() != kafka.ErrTopicAlreadyExists {
			return fmt.Errorf("failed to create topic %s: %s",
				result.Topic, result.Error.String())
		}
		fmt.Printf("Topic %s created successfully\n", result.Topic)
	}

	// Wait a bit for topic to be fully available
	time.Sleep(2 * time.Second)
	return nil
}

func main() {
	// Connect to DB
	client := database.ConnectDB(config.MongoConfig.MongoUri)

	// Repository
	r := repository.NewDocumentRepository(
		client,
		config.MongoConfig.DatabaseName,
		config.MongoConfig.DocumentCollectionName,
	)

	// Ensure topic exists before creating consumer
	fmt.Println("Ensuring Kafka topic exists...")
	if err := ensureTopicExists(kafkaBroker, topic); err != nil {
		log.Printf("Warning: Could not ensure topic exists: %v", err)
		log.Println("Continuing anyway - topic may be auto-created on first message")
	}

	// Create Kafka consumer
	fmt.Println("Trying to connect to Kafka!")
	c := connectConsumerWithRetry(kafkaBroker, groupID)
	defer c.Close()
	fmt.Println("Connected to Kafka!")

	// Subscribe to topic with retry
	subscribeWithRetry(c, topic)
	fmt.Printf("Subscribed to topic %s. Waiting for messages...\n", topic)

	// Setup graceful shutdown
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	// Start consuming messages
	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Received signal %v: terminating\n", sig)
			run = false

		default:
			// Poll for Kafka messages
			ev := c.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Process the consumed message
				fmt.Printf("Received message from topic %s: %s\n",
					*e.TopicPartition.Topic, string(e.Value))

				// Parse message into struct
				var msg types.Message
				if err := json.Unmarshal(e.Value, &msg); err != nil {
					fmt.Printf("[Error] Can't unmarshal message: %v\n", err)
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				go func() {
					defer cancel()
					handler.DocumentUpdatesHandler(ctx, r, msg)
				}()

			case kafka.Error:
				// Handle Kafka errors
				fmt.Printf("Kafka Error: %v (Code: %d)\n", e, e.Code())

				// Check if it's a fatal error
				if e.Code() == kafka.ErrAllBrokersDown {
					fmt.Println("All brokers are down, attempting reconnect...")
					run = false
				}

			default:
				// Ignore other event types
			}
		}
	}

	fmt.Println("Consumer shutting down...")
}
