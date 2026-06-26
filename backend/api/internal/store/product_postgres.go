package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mioru/internal/model"
)

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
	for _, sz := range p.Sizes {
		if _, err := tx.Exec(ctx, `INSERT INTO product_sizes (product_id, size_label, stock_quantity) VALUES ($1, $2, $3)`, productID, sz.Label, sz.StockQuantity); err != nil {
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
	for _, sz := range p.Sizes {
		if _, err := tx.Exec(ctx, `INSERT INTO product_sizes (product_id, size_label, stock_quantity) VALUES ($1, $2, $3)`, productID, sz.Label, sz.StockQuantity); err != nil {
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
	where, args, argIdx := buildProductFilterWhere(filter, 1)

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
		p.Sizes = []model.ProductSize{}
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
	rows, err := s.pool.Query(ctx, `SELECT product_id, size_label, COALESCE(stock_quantity, 0) FROM product_sizes WHERE product_id = ANY($1)
		ORDER BY product_id,
			CASE WHEN size_label ~ '^[0-9]' THEN 0 ELSE 1 END,
			CASE WHEN size_label ~ '^[0-9]' THEN NULLIF(regexp_replace(size_label, '[^0-9.]', '', 'g'), '')::numeric END,
			size_label`, ids)
	if err != nil {
		return fmt.Errorf("query sizes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var sz model.ProductSize
		if err := rows.Scan(&pid, &sz.Label, &sz.StockQuantity); err != nil {
			return fmt.Errorf("scan size: %w", err)
		}
		if p, ok := byID[pid]; ok {
			p.Sizes = append(p.Sizes, sz)
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

func (s *PostgresStore) getProductSizes(ctx context.Context, productID int64) ([]model.ProductSize, error) {
	rows, err := s.pool.Query(ctx, `SELECT size_label, COALESCE(stock_quantity, 0) FROM product_sizes WHERE product_id = $1
		ORDER BY
			CASE WHEN size_label ~ '^[0-9]' THEN 0 ELSE 1 END,
			CASE WHEN size_label ~ '^[0-9]' THEN NULLIF(regexp_replace(size_label, '[^0-9.]', '', 'g'), '')::numeric END,
			size_label`, productID)
	if err != nil {
		return nil, fmt.Errorf("query sizes: %w", err)
	}
	defer rows.Close()

	var sizes []model.ProductSize
	for rows.Next() {
		var s model.ProductSize
		if err := rows.Scan(&s.Label, &s.StockQuantity); err != nil {
			return nil, fmt.Errorf("scan size: %w", err)
		}
		sizes = append(sizes, s)
	}
	if sizes == nil {
		sizes = []model.ProductSize{}
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

// buildProductFilterWhere composes the SQL WHERE clause (with leading "WHERE 1=1")
// and matching args list for the storefront product filters. It returns the next
// available positional placeholder index so the caller can append LIMIT/OFFSET or
// other clauses without colliding. Used by both ListProducts and ListProductFacets
// so the facet counts always match the result set the user is about to see.
func buildProductFilterWhere(filter model.ProductFilter, startIdx int) (where string, args []interface{}, nextIdx int) {
	where = "WHERE 1=1"
	argIdx := startIdx

	cats := filter.CategoryIDs
	if filter.CategoryID > 0 {
		cats = append(cats, filter.CategoryID)
	}
	if len(cats) > 0 {
		where += fmt.Sprintf(" AND p.category_id = ANY($%d::int[])", argIdx)
		args = append(args, cats)
		argIdx++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.brand ILIKE $%d OR p.slug ILIKE $%d)", argIdx, argIdx+1, argIdx+2)
		s := "%" + filter.Search + "%"
		args = append(args, s, s, s)
		argIdx += 3
	}
	// Brand (legacy single) + Brands (multi) collapse to a single ANY clause so
	// the storefront can pass either without surprise.
	brands := filter.Brands
	if filter.Brand != "" {
		brands = append(brands, filter.Brand)
	}
	if len(brands) > 0 {
		where += fmt.Sprintf(" AND p.brand = ANY($%d::text[])", argIdx)
		args = append(args, brands)
		argIdx++
	}
	if len(filter.Colors) > 0 {
		where += fmt.Sprintf(" AND p.color = ANY($%d::text[])", argIdx)
		args = append(args, filter.Colors)
		argIdx++
	}
	if len(filter.Sizes) > 0 {
		where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM product_sizes ps WHERE ps.product_id = p.id AND ps.size_label = ANY($%d::text[]))", argIdx)
		args = append(args, filter.Sizes)
		argIdx++
	}
	if filter.PriceMin > 0 {
		where += fmt.Sprintf(" AND p.price >= $%d", argIdx)
		args = append(args, filter.PriceMin)
		argIdx++
	}
	if filter.PriceMax > 0 {
		where += fmt.Sprintf(" AND p.price <= $%d", argIdx)
		args = append(args, filter.PriceMax)
		argIdx++
	}
	if filter.Status != "" {
		where += fmt.Sprintf(" AND p.status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	return where, args, argIdx
}

// ListProductFacets returns the distinct brand/color/size values present in the
// product set matching the given filter (sans facet-specific selections —
// callers should pass the filter *without* Brands/Colors/Sizes so each facet's
// option list stays useful even after the user picks one). Used by the
// storefront filter UI so it shows only options that will actually return
// results within the active category/search scope.
func (s *PostgresStore) ListProductFacets(ctx context.Context, filter model.ProductFilter) (model.ProductFacets, error) {
	where, args, _ := buildProductFilterWhere(filter, 1)

	facets := model.ProductFacets{
		Brands: []string{},
		Colors: []string{},
		Sizes:  []string{},
	}

	brandQ := `SELECT DISTINCT p.brand FROM products p ` + where + ` AND p.brand <> '' ORDER BY p.brand`
	rows, err := s.pool.Query(ctx, brandQ, args...)
	if err != nil {
		return facets, fmt.Errorf("query brand facets: %w", err)
	}
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			rows.Close()
			return facets, fmt.Errorf("scan brand facet: %w", err)
		}
		facets.Brands = append(facets.Brands, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return facets, fmt.Errorf("brand facets iteration: %w", err)
	}

	colorQ := `SELECT DISTINCT p.color FROM products p ` + where + ` AND p.color <> '' ORDER BY p.color`
	rows, err = s.pool.Query(ctx, colorQ, args...)
	if err != nil {
		return facets, fmt.Errorf("query color facets: %w", err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			rows.Close()
			return facets, fmt.Errorf("scan color facet: %w", err)
		}
		facets.Colors = append(facets.Colors, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return facets, fmt.Errorf("color facets iteration: %w", err)
	}

	sizeQ := `SELECT DISTINCT ps.size_label FROM product_sizes ps JOIN products p ON p.id = ps.product_id ` + where + ` ORDER BY
		CASE WHEN ps.size_label ~ '^[0-9]' THEN 0 ELSE 1 END,
		CASE WHEN ps.size_label ~ '^[0-9]' THEN NULLIF(regexp_replace(ps.size_label, '[^0-9.]', '', 'g'), '')::numeric END,
		ps.size_label`
	rows, err = s.pool.Query(ctx, sizeQ, args...)
	if err != nil {
		return facets, fmt.Errorf("query size facets: %w", err)
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return facets, fmt.Errorf("scan size facet: %w", err)
		}
		facets.Sizes = append(facets.Sizes, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return facets, fmt.Errorf("size facets iteration: %w", err)
	}

	return facets, nil
}
