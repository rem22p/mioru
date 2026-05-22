package model

// User stored in Redis
type User struct {
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	HashedPW    string `json:"hashed_password"`
	DisplayName string `json:"display_name"`
	AvatarColor string `json:"avatar_color"`
}

// Note stored individually in Redis
type Note struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Color     string `json:"color"`
	Author    string `json:"author"`
	PosX      int    `json:"position_x"`
	PosY      int    `json:"position_y"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// WebSocket message
type WSMessage struct {
	Action string `json:"action,omitempty"`
	Note   *Note  `json:"note,omitempty"`
	Type   string `json:"type,omitempty"`
	NoteID string `json:"noteId,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
}
