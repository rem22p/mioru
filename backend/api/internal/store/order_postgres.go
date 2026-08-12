package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"mioru/internal/model"
)

// IdempotencyRecord is the row stored in order_idempotency. The handler
// fetches it on the race-loser path to return the winner's response
// when the request hash matches. Status is the HTTP status (typically
// 201), ResponseBody is the marshalled order that the winner saw.
type IdempotencyRecord struct {
	Key          string
	CustomerID   int64
	OrderID      int64
	RequestHash  string
	Status       int
	ResponseBody string
}

// validOrderStatuses is the set of allowed order status values.
var validOrderStatuses = map[string]bool{
	"pending":    true,
	"processing": true,
	"shipped":    true,
	"delivered":  true,
	"cancelled":  true,
}

// Clock returns the current time. In production this is time.Now; tests inject a
// fixed clock so idempotency TTL and order timestamps are deterministic.
type Clock func() time.Time

// ListCustomerOrders returns paginated orders for a customer, newest first.
func (s *PostgresStore) ListCustomerOrders(ctx context.Context, customerID int64, page, perPage int) ([]model.Order, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	var total int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE customer_id = $1`, customerID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count customer orders: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx, `
		SELECT id, order_code, customer_id, type, total_minor, status,
		       phone,
		       city, delivery_method, payment_method,
		       street, house, apartment, comment,
		       height, weight, delivery_time, photos,
		       created_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3`, customerID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list customer orders: %w", err)
	}
	defer rows.Close()

	// Empty slice (not nil) so JSON serializes as [] — storefront/profile
	// and admin do len(orders) and crash on null.
	var orders = []model.Order{}
	var orderIDs []int64
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.OrderCode, &o.CustomerID, &o.Type, &o.TotalMinor, &o.Status,
			&o.Phone,
			&o.City, &o.DeliveryMethod, &o.PaymentMethod,
			&o.Street, &o.House, &o.Apartment, &o.Comment,
			&o.Height, &o.Weight, &o.DeliveryTime, &o.Photos,
			&o.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}

	// Batch load items with product names
	if len(orderIDs) > 0 {
		itemsMap, err := s.loadOrderItems(ctx, orderIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("load order items: %w", err)
		}
		for i := range orders {
			orders[i].Items = itemsMap[orders[i].ID]
			if orders[i].Items == nil {
				orders[i].Items = []model.OrderItem{}
			}
		}
	}

	return orders, total, nil
}

// GetProductPriceMap returns a map of product_id → price_minor for the given IDs.
// Products not found are omitted from the map (caller should validate).
func (s *PostgresStore) GetProductPriceMap(ctx context.Context, productIDs []int64) (map[int64]int64, error) {
	if len(productIDs) == 0 {
		return map[int64]int64{}, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, price FROM products WHERE id = ANY($1)`, productIDs)
	if err != nil {
		return nil, fmt.Errorf("get product prices: %w", err)
	}
	defer rows.Close()

	m := make(map[int64]int64, len(productIDs))
	for rows.Next() {
		var id int64
		var price int
		if err := rows.Scan(&id, &price); err != nil {
			return nil, fmt.Errorf("scan product price: %w", err)
		}
		m[id] = int64(price) * 100 // convert MDL to minor (kopecks)
	}
	return m, rows.Err()
}

