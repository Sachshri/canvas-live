package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user document in MongoDB.
type User struct {
	// primitive.ObjectID is the standard type for MongoDB's _id field.
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username string             `bson:"name" json:"username"`
	Email    string             `bson:"email" json:"email"`
	Password string             `bson:"password" json:"password"`
	JoinedAt time.Time          `bson:"joinedAt" json:"joinedAt"`
}
