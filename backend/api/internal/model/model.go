package model

// User is an admin / staff account.
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
	CategoryID  int   // legacy single-value filter; combined with CategoryIDs via OR
	CategoryIDs []int // multi-value, lets the storefront fold a parent category + its children into one query
	Search      string
	Brand       string // legacy single-value filter; combined with Brands via OR
	Brands      []string
	Colors      []string
	Sizes       []string
	PriceMin    int // 0 = no lower bound
	PriceMax    int // 0 = no upper bound
	Sort        string
	Page        int
	PerPage     int
}

// ProductFacets enumerates the distinct values available for filtering within
// a given scope (typically a category). Returned by storeReader.ListProductFacets
// so the storefront filter UI shows only what actually matches.
type ProductFacets struct {
	Brands []string `json:"brands"`
	Colors []string `json:"colors"`
	Sizes  []string `json:"sizes"`
}

// Customer represents a store customer (separate from admin users).
// After 004_oauth.sql, Email and HashedPW may be empty for OAuth-only customers.
type Customer struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	HashedPW    string `json:"hashed_password"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Phone       string `json:"phone"`
	AvatarColor string `json:"avatar_color"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// CustomerOAuth records an OAuth provider link for a store customer.
type CustomerOAuth struct {
	ID          int64  `json:"id"`
	CustomerID  int64  `json:"customer_id"`
	Provider    string `json:"provider"`
	OAuthID     string `json:"oauth_id"`
	ProfileData string `json:"profile_data"`
	CreatedAt   string `json:"created_at"`
}

// Order represents a customer order. Minimal for order history display.
type Order struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customer_id"`
	Total      int    `json:"total"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}
