package store

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"

	"mioru/internal/auth"
	"mioru/internal/model"
)

// migrationsFS holds the versioned SQL migrations, embedded so the binary is
// self-contained (no migrations directory needs shipping alongside it).
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// PostgresStore provides PostgreSQL-based persistence for users, products, and categories.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a connection pool and runs migrations.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// Close closes the connection pool.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// Pool returns the underlying *pgxpool.Pool (used by migration handler).
func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	if err := s.runMigrations(ctx); err != nil {
		return err
	}

	// Seed the first admin from BOOTSTRAP_ADMIN_* env. Registration is
	// invite-only (admins create admins), so this resolves the chicken-and-egg
	// for the very first admin on a clean database. It depends on runtime env
	// and bcrypt, so it stays a Go step rather than a versioned SQL migration.
	if err := s.seedAdmin(ctx); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}

	return nil
}

// runMigrations applies every embedded versioned migration up to the latest via
// tern. The applied version is tracked in public.schema_version (created on
// first run) and each migration runs in its own transaction, so a restart or a
// deploy onto an already-migrated database is a safe no-op. tern operates on a
// single *pgx.Conn, so we borrow one from the pool for the duration.
func (s *PostgresStore) runMigrations(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration conn: %w", err)
	}
	defer conn.Release()

	m, err := migrate.NewMigrator(ctx, conn.Conn(), "public.schema_version")
	if err != nil {
		return fmt.Errorf("new migrator: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("sub migrations fs: %w", err)
	}
	if err := m.LoadMigrations(sub); err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// seedAdmin inserts the bootstrap admin defined by BOOTSTRAP_ADMIN_USERNAME,
// BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD. It is a no-op when any of
// those vars is unset, and idempotent via ON CONFLICT DO NOTHING so repeated
// boots never duplicate or overwrite an existing admin.
func (s *PostgresStore) seedAdmin(ctx context.Context) error {
	username := os.Getenv("BOOTSTRAP_ADMIN_USERNAME")
	emailAddr := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if username == "" || emailAddr == "" || password == "" {
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		INSERT INTO users (username, email, hashed_password, display_name, avatar_color, role)
		VALUES ($1, $2, $3, $1, '#44944A', 'admin')
		ON CONFLICT DO NOTHING`,
		username, strings.ToLower(emailAddr), hash,
	)
	if err != nil {
		return fmt.Errorf("insert bootstrap admin: %w", err)
	}
	if tag.RowsAffected() > 0 {
		log.Printf("Seeded bootstrap admin %q from BOOTSTRAP_ADMIN_* env", username)
	}
	return nil
}

// ── Categories ──

// GetCategories returns all categories organized as a tree.
func (s *PostgresStore) GetCategories(ctx context.Context) ([]model.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, parent_id, name, slug, criteria, sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var all []model.Category
	for rows.Next() {
		var c model.Category
		var parentID *int
		var criteriaJSON string
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &criteriaJSON, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.ParentID = parentID
		json.Unmarshal([]byte(criteriaJSON), &c.Criteria)
		if c.Criteria == nil {
			c.Criteria = []string{}
		}
		all = append(all, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return buildCategoryTree(all), nil
}

// GetCategoriesFlat returns all categories as a flat list with ParentID references.
func (s *PostgresStore) GetCategoriesFlat(ctx context.Context) ([]model.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, parent_id, name, slug, criteria, sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var all []model.Category
	for rows.Next() {
		var c model.Category
		var parentID *int
		var criteriaJSON string
		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &criteriaJSON, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.ParentID = parentID
		json.Unmarshal([]byte(criteriaJSON), &c.Criteria)
		if c.Criteria == nil {
			c.Criteria = []string{}
		}
		all = append(all, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return all, nil
}

func buildCategoryTree(cats []model.Category) []model.Category {
	byID := make(map[int]*model.Category)
	for i := range cats {
		cats[i].Children = []model.Category{}
		byID[cats[i].ID] = &cats[i]
	}

	var roots []*model.Category
	for i := range cats {
		c := &cats[i]
		if c.ParentID == nil {
			roots = append(roots, c)
		} else if parent, ok := byID[*c.ParentID]; ok {
			parent.Children = append(parent.Children, *c)
		}
	}

	result := make([]model.Category, len(roots))
	for i, r := range roots {
		cleanEmptyChildren(r)
		result[i] = *r
	}
	return result
}

func cleanEmptyChildren(c *model.Category) {
	for i := range c.Children {
		cleanEmptyChildren(&c.Children[i])
	}
	if len(c.Children) == 0 {
		c.Children = nil
	}
}

// ── Products ──

// CreateProduct inserts a new product with sizes, size chart, and images in a transaction.
func (s *PostgresStore) CreateProduct(ctx context.Context, p model.Product) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	careJSON, _ := json.Marshal(p.Care)
	inStock := int16(0)
	if p.InStock {
		inStock = 1
	}

	var productID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO products (slug, category_id, brand, name, price, color, model, fit, material, care, description, xp_reward, in_stock, status, stock_quantity, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id`,
		p.Slug, p.CategoryID, p.Brand, p.Name, p.Price, p.Color, p.Model, p.Fit, p.Material, string(careJSON), p.Description, p.XPReward, inStock, p.Status, p.StockQty, p.CreatedBy,
	).Scan(&productID)
	if err != nil {
		return 0, fmt.Errorf("insert product: %w", err)
	}

	// Insert sizes
	for _, size := range p.Sizes {
		if _, err := tx.Exec(ctx, `INSERT INTO product_sizes (product_id, size_label) VALUES ($1, $2)`, productID, size); err != nil {
			return 0, fmt.Errorf("insert size: %w", err)
		}
	}

	// Insert size chart rows
	for i, row := range p.SizeChart {
		if _, err := tx.Exec(ctx, `
			INSERT INTO size_chart_rows (product_id, label, chest, waist, hips, length, foot_length, wrist, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			productID, row.Label, row.Chest, row.Waist, row.Hips, row.Length, row.FootLength, row.Wrist, i,
		); err != nil {
			return 0, fmt.Errorf("insert size chart: %w", err)
		}
	}

	// Insert images
	for i, img := range p.Images {
		if _, err := tx.Exec(ctx, `INSERT INTO product_images (product_id, url, sort_order) VALUES ($1, $2, $3)`, productID, img.URL, i); err != nil {
			return 0, fmt.Errorf("insert image: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return productID, nil
}

// UpdateProduct updates a product and replaces its sizes, size chart rows, and images in a transaction.
func (s *PostgresStore) UpdateProduct(ctx context.Context, slug string, p model.Product) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get product ID
	var productID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM products WHERE slug = $1`, slug).Scan(&productID); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return fmt.Errorf("product not found: %s", slug)
		}
		return fmt.Errorf("find product: %w", err)
	}

	careJSON, _ := json.Marshal(p.Care)
	inStock := int16(0)
	if p.InStock {
		inStock = 1
	}

	_, err = tx.Exec(ctx, `
		UPDATE products SET slug=$1, category_id=$2, brand=$3, name=$4, price=$5, color=$6, model=$7, fit=$8, material=$9, care=$10, description=$11, xp_reward=$12, in_stock=$13, status=$14, stock_quantity=$15, updated_at=NOW()
		WHERE id=$16`,
		p.Slug, p.CategoryID, p.Brand, p.Name, p.Price, p.Color, p.Model, p.Fit, p.Material, string(careJSON), p.Description, p.XPReward, inStock, p.Status, p.StockQty, productID,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}

	// Delete existing children
	if _, err := tx.Exec(ctx, `DELETE FROM product_sizes WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("delete sizes: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM size_chart_rows WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("delete size chart: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM product_images WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("delete images: %w", err)
	}

	// Re-insert sizes
	for _, size := range p.Sizes {
		if _, err := tx.Exec(ctx, `INSERT INTO product_sizes (product_id, size_label) VALUES ($1, $2)`, productID, size); err != nil {
			return fmt.Errorf("insert size: %w", err)
		}
	}

	// Re-insert size chart rows
	for i, row := range p.SizeChart {
		if _, err := tx.Exec(ctx, `
			INSERT INTO size_chart_rows (product_id, label, chest, waist, hips, length, foot_length, wrist, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			productID, row.Label, row.Chest, row.Waist, row.Hips, row.Length, row.FootLength, row.Wrist, i,
		); err != nil {
			return fmt.Errorf("insert size chart: %w", err)
		}
	}

	// Re-insert images
	for i, img := range p.Images {
		if _, err := tx.Exec(ctx, `INSERT INTO product_images (product_id, url, sort_order) VALUES ($1, $2, $3)`, productID, img.URL, i); err != nil {
			return fmt.Errorf("insert image: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// DeleteProduct deletes a product by slug. CASCADE handles related rows.
func (s *PostgresStore) DeleteProduct(ctx context.Context, slug string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM products WHERE slug = $1`, slug)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found: %s", slug)
	}
	return nil
}

// GetProduct retrieves a single product with all related data.
func (s *PostgresStore) GetProduct(ctx context.Context, slug string) (*model.Product, error) {
	p, err := s.queryProduct(ctx, `WHERE p.slug = $1`, slug)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("product not found: %s", slug)
	}
	return p, nil
}

// ListProducts retrieves products with filtering, sorting, and pagination.
func (s *PostgresStore) ListProducts(ctx context.Context, filter model.ProductFilter) ([]model.Product, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.CategoryID > 0 {
		where += fmt.Sprintf(" AND p.category_id = $%d", argIdx)
		args = append(args, filter.CategoryID)
		argIdx++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND (p.name LIKE $%d OR p.brand LIKE $%d OR p.slug LIKE $%d)", argIdx, argIdx+1, argIdx+2)
		s := "%" + filter.Search + "%"
		args = append(args, s, s, s)
		argIdx += 3
	}
	if filter.Brand != "" {
		where += fmt.Sprintf(" AND p.brand = $%d", argIdx)
		args = append(args, filter.Brand)
		argIdx++
	}

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM products p ` + where
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	// Sort
	order := "p.created_at DESC"
	if filter.Sort != "" {
		col := filter.Sort
		desc := false
		if strings.HasPrefix(col, "-") {
			desc = true
			col = col[1:]
		}
		allowedCols := map[string]bool{"created_at": true, "price": true, "name": true, "brand": true}
		if allowedCols[col] {
			dir := "ASC"
			if desc {
				dir = "DESC"
			}
			order = "p." + col + " " + dir
		}
	}

	// Pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	query := `SELECT p.id FROM products p ` + where + ` ORDER BY ` + order + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration: %w", err)
	}

	// Fetch the full products for this page in a fixed number of queries
	// (instead of one-per-id), preserving the paginated order above.
	products, err := s.listProductsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// listProductsByIDs loads full products for the given ids using a fixed number
// of queries regardless of len(ids): one for the products, plus one each for
// sizes, size charts, and images (batched via = ANY($1)). Results are returned
// in the same order as ids; ids not found are skipped.
func (s *PostgresStore) listProductsByIDs(ctx context.Context, ids []int64) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}

	rows, err := s.pool.Query(ctx, productSelectBase+`WHERE p.id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	byID := make(map[int64]*model.Product, len(ids))
	for rows.Next() {
		var p model.Product
		var careJSON string
		var inStock int16
		if err := rows.Scan(
			&p.ID, &p.Slug, &p.CategoryID, &p.CategoryName,
			&p.Brand, &p.Name, &p.Price, &p.Color, &p.Model, &p.Fit, &p.Material, &careJSON,
			&p.Description, &p.XPReward, &inStock, &p.Status, &p.StockQty, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		p.InStock = inStock == 1
		json.Unmarshal([]byte(careJSON), &p.Care)
		if p.Care == nil {
			p.Care = []string{}
		}
		// Initialise related slices so absent rows serialise as [] not null.
		p.Sizes = []string{}
		p.SizeChart = []model.SizeChartRow{}
		p.Images = []model.ProductImage{}
		byID[p.ID] = &p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if err := s.attachSizes(ctx, byID, ids); err != nil {
		return nil, err
	}
	if err := s.attachSizeCharts(ctx, byID, ids); err != nil {
		return nil, err
	}
	if err := s.attachImages(ctx, byID, ids); err != nil {
		return nil, err
	}

	out := make([]model.Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, *p)
		}
	}
	return out, nil
}

// attachSizes loads sizes for all products in one query and appends them to the
// matching product. Ordered by (product_id, id) so each product's sizes keep
// their insertion order.
func (s *PostgresStore) attachSizes(ctx context.Context, byID map[int64]*model.Product, ids []int64) error {
	rows, err := s.pool.Query(ctx, `SELECT product_id, size_label FROM product_sizes WHERE product_id = ANY($1) ORDER BY product_id, id`, ids)
	if err != nil {
		return fmt.Errorf("query sizes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var label string
		if err := rows.Scan(&pid, &label); err != nil {
			return fmt.Errorf("scan size: %w", err)
		}
		if p, ok := byID[pid]; ok {
			p.Sizes = append(p.Sizes, label)
		}
	}
	return rows.Err()
}

// attachSizeCharts loads size chart rows for all products in one query and
// appends them to the matching product, ordered by (sort_order, id) within each.
func (s *PostgresStore) attachSizeCharts(ctx context.Context, byID map[int64]*model.Product, ids []int64) error {
	rows, err := s.pool.Query(ctx, `SELECT product_id, label, chest, waist, hips, length, foot_length, wrist, sort_order FROM size_chart_rows WHERE product_id = ANY($1) ORDER BY product_id, sort_order, id`, ids)
	if err != nil {
		return fmt.Errorf("query size charts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var r model.SizeChartRow
		if err := rows.Scan(&pid, &r.Label, &r.Chest, &r.Waist, &r.Hips, &r.Length, &r.FootLength, &r.Wrist, &r.SortOrder); err != nil {
			return fmt.Errorf("scan chart row: %w", err)
		}
		if p, ok := byID[pid]; ok {
			p.SizeChart = append(p.SizeChart, r)
		}
	}
	return rows.Err()
}

// attachImages loads images for all products in one query and appends them to
// the matching product, ordered by (sort_order, id) within each.
func (s *PostgresStore) attachImages(ctx context.Context, byID map[int64]*model.Product, ids []int64) error {
	rows, err := s.pool.Query(ctx, `SELECT product_id, id, url, sort_order FROM product_images WHERE product_id = ANY($1) ORDER BY product_id, sort_order, id`, ids)
	if err != nil {
		return fmt.Errorf("query images: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var img model.ProductImage
		if err := rows.Scan(&pid, &img.ID, &img.URL, &img.SortOrder); err != nil {
			return fmt.Errorf("scan image: %w", err)
		}
		if p, ok := byID[pid]; ok {
			p.Images = append(p.Images, img)
		}
	}
	return rows.Err()
}

// productSelectBase is the shared SELECT (with category name joined) for reading
// products. Append a WHERE clause to it. Column order matches scanProduct.
const productSelectBase = `SELECT p.id, p.slug, p.category_id, COALESCE(c.name, '') as category_name,
	p.brand, p.name, p.price, p.color, p.model, p.fit, p.material, p.care,
	p.description, p.xp_reward, p.in_stock, p.status, p.stock_quantity, p.created_by,
	COALESCE(p.created_at::text, '') as created_at, COALESCE(p.updated_at::text, '') as updated_at
	FROM products p
	LEFT JOIN categories c ON c.id = p.category_id `

// queryProduct is a helper that fetches a full product with all related data.
func (s *PostgresStore) queryProduct(ctx context.Context, whereClause string, arg interface{}) (*model.Product, error) {
	query := productSelectBase + whereClause

	var p model.Product
	var careJSON string
	var inStock int16
	if err := s.pool.QueryRow(ctx, query, arg).Scan(
		&p.ID, &p.Slug, &p.CategoryID, &p.CategoryName,
		&p.Brand, &p.Name, &p.Price, &p.Color, &p.Model, &p.Fit, &p.Material, &careJSON,
		&p.Description, &p.XPReward, &inStock, &p.Status, &p.StockQty, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, fmt.Errorf("query product: %w", err)
	}

	p.InStock = inStock == 1
	json.Unmarshal([]byte(careJSON), &p.Care)
	if p.Care == nil {
		p.Care = []string{}
	}

	// Load sizes
	sizes, err := s.getProductSizes(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Sizes = sizes

	// Load size chart
	chart, err := s.getProductSizeChart(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.SizeChart = chart

	// Load images
	images, err := s.getProductImages(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Images = images

	return &p, nil
}

func (s *PostgresStore) getProductSizes(ctx context.Context, productID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT size_label FROM product_sizes WHERE product_id = $1 ORDER BY id`, productID)
	if err != nil {
		return nil, fmt.Errorf("query sizes: %w", err)
	}
	defer rows.Close()

	var sizes []string
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return nil, fmt.Errorf("scan size: %w", err)
		}
		sizes = append(sizes, label)
	}
	if sizes == nil {
		sizes = []string{}
	}
	return sizes, rows.Err()
}

