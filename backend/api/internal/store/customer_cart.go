package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ── Cart ──

// CartItem is a single row in customer_cart.
type CartItem struct {
	ProductID   int    `json:"product_id"`
	SizeLabel   string `json:"size_label"`
	Quantity    int    `json:"quantity"`
	ProductSlug string `json:"product_slug"`
}

// GetCustomerCart returns all cart items for a customer.
func (s *PostgresStore) GetCustomerCart(ctx context.Context, customerID int64) ([]CartItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.product_id, c.size_label, c.quantity, p.slug
		FROM customer_cart c
		JOIN products p ON p.id = c.product_id
		WHERE c.customer_id = $1
		ORDER BY c.created_at`, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer cart: %w", err)
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		var item CartItem
		if err := rows.Scan(&item.ProductID, &item.SizeLabel, &item.Quantity, &item.ProductSlug); err != nil {
			return nil, fmt.Errorf("scan cart row: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveCustomerCart replaces the entire cart for a customer atomically
// using a batched insert (pgx.Batch) to avoid N+1 round-trips.
//
// All product IDs in the cart are validated against `products` in a
// single SELECT before the batch INSERT runs. The FK constraint would
// catch missing products at INSERT time, but only with a generic
// SQLSTATE 23503 error that the handler would treat as a 500 ISE —
// here we turn it into a 400 PRODUCT_NOT_FOUND with a clear message
// so the client knows the cart is stale and the user has to refresh.
// Defence-in-depth: even with this pre-check, the FK still protects
// against a race where a product is deleted between the SELECT and
// the INSERT (rare; the handler will return 500 ISE in that case,
// which is the correct response for a true race).
func (s *PostgresStore) SaveCustomerCart(ctx context.Context, customerID int64, items []CartItem) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear existing cart.
	if _, err := tx.Exec(ctx, `DELETE FROM customer_cart WHERE customer_id = $1`, customerID); err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}

	if len(items) > 0 {
		// Collect distinct product IDs to validate existence. CartItem
		// uses `int` (JSON-friendly) but products.id is BIGINT, so we
		// convert at the SQL boundary.
		productIDs := make([]int64, 0, len(items))
		seen := make(map[int]bool, len(items))
		for _, it := range items {
			if !seen[it.ProductID] {
				productIDs = append(productIDs, int64(it.ProductID))
				seen[it.ProductID] = true
			}
		}
		rows, err := tx.Query(ctx, `SELECT id FROM products WHERE id = ANY($1)`, productIDs)
		if err != nil {
			return fmt.Errorf("validate cart products: %w", err)
		}
		existing := make(map[int64]bool, len(productIDs))
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("scan product id: %w", err)
			}
			existing[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate product ids: %w", err)
		}
		// Fail-fast: report the first unknown product id. We don't
		// enumerate every missing product because the user can only
		// fix one stale line at a time in the UI anyway, and the
		// caller already has the full cart to retry from.
		for _, id := range productIDs {
			if !existing[id] {
				return fmt.Errorf("product %d: %w", id, ErrProductNotFound)
			}
		}

		batch := &pgx.Batch{}
		for _, item := range items {
			batch.Queue(`
				INSERT INTO customer_cart (customer_id, product_id, size_label, quantity)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (customer_id, product_id, size_label)
				DO UPDATE SET quantity = EXCLUDED.quantity`,
				customerID, item.ProductID, item.SizeLabel, item.Quantity)
		}
		br := tx.SendBatch(ctx, batch)
		// Close the batch result — errors surface on Close.
		if err := br.Close(); err != nil {
			return fmt.Errorf("batch insert cart: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ── Favorites ──

// GetCustomerFavorites returns all favorite product IDs for a customer.
func (s *PostgresStore) GetCustomerFavorites(ctx context.Context, customerID int64) ([]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT product_id FROM customer_favorites
		WHERE customer_id = $1
		ORDER BY created_at`, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer favorites: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan favorite row: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SaveCustomerFavorites replaces all favorites for a customer atomically
// using a batched insert (pgx.Batch) to avoid N+1 round-trips.
func (s *PostgresStore) SaveCustomerFavorites(ctx context.Context, customerID int64, productIDs []int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM customer_favorites WHERE customer_id = $1`, customerID); err != nil {
		return fmt.Errorf("clear favorites: %w", err)
	}

	if len(productIDs) > 0 {
		batch := &pgx.Batch{}
		for _, pid := range productIDs {
			batch.Queue(`
				INSERT INTO customer_favorites (customer_id, product_id)
				VALUES ($1, $2)
				ON CONFLICT (customer_id, product_id) DO NOTHING`,
				customerID, pid)
		}
		br := tx.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			return fmt.Errorf("batch insert favorites: %w", err)
		}
	}

	return tx.Commit(ctx)
}
