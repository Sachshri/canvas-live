package websocket

import (
	"UpdatesService/kafkaUtils"
	"UpdatesService/types"
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Pool struct {
	Register      chan *Client
	Unregister    chan *Client
	RoomBroadcast chan types.Message
	PushToKafka   chan types.KafkaInterMessage
	Rooms         map[string]map[*Client]bool
	KafkaProducer *kafka.Producer
}

func NewPool(p *kafka.Producer) *Pool {
	return &Pool{
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		RoomBroadcast: make(chan types.Message),
		Rooms:         make(map[string]map[*Client]bool),
		KafkaProducer: p,
		PushToKafka:   make(chan types.KafkaInterMessage),
	}
}

func SerializeMessage(message types.Message) ([]byte, error) {
	serialized, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}
	return serialized, nil
}

func (pool *Pool) Start() types.Message {
	for {
		select {
		case client := <-pool.Register:
			fmt.Println("Trying to register a client")

			// initialize the inner map
			if _, ok := pool.Rooms[client.DocumentID]; !ok {
				pool.Rooms[client.DocumentID] = make(map[*Client]bool)
			}

			pool.Rooms[client.DocumentID][client] = true

			// send joining message to each particpant of the room
			for c := range pool.Rooms[client.DocumentID] {
				message, err := json.Marshal(types.Message{
					DocumentID: c.DocumentID,
					UserID:     c.UserID,
					Username:   c.Username,
					Type:       1,
					Body:       "New user joined",
				})

				if err != nil {
					fmt.Println("[Pool][Register] json marshalling error")
					break
				}

				fmt.Println("[Pool][Register] Sending new user joined message")
				client.Send <- message
			}
			fmt.Println("Client registered")

		case client := <-pool.Unregister:
			delete(pool.Rooms[client.DocumentID], client)
			// send disconnection message to each participant of the room
			for c := range pool.Rooms[client.DocumentID] {
				message, err := json.Marshal(types.Message{
					DocumentID: c.DocumentID,
					UserID:     c.UserID,
					Username:   c.Username,
					Type:       1,
					Body:       "User disconnected",
				})

				if err != nil {
					fmt.Println("[Pool][Unregister] json marshalling error")
					continue
				}

				client.Send <- message
			}

		case message := <-pool.RoomBroadcast:
			fmt.Printf("Broadcasting to room -> ")
			for client := range pool.Rooms[message.DocumentID] {
				if client.UserID == message.UserID {
					continue
				}

				// Convert message (struct) to []byte
				jsonData, err := json.Marshal(message)
				if err != nil {
					fmt.Println("[Pool][RoomBroadcast] json Marshalling error")
					break
				}

				client.Send <- jsonData
				// if err := client.Conn.WriteJSON(message); err != nil {
				// 	fmt.Println("[Pool][RoomBroadcast]", err)
				// 	continue
				// }
			}

			fmt.Println("Broadcasted!")

		case message := <-pool.PushToKafka:
			fmt.Println("[Pool][PushToKafka] Pushing message to kafka!")
			serialized, err := SerializeMessage(message.Message)
			if err != nil {
				fmt.Println("[Pool][PushToKafka]", err)
				break
			}
			err = kafkaUtils.ProduceMessage(pool.KafkaProducer, message.Topic, serialized)
			if err != nil {
				fmt.Println("[Pool][PushToKafka] Error pushing message to kafka: ", err)
			}
		}

	}
}
