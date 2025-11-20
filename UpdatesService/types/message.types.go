package types

type Message struct {
	DocumentID string `json:"documentId"`
	UserID     string `json:"userId"`
	Username   string `json:"username"`
	Type       int    `json:"type"`
	Body       string `json:"body"`
}

// Update Message
type UpdateMessage struct {
	Action            string                 `json:"action"`
	ObjectID          string                 `json:"objectId"`
	SlideID           string                 `json:"slideId"`
	ObjectType        string                 `json:"objectType"`
	UpdatedAttributes map[string]interface{} `json:"updatedAttributes"` // only attributes which have changed
}

// Delete Message
type DeleteMessage struct {
	Action     string `json:"action"`
	ObjectID   string `json:"objectId"`
	SlideID    string `json:"slideId"`
	ObjectType string `json:"objectType"`
}

// Create Message
type CreateMessage struct {
	Action     string                 `json:"action"`
	SlideID    string                 `json:"slideId"`
	ObjectID   string                 `json:"objectId"`
	Type       string                 `json:"objectType"`
	Attributes map[string]interface{} `json:"attributes"`
}

// CursorMove message
type CursorMoveMessage struct {
	Action            string     `json:"action"`
	SlideID           string     `json:"slideId"`
	NewCursorLocation [2]float64 `json:"newCursorLocation"`
}

// Select message
type SelectMessage struct {
	Action   string `json:"action"` // {'select'} // if already selected then deselect
	ObjectID string `json:"objectId"`
	SlideID  string `json:"slideId"`
}

// Add slide
type AddSlide struct {
	Action  string `json:"action"`
	SlideID string `json:"slideId"`
}

// Remove slide
type RemoveSlide struct {
	Action  string `json:"action"`
	SlideID string `json:"slideId"`
}

// ========================================================

type KafkaInterMessage struct {
	Topic   string
	Message Message
}

type ServerResponseMessage struct {
	Success bool `json:"success"` // true for success false for failure
}