// CreateOrder inserts an order with its line items, stock decrement, and
// idempotency guard inside a single transaction.
//
// Price recalculation: prices are loaded from the DB by product_id and MUST
// match the client-supplied items; client price_minor is ignored. total_minor
// is recalculated server-side from DB prices × quantity.
//
// Stock: each item's quantity is atomically decremented via
// UPDATE … SET stock_quantity = stock_quantity - $n WHERE stock_quantity >= $n.
// If any item is out of stock the transaction rolls back.
//
// Idempotency: SELECT-first on (key, user_id). Same key + same hash → return
// the stored order. Same key + different hash → ErrIdempotencyHashMismatch.
func (s *PostgresStore) CreateOrder(ctx context.Context, customerID int64, o *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	// ── Step 0: SELECT-first idempotency check ──
	var storedOrderID int64
	var storedHash string
	var storedStatus int
	var storedBody string
	err := s.pool.QueryRow(ctx,
		`SELECT order_id, request_hash, status, response_body
		 FROM order_idempotency WHERE key = $1 AND user_id = $2`,
		idempotencyKey, customerID,
	).Scan(&storedOrderID, &storedHash, &storedStatus, &storedBody)
	if err == nil {
		// Key exists — check hash
		if storedHash == requestHash {
			// Replay — return the stored response
			var stored model.Order
			if err := json.Unmarshal([]byte(storedBody), &stored); err != nil {
				return nil, fmt.Errorf("unmarshal stored idempotent response: %w", err)
			}
			return &stored, nil
		}
		return nil, ErrIdempotencyHashMismatch
	}
	// If not found, proceed. Any other error is an infrastructure problem.
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}

	// ── Step 1: Load prices from DB, recalculate ──
	prodIDs := make([]int64, 0, len(items))
	for _, it := range items {
		prodIDs = append(prodIDs, it.ProductID)
	}
	prices, err := s.GetProductPriceMap(ctx, prodIDs)
	if err != nil {
		return nil, fmt.Errorf("load prices: %w", err)
	}

	var totalMinor int64
	for i := range items {
		dbPrice, ok := prices[items[i].ProductID]
		if !ok {
			return nil, fmt.Errorf("product %d: %w", items[i].ProductID, ErrProductNotFound)
		}
		items[i].PriceMinor = dbPrice
		totalMinor += dbPrice * int64(items[i].Quantity)
	}
	o.TotalMinor = totalMinor

	// ── Step 2: Transaction — order + items + stock + idempotency ──
	//
	// The retry loop handles the rare case where two concurrent
	// CreateOrder calls generate the same order_code.  The partial
	// unique index (migration 019) guarantees at most one row wins;
	// the loser gets a 23505 (duplicate key) and we regenerate +
	// retry the whole transaction.  Boundary: max 10 attempts.
	const maxCodeAttempts = 10
	for codeAttempt := 0; codeAttempt < maxCodeAttempts; codeAttempt++ {
		orderCode, err := s.generateOrderCode(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("generate order code: %w", err)
		}

		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, fmt.Errorf("begin tx: %w", err)
		}

		// Clean expired idempotency records
		_, _ = tx.Exec(ctx, `DELETE FROM order_idempotency WHERE expires_at < NOW()`)

		// Default nil slices to empty arrays (NOT NULL columns)
		if o.DeliveryTime == nil {
			o.DeliveryTime = []string{}
		}
		if o.Photos == nil {
			o.Photos = []string{}
		}

		// Insert order
		err = tx.QueryRow(ctx, `
			INSERT INTO orders (customer_id, type, total_minor, status,
			                    order_code,
			                    phone,
			                    city, delivery_method, payment_method,
			                    street, house, apartment, comment,
			                    height, weight, delivery_time, photos)
			VALUES ($1, $2, $3, 'pending',
			        $4,
			        $5,
			        $6, $7, $8,
			        $9, $10, $11, $12,
			        $13, $14, $15, $16)
			RETURNING id, created_at`,
			customerID, o.Type, o.TotalMinor,
			orderCode,
			o.Phone,
			o.City, o.DeliveryMethod, o.PaymentMethod,
			o.Street, o.House, o.Apartment, o.Comment,
			o.Height, o.Weight, o.DeliveryTime, o.Photos,
		).Scan(&o.ID, &o.CreatedAt)

		// Duplicate order_code — rollback, regenerate, retry.
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
				pgErr.ConstraintName == "orders_order_code_unique" {
				tx.Rollback(ctx)
				continue
			}
			tx.Rollback(ctx)
			return nil, fmt.Errorf("insert order: %w", err)
		}

		o.OrderCode = orderCode
		o.CustomerID = customerID
		o.Status = "pending"

		// Insert line items. We also denormalise product_name
		// and product_slug from the products table into
		// order_items (see migration 014) so the Telegram
		// "new order" notification can render a clickable
		// "<a href=\"...\">Product Name</a>" link even if the
		// product gets renamed or deleted later. The lookup is
		// batched once before the loop (not N+1) and a missing
		// product causes the transaction to roll back (the FK
		// constraint on order_items.product_id is the second
		// line of defence). Once the order is committed, the
		// denormalised name/slug survive any future product
		// change.
		productIDs := make([]int64, 0, len(items))
		seen := make(map[int64]bool, len(items))
		for _, it := range items {
			if !seen[it.ProductID] {
				productIDs = append(productIDs, it.ProductID)
				seen[it.ProductID] = true
			}
		}
		productMeta := make(map[int64]struct{ Name, Slug, Status string }, len(productIDs))
		rows, err := tx.Query(ctx,
			`SELECT id, name, slug, status FROM products WHERE id = ANY($1)`, productIDs)
		if err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("batch lookup products: %w", err)
		}
		for rows.Next() {
			var id int64
			var name, slug, status string
			if err := rows.Scan(&id, &name, &slug, &status); err != nil {
				rows.Close()
				tx.Rollback(ctx)
				return nil, fmt.Errorf("scan product meta: %w", err)
			}
			productMeta[id] = struct{ Name, Slug, Status string }{name, slug, status}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("iterate product meta: %w", err)
		}

		for i := range items {
			items[i].OrderID = o.ID
			meta, ok := productMeta[items[i].ProductID]
			if !ok {
				tx.Rollback(ctx)
				return nil, fmt.Errorf("product %d not found during order insert", items[i].ProductID)
			}
			pname, pslug := meta.Name, meta.Slug
			items[i].ProductName = pname
			items[i].ProductSlug = pslug
			var measurementsJSON []byte
			if len(items[i].Measurements) > 0 {
				measurementsJSON, err = json.Marshal(items[i].Measurements)
				if err != nil {
					tx.Rollback(ctx)
					return nil, fmt.Errorf("marshal measurements: %w", err)
				}
			}
			err = tx.QueryRow(ctx, `
				INSERT INTO order_items (order_id, product_id, product_name, product_slug, size_label, quantity, price_minor, measurements)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING id`,
				o.ID, items[i].ProductID, pname, pslug, items[i].SizeLabel, items[i].Quantity, items[i].PriceMinor,
				measurementsJSON,
			).Scan(&items[i].ID)
			if err != nil {
				tx.Rollback(ctx)
				return nil, fmt.Errorf("insert order item: %w", err)
			}
		}
		o.Items = items

		// Decrement per-size stock atomically — oversell fails the transaction.
		// Preorder items skip stock check entirely (they are made to order).
		for _, it := range items {
			meta := productMeta[it.ProductID]
			if meta.Status == "preorder" {
				continue
			}
			tag, err := tx.Exec(ctx,
				`UPDATE product_sizes SET stock_quantity = stock_quantity - $1
				 WHERE product_id = $2 AND size_label = $3 AND stock_quantity >= $1`,
				it.Quantity, it.ProductID, it.SizeLabel,
			)
			if err != nil {
				tx.Rollback(ctx)
				return nil, fmt.Errorf("decrement stock for product %d size %s: %w", it.ProductID, it.SizeLabel, err)
			}
			if tag.RowsAffected() == 0 {
				tx.Rollback(ctx)
				return nil, fmt.Errorf("product %d size %s: %w (requested %d)", it.ProductID, it.SizeLabel, ErrInsufficientStock, it.Quantity)
			}
		}

		// Refresh products.stock_quantity from per-size totals so
		// admin badges and cover-image ordering stay accurate.
		for _, pid := range productIDs {
			if _, err := tx.Exec(ctx,
				`UPDATE products SET stock_quantity = (
					SELECT COALESCE(SUM(stock_quantity), 0)
					FROM product_sizes
					WHERE product_id = $1
				) WHERE id = $1`, pid); err != nil {
				tx.Rollback(ctx)
				return nil, fmt.Errorf("refresh product stock: %w", err)
			}
		}

		// Store idempotency record
		respBody, err := json.Marshal(o)
		if err != nil {
			respBody = []byte(`{}`)
		}
		now := s.clock()
		_, err = tx.Exec(ctx, `
			INSERT INTO order_idempotency (key, user_id, order_id, request_hash, status, response_body, expires_at)
			VALUES ($1, $2, $3, $4, 201, $5, $6)`,
			idempotencyKey, customerID, o.ID, requestHash, string(respBody),
			now.Add(48*time.Hour),
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
				pgErr.ConstraintName == "order_idempotency_pkey" {
				tx.Rollback(ctx)
				return nil, fmt.Errorf("idempotency insert: %w", ErrIdempotencyRace)
			}
			tx.Rollback(ctx)
			return nil, fmt.Errorf("insert idempotency: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit order: %w", err)
		}

		return o, nil
	}

	return nil, fmt.Errorf("failed to generate unique order code after %d attempts", maxCodeAttempts)
}

