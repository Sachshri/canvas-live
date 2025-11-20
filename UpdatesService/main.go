package main

import (
	"UpdatesService/handler"
	"UpdatesService/kafkaUtils"
	"UpdatesService/redis"
	"UpdatesService/websocket"
	"fmt"
	"net/http"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
)

func connectProducer(brokers string) (*kafka.Producer, error) {
	var producer *kafka.Producer
	var err error

	maxRetries := 30
	retryInterval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Attempting to connect Producer to Kafka (Attempt %d/%d)...\n", i+1, maxRetries)

		producer, err = kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": brokers,
		})

		if err == nil {
			// Verify connection by requesting metadata.
			// NewProducer is lazy; this forces a network call.
			_, err = producer.GetMetadata(nil, false, 5000)
			if err == nil {
				fmt.Println("Successfully connected Producer to Kafka!")
				return producer, nil
			}
			// Cleanup failed producer instance
			producer.Close()
		}

		fmt.Printf("Failed to connect Producer: %v. Retrying in %v...\n", err, retryInterval)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to connect producer after %d attempts: %w", maxRetries, err)
}

func main() {
	// kafka Setup
	fmt.Println("Trying to connect to Kafka!")
	p, err := connectProducer(kafkaUtils.KafkaBroker)
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		return
	}
	defer p.Close()
	fmt.Println("Connected to Kafka!")

	// Redis Setup
	redis_client := redis.NewRedisClient("canvas-live-redis:6379")

	// Websocket pool
	pool := websocket.NewPool(p)
	go pool.Start()

	// Server setup
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Server running.")
	})

	router.GET("/updates/ws/docId/:docId/token/:token", handler.WsHandler(pool, redis_client))

	router.Run(":8083")
}
