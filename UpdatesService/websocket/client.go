package websocket

import (
	"UpdatesService/redis"
	"UpdatesService/types"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID      string
	Username    string
	DocumentID  string
	Conn        *websocket.Conn
	Pool        *Pool
	Send        chan []byte
	RedisClient *redis.RedisClient
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			fmt.Println("[Client Reader] Error reading message")
			return
		}

		switch messageType {
		case 1: // Text message
			fmt.Printf("[Client Reader] Received TEXT data: %s\n", string(p))

			// Data validation
			err := c.HandleMessage(p)
			if err != nil {
				fmt.Printf("[Error] %s", err)
				c.FailureResponseMessage()
			} else {
				c.SuccessResponseMessage()
			}

		case 2: // Binary message
			fmt.Printf("[Client Reader] Received BINARY data (%d bytes)\n", len(p))
		}

	}
}

func (c *Client) Writer() {
	// PING / PONG Connection Keep-Alive mechanism
	pongWait := 60 * time.Second      // The maximum time server will wait for a pong message before assuming that the connection is dead
	pingPeriod := (pongWait * 9) / 10 // The interval at which the server sends a PING message
	const writeWait = 10 * time.Second

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			fmt.Println("[Client Writer] Received message")
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				fmt.Println("[Client Writer] Error receiving message from Send channel!")
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				fmt.Println("[Client Writer] Failed to send message")
				return // Exit the loop on failure
			}

		case <-ticker.C: // Ping trigger
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("[Client Writer] PING fails")
				return // Exit the loop if PING fails (connection is likely dead)
			}
		}
	}

}

func (c *Client) HandleMessage(p []byte) error {

	var msg map[string]interface{}
	if err := json.Unmarshal(p, &msg); err != nil {
		fmt.Printf("[Client Reader] Error Unmarshaling Action Message - %s\n", err)
		// Send Error message to client
		// c.Send <- []byte("[Error] Invalid message format - Only JSON is allowed.")
		return err
	}

	// Read action message
	actVal, ok := msg["action"]
	if !ok {
		fmt.Println("[Client Reader] action key not available in message")
		// c.Send <- []byte("[Error] Invalid message format - action key missing")
		return fmt.Errorf("[Error] action key missing")
	}
	actionStr, ok := actVal.(string)
	if !ok {
		fmt.Println("[Client Reader] action key is not a string")
		// c.Send <- []byte("[Error] Invalid message format - action key must be a string")
		return fmt.Errorf("[Error] action key is not a string")
	}

	outMsg := types.Message{
		DocumentID: c.DocumentID,
		Username:   c.Username,
		UserID:     c.UserID,
		Type:       1,
		Body:       string(p),
	}

	// Data Validation
	// If action is cursorMove
	switch actionStr {
	case "cursormove":
		if types.ValidateCursorMoveMessage(msg) {
			c.Broadcast(outMsg)
		}

	case "create":

		if types.ValidateCreateMessage(msg) {
			attr, ok := msg["attributes"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] attributes missing")

			}
			objectType, ok := msg["objectType"]
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] objectType missing")
			}
			objectId, ok := msg["objectId"].(string)
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] objectId missing")
			}

			if objectType == "rectangle" && types.ValidateRectangleAttributes(attr) {
				if err := c.CheckLockAndBroadcastAndPushToKafka(outMsg, objectId); err != nil {
					return err
				}
			}
			if objectType == "circle" && types.ValidateCircleAttributes(attr) {
				if err := c.CheckLockAndBroadcastAndPushToKafka(outMsg, objectId); err != nil {
					return err
				}
			}
			if objectType == "text" && types.ValidateTextAttributes(attr) {
				if err := c.CheckLockAndBroadcastAndPushToKafka(outMsg, objectId); err != nil {
					return err
				}
			}

		}

	case "update":
		if types.ValidateUpdateMessage(msg) {
			objectId, ok := msg["objectId"].(string)
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] objectId missing")
			}

			if err := c.CheckLockAndBroadcastAndPushToKafka(outMsg, objectId); err != nil {
				return err
			}
		}
	case "delete":
		if types.ValidateDeleteMessage(msg) {
			objectId, ok := msg["objectId"].(string)
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] objectId missing")
			}

			if err := c.CheckLockAndBroadcastAndPushToKafka(outMsg, objectId); err != nil {
				return err
			}
		}
	case "select":
		if types.ValidateSelectMessage(msg) {
			objectId, ok := msg["objectId"].(string)
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] objectId missing")
			}

			if err := c.CheckLockAndBroadcast(outMsg, objectId); err != nil {
				return err
			}
		}
	case "deselect":
		if types.ValidateSelectMessage(msg) {
			objectId, ok := msg["objectId"].(string)
			if !ok {
				return fmt.Errorf("[Client][HandleMessage][Error] objectId missing")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			anyKeyDeleted, err := c.RedisClient.ReleaseLock(ctx, objectId)
			if err != nil {
				return err
			}

			// if the object had been selected then it has been deleted
			if anyKeyDeleted {
				c.Broadcast(outMsg)
			}

		}
	case "add_slide":
		if types.ValidateAddSlideMessage(msg) {
			c.BroadcastAndPushToKafka(outMsg)
		}
	case "remove_slide":
		if types.ValidateRemoveSlideMessage(msg) {
			c.BroadcastAndPushToKafka(outMsg)
		}
	default:
		// c.Send <- []byte("[Error] Invalid m essage format")
		return fmt.Errorf("[Client][HandleMessage][Error] Invalid message format received")
	}

	return nil
}