// GetOrderByIdempotencyKey fetches the winner's stored response on the
// race-loser path. Called by the handler after CreateOrder returns
// ErrIdempotencyRace. The handler compares the fetched RequestHash
// with the loser's hash: match → return 201 + ResponseBody (true
// replay), mismatch → 409 IDEMPOTENCY_REPLAY. Returns pgx.ErrNoRows
// if the row is missing (shouldn't happen after a confirmed 23505 —
// vacuumed, perhaps, or expired mid-flight).
func (s *PostgresStore) GetOrderByIdempotencyKey(ctx context.Context, key string, customerID int64) (*IdempotencyRecord, error) {
	rec := &IdempotencyRecord{}
	var status int
	err := s.pool.QueryRow(ctx, `
		SELECT key, user_id, order_id, request_hash, status, response_body
		FROM order_idempotency
		WHERE key = $1 AND user_id = $2`,
		key, customerID,
	).Scan(&rec.Key, &rec.CustomerID, &rec.OrderID, &rec.RequestHash, &status, &rec.ResponseBody)
	if err != nil {
		return nil, err
	}
	rec.Status = status
	return rec, nil
}

// ListAllOrders returns paginated orders for admin with customer info, newest first.
// status and type filters are optional (empty = all).
func (s *PostgresStore) ListAllOrders(ctx context.Context, page, perPage int, status, orderType string) ([]model.Order, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Build WHERE clauses in a single pass
	var conditions []string
	args := []any{}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf(`o.status = $%d`, len(args)+1))
		args = append(args, status)
	}
	if orderType != "" {
		conditions = append(conditions, fmt.Sprintf(`o.type = $%d`, len(args)+1))
		args = append(args, orderType)
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			where += " AND " + c
		}
	}

	// Count query
	countSQL := `SELECT COUNT(*) FROM orders o ` + where
	var total int
	var err error
	if len(args) > 0 {
		err = s.pool.QueryRow(ctx, countSQL, args...).Scan(&total)
	} else {
		err = s.pool.QueryRow(ctx, countSQL).Scan(&total)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	offset := (page - 1) * perPage

	// Build query args in one pass: filter args + LIMIT + OFFSET
	queryArgs := make([]any, len(args))
	copy(queryArgs, args)
	queryArgs = append(queryArgs, perPage, offset)
	limitN := len(args) + 1
	offsetN := len(args) + 2

	query := fmt.Sprintf(`
		SELECT o.id, o.order_code, o.customer_id, o.type, o.total_minor, o.status,
		       o.phone,
		       o.city, o.delivery_method, o.payment_method,
		       o.street, o.house, o.apartment, o.comment,
		       o.height, o.weight, o.delivery_time, o.photos,
		       o.created_at,
		       COALESCE(c.email, '') as customer_email,
		       COALESCE(c.first_name, '') as customer_first_name
		FROM orders o
		LEFT JOIN customers c ON c.id = o.customer_id
		%s
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $%d OFFSET $%d`, where, limitN, offsetN)

	rows, err := s.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	// Empty slice (not nil) so JSON serializes as [] — storefront/profile
	// and admin do len(orders) and crash on null.
	var orders = []model.Order{}
	var orderIDs []int64
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.OrderCode, &o.CustomerID, &o.Type, &o.TotalMinor, &o.Status,
			&o.Phone,
			&o.City, &o.DeliveryMethod, &o.PaymentMethod,
			&o.Street, &o.House, &o.Apartment, &o.Comment,
			&o.Height, &o.Weight, &o.DeliveryTime, &o.Photos,
			&o.CreatedAt,
			&o.CustomerEmail, &o.CustomerFirstName); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}

	// Batch load items with product names
	if len(orderIDs) > 0 {
		itemsMap, err := s.loadOrderItems(ctx, orderIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("load order items: %w", err)
		}
		for i := range orders {
			orders[i].Items = itemsMap[orders[i].ID]
			if orders[i].Items == nil {
				orders[i].Items = []model.OrderItem{}
			}
		}
	}

	return orders, total, nil
}

