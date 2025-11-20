package handler

import (
	"document-service/repository"
	"document-service/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ===========================================

type DocumentHandler struct {
	DocumentRepository *repository.DocumentRepository
}

// Helper to get authenticated UserID (assuming it's set in a middleware header)
func getAuthUserID(c *gin.Context) (string, bool) {
	// Retrieving directly from the raw request header
	userId := c.Request.Header.Get("X-User-ID")
	if userId == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return "", false
	}
	return userId, true
}

// ====================== Get all documents handler =======================================

// GetAllDocuments returns a Gin HandlerFunc to retrieve all documents owned by or shared with the user.
func (h DocumentHandler) GetAllDocuments(c *gin.Context) {
	// The router (router.GET) already ensures r.Method is GET

	// Retrieve user data
	userId, ok := getAuthUserID(c)
	if !ok {
		return // Response already sent by helper
	}

	// Get all owned documents
	ownedDocuments, err := h.DocumentRepository.FindOwnedDocuments(c, userId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving owned documents"})
		return
	}

	// Get all shared documents
	sharedDocuments, err := h.DocumentRepository.FindSharedDocuments(c, userId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving shared documents"})
		return
	}

	result := types.AllDocumentsDto{OwnedDocuments: ownedDocuments, SharedDocuments: sharedDocuments}

	// Json response
	c.JSON(http.StatusOK, result)
}

// ================================ Create New Empty Document Handler ===========================

// CreateNewDocument returns a Gin HandlerFunc to create a new document.
func (h DocumentHandler) CreateNewDocument(c *gin.Context) {
	// The router (router.POST) already ensures r.Method is POST

	// Retrieve user data
	userId, ok := getAuthUserID(c)
	if !ok {
		return
	}

	// Create document
	createdDoc, err := h.DocumentRepository.CreateNewDocument(c, "Untitled", userId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error creating document"})
		return
	}

	response := types.CreatedResponse{ID: createdDoc.ID.Hex()}

	c.JSON(http.StatusCreated, response) // Use 201 Created status
}

// ================================= Share Document Handler ==============================

// ShareDocument returns a Gin HandlerFunc to create a new sharing record.
func (h DocumentHandler) ShareDocument(c *gin.Context) {
	// The router (router.POST) already ensures r.Method is POST

	// Retrieve user data
	userId, ok := getAuthUserID(c)
	if !ok {
		return
	}

	// Decode and bind data from request body
	var data types.ShareDocumentPostData
	// Gin's ShouldBindJSON handles decoding and error check
	if err := c.ShouldBindJSON(&data); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid data format or missing fields"})
		return
	}

	// Check if the user actually owns the document
	isUserOwner, err := h.DocumentRepository.IsDocumentOwnedByUser(c, userId, data.DocumentID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error verifying ownership of the document"})
		return
	}

	if !isUserOwner {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Only the owner can share documents with other users"})
		return
	}

	// Create sharing record
	// NOTE: Using the context provided by Gin (c.Request.Context() is implicit in Gin handler functions)
	_, err = h.DocumentRepository.CreateCollaborationRecord(c, data.CollaboratorUserID, data.DocumentID, data.AccessType)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error creating a collaboration record"})
		return
	}

	c.String(http.StatusOK, "Success")
}

// ================================= Delete Document Handler ==============================

// DeleteDocument returns a Gin HandlerFunc to delete a document.
func (h DocumentHandler) DeleteDocument(c *gin.Context) {
	// The router (router.POST) already ensures r.Method is POST

	// Retrieve user data
	userId, ok := getAuthUserID(c)
	if !ok {
		return
	}

	// Decode and bind data from request body
	var data types.DeleteDocumentPostData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid data format or missing fields"})
		return
	}

	// Check if the user actually owns the document
	isUserOwner, err := h.DocumentRepository.IsDocumentOwnedByUser(c, userId, data.DocumentID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error verifying ownership of the document"})
		return
	}

	if !isUserOwner {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Only the owner can delete their documents"})
		return
	}

	// Delete document
	err = h.DocumentRepository.DeleteDocument(c, data.DocumentID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error deleting document: %s", err.Error())})
		return
	}

	c.String(http.StatusOK, "Success")
}

// Route: GET /document/:id
func (h DocumentHandler) GetDocumentByID(c *gin.Context) {
	// 1. Get Path Parameter
	docID := c.Param("id")
	if docID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Document ID is required in the path"})
		return
	}

	// 2. Auth Check (optional, but good practice before database access)
	// You should ideally check if the authenticated user has access to this document.
	// userID, ok := getAuthUserID(c)
	// if !ok {
	// 	return
	// }

	// 3. Call Repository to find the document
	document, err := h.DocumentRepository.FindDocumentByID(c.Request.Context(), docID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving document"})
		return
	}

	// 4. Handle Not Found (Repository returns nil, nil for ErrNoDocuments)
	if document == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// 5. Authorization Check (if not owner, check sharing)
	// Add logic here to check if userID is the owner or in shared list

	// 6. Return Document
	c.JSON(http.StatusOK, document)
}
