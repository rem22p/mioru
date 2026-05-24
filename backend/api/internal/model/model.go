package model

// User stored in SQLite (was Redis)
type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	HashedPW    string `json:"hashed_password"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	AvatarColor string `json:"avatar_color"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
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

// Category represents a product category with tree structure
type Category struct {
	ID        int        `json:"id"`
	ParentID  *int       `json:"parent_id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Criteria  []string   `json:"criteria"`
	SortOrder int        `json:"sort_order"`
	Children  []Category `json:"children,omitempty"`
}

// Product represents a product in the catalog
type Product struct {
	ID           int64          `json:"id"`
	Slug         string         `json:"slug"`
	CategoryID   int64          `json:"category_id"`
	CategoryName string         `json:"category_name"`
	Brand        string         `json:"brand"`
	Name         string         `json:"name"`
	Price        int            `json:"price"`
	Color        string         `json:"color"`
	Model        string         `json:"model"`
	Fit          string         `json:"fit"`
	Material     string         `json:"material"`
	Care         []string       `json:"care"`
	Description  string         `json:"description"`
	XPReward     int            `json:"xp_reward"`
	InStock      bool           `json:"in_stock"`
	Status       string         `json:"status"`
	StockQty     int            `json:"stock_quantity"`
	CreatedBy    string         `json:"created_by"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	Sizes        []string       `json:"sizes"`
	SizeChart    []SizeChartRow `json:"size_chart"`
	Images       []ProductImage `json:"images"`
}

// SizeChartRow represents a row in the product's size chart
type SizeChartRow struct {
	Label      string   `json:"label"`
	Chest      *float64 `json:"chest,omitempty"`
	Waist      *float64 `json:"waist,omitempty"`
	Hips       *float64 `json:"hips,omitempty"`
	Length     *float64 `json:"length,omitempty"`
	FootLength *float64 `json:"foot_length,omitempty"`
	Wrist      *float64 `json:"wrist,omitempty"`
	SortOrder  int      `json:"sort_order"`
}

// ProductImage represents an image associated with a product
type ProductImage struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	SortOrder int    `json:"sort_order"`
}

// ProductFilter defines filtering, sorting, and pagination options for listing products
type ProductFilter struct {
	CategoryID int
	Search     string
	Brand      string
	Sort       string
	Page       int
	PerPage    int
}
