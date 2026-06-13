package store

import "context"

// ResetTestData truncates every data table so a test starts from a known,
// empty state, while keeping the seeded category tree (categories is not
// truncated). RESTART IDENTITY resets SERIAL counters; CASCADE covers FK
// dependents.
//
// TEST-ONLY: this is destructive and must only be run against a disposable
// database (TEST_DATABASE_URL). It is exported (not a _test helper) so that
// both the in-package white-box store tests and the external internal/storetest
// package can share one canonical table list — Go's import-cycle rule forbids
// the in-package store tests from importing storetest, which imports store.
func (s *PostgresStore) ResetTestData(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`TRUNCATE products, product_sizes, size_chart_rows, product_images,
		         users, customers, customer_oauth, password_reset_tokens, orders,
		         order_items, order_idempotency,
		         customer_cart, customer_favorites
		 RESTART IDENTITY CASCADE`)
	return err
}
