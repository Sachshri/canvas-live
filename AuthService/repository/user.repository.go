package repository

import (
	"auth-service/model"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepository handles all database interactions for the User model.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new repository instance.
func NewUserRepository(client *mongo.Client, database string, collection string) *UserRepository {
	// The client, database name, and collection name are passed during initialization.
	coll := client.Database(database).Collection(collection)
	return &UserRepository{
		collection: coll,
	}
}

// Save inserts a new User document into the collection.
func (r *UserRepository) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	// Set the joined date before saving
	user.JoinedAt = time.Now()

	// Insert the document
	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		return model.User{}, err
	}

	// Set the returned ID on the user object
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid
	}

	return user, nil
}

// FindAll retrieves all User documents.
func (r *UserRepository) FindAll(ctx context.Context) ([]model.User, error) {
	var users []model.User

	// Create a context for the find operation (use a short timeout if required, but context from handler is usually sufficient)
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Error finding users: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// Decode all documents into the users slice
	if err = cursor.All(ctx, &users); err != nil {
		log.Printf("Error decoding users: %v", err)
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	// 1. Define the filter
	filter := bson.M{"email": email}

	var user model.User

	// 2. Call FindOne
	// The result is decoded into 'user' struct
	err := r.collection.FindOne(ctx, filter).Decode(&user)

	// 3. Handle specific errors
	if err != nil {
		// if document is not found, return nil and nil error
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		// For any other error (e.g., network, connection)
		return nil, fmt.Errorf("error finding user by email: %w", err)
	}

	return &user, nil
}
