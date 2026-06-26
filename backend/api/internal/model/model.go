package model

import "time"

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
	ID         int        `json:"id"`
	ParentID   *int       `json:"parent_id"`
	Name       string     `json:"name"`
	Slug       string     `json:"slug"`
	Criteria   []string   `json:"criteria"`
	SortOrder  int        `json:"sort_order"`
	CoverImage *string    `json:"cover_image,omitempty"`
	Children   []Category `json:"children,omitempty"`
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
	PriceMin    int    // 0 = no lower bound
	PriceMax    int    // 0 = no upper bound
	Status      string // "in_stock" | "preorder" | "out_of_stock" | "" (all)
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

// Order represents a customer order. total_minor is in minor currency
// units (kopecks/cents). Display as (total_minor / 100) with 2 decimal places.
type Order struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customer_id"`
	Type       string `json:"type"`
	TotalMinor int64  `json:"total_minor"`
	Status     string `json:"status"`
	// Phone is the contact number the customer typed in at
	// checkout. It is captured on every order (cart / individual
	// / preorder), independent of `customers.phone`, so the
	// historical record stays accurate even if the customer
	// later edits their profile.
	Phone          string `json:"phone"`
	City           string `json:"city"`
	DeliveryMethod string `json:"delivery_method"`
	PaymentMethod  string `json:"payment_method"`
	Street         string `json:"street"`
	House          string `json:"house"`
	Apartment      string `json:"apartment"`
	Comment        string `json:"comment"`
	// Individual order fields
	Height       *float64    `json:"height,omitempty"`
	Weight       *float64    `json:"weight,omitempty"`
	DeliveryTime []string    `json:"delivery_time,omitempty"`
	Photos       []string    `json:"photos,omitempty"`
	Items        []OrderItem `json:"items,omitempty"`
	// Joined fields (populated in admin list queries)
	CustomerEmail     string    `json:"customer_email,omitempty"`
	CustomerFirstName string    `json:"customer_first_name,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// OrderItem is a single line item in an order.
type OrderItem struct {
	ID          int64  `json:"id"`
	OrderID     int64  `json:"order_id"`
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name,omitempty"`
	// ProductSlug + ImageURL are populated by the store via JOIN against
	// `products` / `product_images` so the storefront can render a small
	// thumbnail in the order history without a second round-trip.
	ProductSlug string `json:"product_slug,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	SizeLabel   string `json:"size_label"`
	Quantity    int    `json:"quantity"`
	PriceMinor  int64  `json:"price_minor"`
	HeightCm    *int   `json:"height_cm,omitempty"`
	WeightKg    *int   `json:"weight_kg,omitempty"`
}
