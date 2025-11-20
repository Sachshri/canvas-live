package kafkaUtils

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const (
	KafkaBroker = "canvas-live-kafka:9092"
	Topic       = "document-updates"
)

func ProduceMessage(p *kafka.Producer, topic string, message []byte) error {

	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}

	// Produce the kafka message
	deliveryChan := make(chan kafka.Event)
	err := p.Produce(kafkaMessage, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// wait for delivery report or error
	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %s", m.TopicPartition.Error)
	}

	// close the delivery chanel
	close(deliveryChan)

	return nil
}
