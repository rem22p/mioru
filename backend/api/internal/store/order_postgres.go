package store

import (
	"context"
	"fmt"

	"mioru/internal/model"
)

// ListCustomerOrders returns paginated orders for a customer, newest first.
// page is 1-based, perPage clamped to [1, 100].
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
		SELECT id, customer_id, total_minor, status, created_at::text as created_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, customerID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list customer orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.TotalMinor, &o.Status, &o.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}
