package repository

import (
	"context"
	"document-service/model"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DocumentRepository struct {
	collection                *mongo.Collection
	sharedDocRecordCollection *mongo.Collection
}

func NewDocumentRepository(client *mongo.Client, database string, collection string, sharedDocCollectionName string) *DocumentRepository {
	coll := client.Database(database).Collection(collection)
	shared := client.Database(database).Collection(sharedDocCollectionName)
	return &DocumentRepository{
		collection:                coll,
		sharedDocRecordCollection: shared,
	}
}

func (r *DocumentRepository) FindDocumentByID(ctx context.Context, docID string) (*model.Document, error) {
	// We derive a context with a timeout from the request context
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 1. Convert the string ID to a primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(docID)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID format: %w", err)
	}

	// 2. Define the filter
	filter := bson.M{"_id": objectID}

	// 3. Execute FindOne
	var document model.Document

	// Chain FindOne with Decode.
	err = r.collection.FindOne(ctx, filter).Decode(&document)

	// 4. Handle Errors
	if err != nil {
		// A. Check for the specific "Not Found" error
		if err == mongo.ErrNoDocuments {
			// Return nil document and nil error (success, but nothing found)
			return nil, nil
		}

		// B. Handle other system/database errors
		log.Printf("[Repository] Database query failed: %v", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// 5. Return the successfully decoded document
	return &document, nil
}

func (r *DocumentRepository) CreateNewDocument(ctx context.Context, title string, ownerId string) (model.Document, error) {

	// Create a Document
	emptyDocument := model.Document{
		Title:   title,
		OwnerID: ownerId,
		Slides: []model.Slide{
			{
				ID:         primitive.NewObjectID().Hex(),
				Background: "#FFFFFF",
				Objects:    make([]model.Object, 0, 1),
			},
		},
	}

	// Insert Document
	result, err := r.collection.InsertOne(ctx, emptyDocument)
	if err != nil {
		return model.Document{}, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		emptyDocument.ID = oid
	}

	return emptyDocument, nil
}

func (r *DocumentRepository) DeleteDocument(ctx context.Context, id string) error {
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Printf("[DocumentRepository] Invalid document id: %v\n", err)
		return err
	}
	fmt.Printf("[DocumentRepository][FindOwnedDocuments] Error decoding documents: %v\n", err)

	filter := bson.M{"_id": objectId}

	// Execute Deletion
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		fmt.Printf("[DocumentRepository] Error deleting document: %v\n", err)
		return err
	}

	if result.DeletedCount == 1 {
		fmt.Printf("[DocumentRepository] Successfully deleted 1 document with ID: %s\n", id)
	} else {
		fmt.Printf("[DocumentRepository] No document found with ID: %s\n", id)
	}

	return nil
}

func (r *DocumentRepository) FindOwnedDocuments(ctx context.Context, userId string) ([]model.Document, error) {

	filter := bson.M{"ownerId": userId}
	// Execute the query
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		fmt.Printf("[DocumentRepository][FindOwnedDocuments] Error retrieving documents: %v\n", err)
		return []model.Document{}, err
	}
	defer cursor.Close(ctx)

	// Decode all Documents in documents slice
	documents := []model.Document{}
	if err = cursor.All(ctx, &documents); err != nil {
		fmt.Printf("[DocumentRepository][FindOwnedDocuments] Error decoding documents: %v\n", err)
		return []model.Document{}, err
	}

	return documents, nil
}

func (r *DocumentRepository) FindSharedDocuments(ctx context.Context, userId string) ([]model.Document, error) {

	filter := bson.M{"userId": userId}

	// Get IDs of documents shared with the current user
	cursor, err := r.sharedDocRecordCollection.Find(ctx, filter)
	if err != nil {
		fmt.Printf("[DocumentRepository][FindSharedDocuments] Error retrieving shared document records: %v\n", err)
		return []model.Document{}, err
	}
	defer cursor.Close(ctx)

	var sharedDocRecords []model.CollaborationRecord
	if err = cursor.All(ctx, &sharedDocRecords); err != nil {
		fmt.Printf("[DocumentRepository][FindSharedDocuments] Error decoding shared document records: %v\n", err)
		return []model.Document{}, err
	}

	var ids []primitive.ObjectID
	for _, record := range sharedDocRecords {
		objectId, err := primitive.ObjectIDFromHex(record.DocumentID)
		if err != nil {
			continue
		}
		ids = append(ids, objectId)
	}

	// Get documents
	// if ids is empty return empty slice
	if len(ids) == 0 {
		return []model.Document{}, nil
	}

	filter = bson.M{
		"_id": bson.M{"$in": ids},
	}

	cursor, err = r.collection.Find(ctx, filter)
	if err != nil {
		fmt.Printf("[DocumentRepository][FindSharedDocuments] Error retrieving documents: %v\n", err)
		return []model.Document{}, err
	}
	defer cursor.Close(ctx)

	var documents []model.Document
	if err = cursor.All(ctx, &documents); err != nil {
		fmt.Printf("[DocumentRepository][FindSharedDocuments] Error decoding documents: %v\n", err)
		return []model.Document{}, nil
	}

	return documents, nil
}

func (r *DocumentRepository) IsDocumentOwnedByUser(ctx context.Context, userId string, documentId string) (bool, error) {

	documentObjectId, err := primitive.ObjectIDFromHex(documentId)
	if err != nil {
		fmt.Printf("[DocumentRepository][IsDocumentOwnedByUser] Invalid document id: %v\n", err)
		return false, err
	}

	// retrieve documents
	filter := bson.M{"_id": documentObjectId}

	var document model.Document
	err = r.collection.FindOne(ctx, filter).Decode(&document)
	if err != nil {
		fmt.Printf("[DocumentRepository][IsDocumentOwnedByUser] Error retrieving or decoding document: %v\n", err)
		return false, err
	}

	if document.OwnerID == userId {
		return true, nil
	}

	return false, nil
}

func (r *DocumentRepository) CreateCollaborationRecord(ctx context.Context, collaboratorUserId string, documentId, accessType string) (model.CollaborationRecord, error) {

	// Create shared document record object
	sharedDocRecord := model.CollaborationRecord{
		UserID:     collaboratorUserId,
		DocumentID: documentId,
		AccessType: accessType,
	}

	// Execute the query
	result, err := r.sharedDocRecordCollection.InsertOne(ctx, sharedDocRecord)
	if err != nil {
		fmt.Printf("[DocumentRepository] Error creating sharing record: %v\n", err)
		return model.CollaborationRecord{}, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		sharedDocRecord.ID = oid
	}

	return sharedDocRecord, nil
}
