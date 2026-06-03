package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"mioru/internal/model"
)

// ListCustomerOrders returns paginated orders for a customer, newest first.
func (s *PostgresStore) ListCustomerOrders(ctx context.Context, customerID int64, page, perPage int) ([]model.Order, int, error) {
	if page < 1 { page = 1 }
	if perPage < 1 { perPage = 20 }
	if perPage > 100 { perPage = 100 }

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

// CreateOrder inserts an order with its line items and idempotency guard
// inside a single transaction. idempotencyKey is the client-supplied UUID;
// requestHash is SHA256(method ‖ path ‖ canonicalized_body ‖ customer_id).
func (s *PostgresStore) CreateOrder(ctx context.Context, customerID int64, o *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Expire old idempotency records
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

	// Store idempotency record
	respBody, _ := json.Marshal(o)
	var respHash [32]byte
	copy(respHash[:], requestHash)
	_, err = tx.Exec(ctx, `
		INSERT INTO order_idempotency (key, order_id, request_hash, status, response_body, expires_at)
		VALUES ($1, $2, $3, 201, $4, $5)`,
		idempotencyKey, o.ID, requestHash, string(respBody),
		time.Now().UTC().Add(48*time.Hour),
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
	if page < 1 { page = 1 }
	if perPage < 1 { perPage = 20 }
	if perPage > 100 { perPage = 100 }

	// Build WHERE clauses
	var conditions []string
	args := []any{}
	an := 1 // arg counter
	if status != "" {
		conditions = append(conditions, fmt.Sprintf(`o.status = $%d`, an))
		args = append(args, status)
		an++
	}
	if orderType != "" {
		conditions = append(conditions, fmt.Sprintf(`o.type = $%d`, an))
		args = append(args, orderType)
		an++
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
	// Query orders with customer info
	queryArgs := []any{}
	qan := 1
	if status != "" {
		queryArgs = append(queryArgs, status)
		qan++
	}
	if orderType != "" {
		queryArgs = append(queryArgs, orderType)
		qan++
	}
	queryArgs = append(queryArgs, perPage, offset)
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
		LIMIT $%d OFFSET $%d`, where, qan, qan+1)

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

// UpdateOrderStatus changes the status of an order.
func (s *PostgresStore) UpdateOrderStatus(ctx context.Context, orderID int64, status string) error {
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

var _ = time.Now // pull in time for unused import until tests land