func (c *Client) CheckLockAndBroadcast(outMsg types.Message, objectId string) error {

	// Check Exclusive Lock[]
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := c.RedisClient.SetExclusiveLock(ctx, objectId, outMsg.UserID, 10*time.Minute); err != nil {
		// The lock is not free
		return err
	}

	// broadcast message to everyone in the room
	c.Pool.RoomBroadcast <- outMsg
	fmt.Printf("Message Received: %+v\n", outMsg)
	return nil
}

func (c *Client) CheckLockAndBroadcastAndPushToKafka(outMsg types.Message, objectId string) error {

	// Check Exclusive Lock[]
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := c.RedisClient.SetExclusiveLock(ctx, objectId, outMsg.UserID, 10*time.Minute); err != nil {
		// The lock is not free
		return fmt.Errorf("[Error] Lock is not free")
	}

	// broadcast message to everyone in the room
	c.Pool.RoomBroadcast <- outMsg
	fmt.Printf("Message Received: %+v\n", outMsg)

	// push to kafka
	kafkaMessage := types.KafkaInterMessage{Topic: "document-updates", Message: outMsg}
	c.Pool.PushToKafka <- kafkaMessage

	return nil
}

func (c *Client) BroadcastAndPushToKafka(outMsg types.Message) {
	// broadcast message to everyone in the room
	c.Pool.RoomBroadcast <- outMsg
	fmt.Printf("Message Received: %+v\n", outMsg)

	// push to kafka
	kafkaMessage := types.KafkaInterMessage{Topic: "document-updates", Message: outMsg}
	c.Pool.PushToKafka <- kafkaMessage
}

func (c *Client) Broadcast(outMsg types.Message) {
	// broadcast message to everyone in the room
	c.Pool.RoomBroadcast <- outMsg
	fmt.Printf("Message Received: %+v\n", outMsg)

	// return nil
}

func (c *Client) FailureResponseMessage() error {
	msg := types.ServerResponseMessage{Success: false}
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("[Error] failure to marshal server response message")
	}
	c.Send <- jsonBytes
	return nil
}

func (c *Client) SuccessResponseMessage() error {
	msg := types.ServerResponseMessage{Success: true}
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("[Error] failure to marshal server response message")
	}
	c.Send <- jsonBytes
	return nil
}
