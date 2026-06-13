package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"mioru/internal/model"
)

// ErrIdempotencyHashMismatch is returned when an idempotency key is reused
// with a different request body — a probable client bug or replay attack.
var ErrIdempotencyHashMismatch = errors.New("idempotency key reused with different request hash")

// ErrInsufficientStock is returned when an order requests more units of a
// product than the available stock. Wrapped with %w so the handler can
// branch via errors.Is instead of substring-matching the error text —
// per CLAUDE.md, finance-critical paths use sentinels.
var ErrInsufficientStock = errors.New("insufficient stock")

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
		SELECT id, customer_id, type, total_minor, status,
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

	var orders []model.Order
	var orderIDs []int64
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Type, &o.TotalMinor, &o.Status,
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
			return nil, fmt.Errorf("product %d not found", items[i].ProductID)
		}
		items[i].PriceMinor = dbPrice
		totalMinor += dbPrice * int64(items[i].Quantity)
	}
	o.TotalMinor = totalMinor

	// ── Step 2: Transaction — order + items + stock + idempotency ──
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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
		                    city, delivery_method, payment_method,
		                    street, house, apartment, comment,
		                    height, weight, delivery_time, photos)
		VALUES ($1, $2, $3, 'pending',
		        $4, $5, $6,
		        $7, $8, $9, $10,
		        $11, $12, $13, $14)
		RETURNING id, created_at`,
		customerID, o.Type, o.TotalMinor,
		o.City, o.DeliveryMethod, o.PaymentMethod,
		o.Street, o.House, o.Apartment, o.Comment,
		o.Height, o.Weight, o.DeliveryTime, o.Photos,
	).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}
	o.CustomerID = customerID
	o.Status = "pending"

	// Insert line items
	for i := range items {
		items[i].OrderID = o.ID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (order_id, product_id, size_label, quantity, price_minor)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			o.ID, items[i].ProductID, items[i].SizeLabel, items[i].Quantity, items[i].PriceMinor,
		).Scan(&items[i].ID)
		if err != nil {
			return nil, fmt.Errorf("insert order item: %w", err)
		}
	}
	o.Items = items

	// Decrement stock atomically — oversell fails the transaction
	for _, it := range items {
		tag, err := tx.Exec(ctx,
			`UPDATE products SET stock_quantity = stock_quantity - $1
			 WHERE id = $2 AND stock_quantity >= $1`,
			it.Quantity, it.ProductID,
		)
		if err != nil {
			return nil, fmt.Errorf("decrement stock for product %d: %w", it.ProductID, err)
		}
		if tag.RowsAffected() == 0 {
			return nil, fmt.Errorf("product %d: %w (requested %d)", it.ProductID, ErrInsufficientStock, it.Quantity)
		}
	}

	// Store idempotency record
	respBody, _ := json.Marshal(o)
	now := s.clock()
	_, err = tx.Exec(ctx, `
		INSERT INTO order_idempotency (key, user_id, order_id, request_hash, status, response_body, expires_at)
		VALUES ($1, $2, $3, $4, 201, $5, $6)`,
		idempotencyKey, customerID, o.ID, requestHash, string(respBody),
		now.Add(48*time.Hour),
	)
	if err != nil {
		return nil, fmt.Errorf("insert idempotency: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit order: %w", err)
	}

	return o, nil
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
		SELECT o.id, o.customer_id, o.type, o.total_minor, o.status,
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

	var orders []model.Order
	var orderIDs []int64
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Type, &o.TotalMinor, &o.Status,
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

// loadOrderItems fetches items for given order IDs with product names.
func (s *PostgresStore) loadOrderItems(ctx context.Context, orderIDs []int64) (map[int64][]model.OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT oi.id, oi.order_id, oi.product_id, COALESCE(p.name, ''), oi.size_label, oi.quantity, oi.price_minor
		FROM order_items oi
		LEFT JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = ANY($1)
		ORDER BY oi.id`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int64][]model.OrderItem)
	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.SizeLabel, &item.Quantity, &item.PriceMinor); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		m[item.OrderID] = append(m[item.OrderID], item)
	}
	return m, rows.Err()
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
