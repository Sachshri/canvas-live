package handler

import (
	"UpdatesService/redis"
	"UpdatesService/websocket"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================== Helper Functions ========================================

const (
	authServiceURL = "http://auth-service:8081/auth/authenticate" // Adjust to your auth service
)

// UserInfo holds authenticated user data
type UserInfo struct {
	UserID   string
	Username string
}

// authenticateToken validates JWT token by calling auth service
func authenticateToken(token string) (*UserInfo, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create request to auth service
	req, err := http.NewRequest("GET", authServiceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	// Add Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	log.Printf("Authenticating token with auth service...")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach auth service: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("authentication failed: %s", string(body))
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("auth endpoint not found (404)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Extract user info from response headers
	userID := resp.Header.Get("X-User-ID")
	username := resp.Header.Get("X-Username")

	if userID == "" {
		return nil, fmt.Errorf("auth service did not return X-User-ID header")
	}

	log.Printf("Authentication successful for user: %s (%s)", username, userID)

	return &UserInfo{
		UserID:   userID,
		Username: username,
	}, nil
}

func WsHandler(pool *websocket.Pool, redis_client *redis.RedisClient) gin.HandlerFunc {
	// Return a Gin handler function
	return func(c *gin.Context) {
		docId := c.Param("docId")
		jwtToken := c.Param("token")
		if docId == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "documentId missing"})
			return
		}
		// 1. Authentication Check (Using c.Request)
		// Access header directly from the raw http.Request object
		userInfo, err := authenticateToken(jwtToken)
		if err != nil {
			fmt.Printf("[WsHandler][Error] %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization failed"})
			return
		}
		userId := userInfo.UserID
		username := userInfo.Username
		if userId == "" {
			// Use Gin's method to send HTTP error response before upgrade
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		// 2. Perform WebSocket Upgrade (Using c.Writer and c.Request)
		conn, err := websocket.Upgrade(c.Writer, c.Request)
		if err != nil {
			// Log error after upgrade attempt, as headers may already be sent
			log.Printf("WebSocket Upgrade Failed: %v", err)
			// Note: Since upgrade failed, you cannot use c.JSON here
			return
		}

		// 3. Initialize and Register Client
		client := &websocket.Client{
			UserID:      userId,
			Username:    username,
			DocumentID:  docId, // Ensure this is correctly retrieved or set
			Conn:        conn,
			Pool:        pool,
			Send:        make(chan []byte),
			RedisClient: redis_client,
		}

		fmt.Println("[WsHandler] client reader running!")
		go client.Writer() // Start a goroutine responsible for send message(it receives via Send channel) to the client
		fmt.Println("[WsHandler] client Writer running!")

		pool.Register <- client
		client.Read() // Start the client's read loop
	}
}