func (s *PostgresStore) getProductSizeChart(ctx context.Context, productID int64) ([]model.SizeChartRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT label, chest, waist, hips, length, foot_length, wrist, sort_order FROM size_chart_rows WHERE product_id = $1 ORDER BY sort_order, id`, productID)
	if err != nil {
		return nil, fmt.Errorf("query size chart: %w", err)
	}
	defer rows.Close()

	var chart []model.SizeChartRow
	for rows.Next() {
		var r model.SizeChartRow
		if err := rows.Scan(&r.Label, &r.Chest, &r.Waist, &r.Hips, &r.Length, &r.FootLength, &r.Wrist, &r.SortOrder); err != nil {
			return nil, fmt.Errorf("scan chart row: %w", err)
		}
		chart = append(chart, r)
	}
	if chart == nil {
		chart = []model.SizeChartRow{}
	}
	return chart, rows.Err()
}

func (s *PostgresStore) getProductImages(ctx context.Context, productID int64) ([]model.ProductImage, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, url, sort_order FROM product_images WHERE product_id = $1 ORDER BY sort_order, id`, productID)
	if err != nil {
		return nil, fmt.Errorf("query images: %w", err)
	}
	defer rows.Close()

	var images []model.ProductImage
	for rows.Next() {
		var img model.ProductImage
		if err := rows.Scan(&img.ID, &img.URL, &img.SortOrder); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}
		images = append(images, img)
	}
	if images == nil {
		images = []model.ProductImage{}
	}
	return images, rows.Err()
}
