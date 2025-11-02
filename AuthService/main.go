package main

import (
	"auth-service/handler"
	"auth-service/repository"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func connectDB(uri string) *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse the URI and setup client options
	clientOptions := options.Client().ApplyURI(uri)

	// connect
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB: ", err)
	}

	// ping the database to verify the connection
	if err = client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB: ", err)
	}

	fmt.Println("Successfully connected to MongoDB!")
	return client
}

func main() {
	// Connect to DB
	mongoURI := "mongodb://localhost:27017"
	client := connectDB(mongoURI)

	// Setup repositories
	userRepository := repository.NewUserRepository(client, "default", "user")

	// Handlers
	healthHandler := handler.HealthHandler{}
	authHandler := handler.AuthHandler{UserRepository: userRepository}

	// Server
	mux := http.NewServeMux()
	mux.Handle("/health", healthHandler)
	mux.Handle("/register", http.HandlerFunc(authHandler.RegisterUser))
	mux.Handle("/login", http.HandlerFunc(authHandler.LoginUser))
	mux.Handle("/authenticate", http.HandlerFunc(authHandler.AuthenticateRequest))

	fmt.Println("Starting server on port 8080...")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}