// loadOrderItems fetches items for given order IDs with product names,
// slugs, and a single representative image. We do TWO batched queries
// instead of N+1 / correlated subqueries per row:
//  1. order_items JOIN products (name, slug)
//  2. product_images for the involved product IDs (one image per
//     product, the lowest sort_order), then join in Go
func (s *PostgresStore) loadOrderItems(ctx context.Context, orderIDs []int64) (map[int64][]model.OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT oi.id, oi.order_id, oi.product_id,
		       COALESCE(p.name, ''), COALESCE(p.slug, ''),
		       oi.size_label, oi.quantity, oi.price_minor,
		       oi.measurements
		FROM order_items oi
		LEFT JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = ANY($1)
		ORDER BY oi.id`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int64][]model.OrderItem)
	productIDs := make(map[int64]struct{})
	for rows.Next() {
		var item model.OrderItem
		var measurementsRaw []byte
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID,
			&item.ProductName, &item.ProductSlug,
			&item.SizeLabel, &item.Quantity, &item.PriceMinor,
			&measurementsRaw); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		if len(measurementsRaw) > 0 {
			if err := json.Unmarshal(measurementsRaw, &item.Measurements); err != nil {
				return nil, fmt.Errorf("unmarshal measurements: %w", err)
			}
		}
		m[item.OrderID] = append(m[item.OrderID], item)
		productIDs[item.ProductID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch load one image per product (DISTINCT ON).
	if len(productIDs) > 0 {
		ids := make([]int64, 0, len(productIDs))
		for id := range productIDs {
			ids = append(ids, id)
		}
		imgRows, err := s.pool.Query(ctx, `
			SELECT DISTINCT ON (product_id) product_id, url
			FROM product_images
			WHERE product_id = ANY($1)
			ORDER BY product_id, sort_order`, ids)
		if err != nil {
			return nil, fmt.Errorf("load order item images: %w", err)
		}
		defer imgRows.Close()
		imageByProduct := make(map[int64]string, len(ids))
		for imgRows.Next() {
			var pid int64
			var url string
			if err := imgRows.Scan(&pid, &url); err != nil {
				return nil, fmt.Errorf("scan order item image: %w", err)
			}
			imageByProduct[pid] = url
		}
		if err := imgRows.Err(); err != nil {
			return nil, err
		}
		// Patch images onto items in-place.
		for orderID, items := range m {
			for i := range items {
				if url, ok := imageByProduct[items[i].ProductID]; ok {
					items[i].ImageURL = url
				}
			}
			m[orderID] = items
		}
	}
	return m, nil
}

// UpdateOrderStatus changes the status of an order. Only valid statuses
// (pending, processing, shipped, delivered, cancelled) are accepted.
func (s *PostgresStore) UpdateOrderStatus(ctx context.Context, orderID int64, status string) error {
	if !validOrderStatuses[status] {
		return fmt.Errorf("invalid order status: %q", status)
	}
	tag, err := s.pool.Exec(ctx, `UPDATE orders SET status = $1 WHERE id = $2`, status, orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("order %d not found", orderID)
	}
	return nil
}

// OrderRequestHash computes the request hash for idempotency.
func OrderRequestHash(method, path, body string, customerID int64) string {
	data := fmt.Sprintf("%s‖%s‖%s‖%d", method, path, body, customerID)
	sum := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", sum)
}

// generateOrderCode produces a human-friendly, unique order code in the form
// of two uppercase letters + hyphen + three digits (e.g. AB-017, KX-429).
// Uniqueness is enforced by a partial unique index on orders(order_code)
// WHERE order_code != '' (migration 019).  Concurrent inserts that pick the
// same code get a 23505 (duplicate key) and the retry loop regenerates.
// Codespace: 26² × 1,000 = 676,000 unique codes.
func (s *PostgresStore) generateOrderCode(ctx context.Context, tx pgx.Tx) (string, error) {
	const maxAttempts = 10
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for attempt := 0; attempt < maxAttempts; attempt++ {
		a := letters[rand.IntN(len(letters))]
		b := letters[rand.IntN(len(letters))]
		code := fmt.Sprintf("%c%c-%03d", a, b, rand.IntN(1000))

		// Uniqueness is guaranteed by the partial unique index —
		// no SELECT EXISTS needed.  Two concurrent INSERTs with
		// the same code will both attempt the INSERT; one wins,
		// the other gets 23505 and retries.
		return code, nil
	}
	return "", fmt.Errorf("failed to generate unique order code after %d attempts", maxAttempts)
}
