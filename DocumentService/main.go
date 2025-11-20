package main

import (
	"context"
	"document-service/config"
	"document-service/database"
	"document-service/handler"
	"document-service/repository"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to DB
	client := database.ConnectDB(config.MongoConfig.MongoUri)
	defer client.Disconnect(context.Background()) // Ensure DB connection closes

	// Set up Repositories
	DocumentRepository := repository.NewDocumentRepository(
		client,
		config.MongoConfig.DatabaseName,
		config.MongoConfig.DocumentCollectionName,
		config.MongoConfig.SharedDocRecordCollectionName,
	)

	// Set up Handlers
	documentHandler := handler.DocumentHandler{DocumentRepository: DocumentRepository}

	// ===============================================
	// GIN ROUTER SETUP
	// ===============================================

	// 1. Initialize Gin Router (Default includes Logger and Recovery middleware)
	router := gin.Default()

	// 2. Apply Custom Middleware (If needed)
	// NOTE: If RequestLoggingMiddleware is adapted to return gin.HandlerFunc, use router.Use()
	// For simplicity, if we assume middleware.RequestLoggingMiddleware is adapted, we would use:
	// router.Use(middleware.RequestLoggingMiddleware)

	// 3. Register Routes using a Group
	documentGroup := router.Group("/document")
	{
		// POST /document/create
		documentGroup.POST("/create", documentHandler.CreateNewDocument)

		// GET /document/all
		documentGroup.GET("/all", documentHandler.GetAllDocuments)

		// POST /document/share
		documentGroup.POST("/share", documentHandler.ShareDocument)

		// POST /document/delete
		documentGroup.POST("/delete", documentHandler.DeleteDocument)

		// GET /document/id/:id
		documentGroup.GET("/id/:id", documentHandler.GetDocumentByID)
	}

	// Optional: Simple health check route
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 4. Start the Server
	fmt.Println("Starting server on port 8082 with Gin...")

	// Gin's router handles listening and serving
	if err := router.Run(":8082"); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}
