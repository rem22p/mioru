package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"mioru/internal/model"
)

// TestListCustomerOrdersIsolation verifies two critical properties:
// 1. A customer sees only their own orders (IDOR/scoping guarantee).
// 2. Pagination — total reflects the correct customer, LIMIT/OFFSET works.
func TestListCustomerOrdersIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Seed two customers.
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "alice@example.com", HashedPW: "h1", FirstName: "Alice",
	}); err != nil {
		t.Fatalf("CreateCustomer A: %v", err)
	}
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "bob@example.com", HashedPW: "h2", FirstName: "Bob",
	}); err != nil {
		t.Fatalf("CreateCustomer B: %v", err)
	}

	a, err := s.GetCustomerByEmail(ctx, "alice@example.com")
	if err != nil || a == nil {
		t.Fatalf("GetCustomerByEmail A: cust=%v err=%v", a, err)
	}
	b, err := s.GetCustomerByEmail(ctx, "bob@example.com")
	if err != nil || b == nil {
		t.Fatalf("GetCustomerByEmail B: cust=%v err=%v", b, err)
	}

	// Insert 3 orders for Alice, 2 for Bob.
	for i := 0; i < 3; i++ {
		code := fmt.Sprintf("AA-%03d", i+1)
		_, err := s.pool.Exec(ctx,
			`INSERT INTO orders (customer_id, total_minor, status, order_code) VALUES ($1, $2, 'pending', $3)`,
			a.ID, int64(1000+i*100), code,
		)
		if err != nil {
			t.Fatalf("insert Alice order: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		code := fmt.Sprintf("BB-%03d", i+1)
		_, err := s.pool.Exec(ctx,
			`INSERT INTO orders (customer_id, total_minor, status, order_code) VALUES ($1, $2, 'processing', $3)`,
			b.ID, int64(2000+i*100), code,
		)
		if err != nil {
			t.Fatalf("insert Bob order: %v", err)
		}
	}

	// Alice sees only her 3 orders.
	orders, total, err := s.ListCustomerOrders(ctx, a.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListCustomerOrders A: %v", err)
	}
	if total != 3 {
		t.Errorf("Alice total: want 3, got %d", total)
	}
	if len(orders) != 3 {
		t.Errorf("Alice orders len: want 3, got %d", len(orders))
	}
	for _, o := range orders {
		if o.CustomerID != a.ID {
			t.Errorf("Alice order %d has customer_id %d, want %d", o.ID, o.CustomerID, a.ID)
		}
		// Every order must carry a human-readable order_code (migration 019).
		if o.OrderCode == "" {
			t.Errorf("Alice order %d has empty order_code", o.ID)
		}
	}

	// Bob sees only his 2 orders.
	orders2, total2, err := s.ListCustomerOrders(ctx, b.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListCustomerOrders B: %v", err)
	}
	if total2 != 2 {
		t.Errorf("Bob total: want 2, got %d", total2)
	}
	if len(orders2) != 2 {
		t.Errorf("Bob orders len: want 2, got %d", len(orders2))
	}
	for _, o := range orders2 {
		if o.CustomerID != b.ID {
			t.Errorf("Bob order %d has customer_id %d, want %d", o.ID, o.CustomerID, b.ID)
		}
	}

	// Pagination: Alice page 1 with per_page=2 → 2 items, total still 3.
	orders3, total3, err := s.ListCustomerOrders(ctx, a.ID, 1, 2)
	if err != nil {
		t.Fatalf("ListCustomerOrders A page=1 per=2: %v", err)
	}
	if total3 != 3 {
		t.Errorf("pagination total: want 3, got %d", total3)
	}
	if len(orders3) != 2 {
		t.Errorf("pagination len: want 2, got %d", len(orders3))
	}

	// Alice page 2 with per_page=2 → 1 remaining item.
	orders4, total4, err := s.ListCustomerOrders(ctx, a.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListCustomerOrders A page=2 per=2: %v", err)
	}
	if total4 != 3 {
		t.Errorf("pagination page2 total: want 3, got %d", total4)
	}
	if len(orders4) != 1 {
		t.Errorf("pagination page2 len: want 1, got %d", len(orders4))
	}

	// Newest-first ordering: Alice has 3 orders inserted sequentially,
	// the one with highest ID (last inserted) should be first.
	if len(orders) > 1 && orders[0].ID < orders[1].ID {
		t.Errorf("orders not newest-first: [0].ID=%d < [1].ID=%d", orders[0].ID, orders[1].ID)
	}
}

// ── CreateOrder tests ──

// seedProduct inserts a product with stock and returns its ID.
// The product price is in MDL (minor = price * 100).
func seedProduct(t *testing.T, s *PostgresStore, slug string, priceMDL int, stock int) int64 {
	t.Helper()
	var id int64
	err := s.pool.QueryRow(context.Background(), `
		INSERT INTO products (slug, category_id, brand, name, price, color, status, stock_quantity, created_by)
		VALUES ($1, 1, 'TestBrand', 'Test Product', $2, 'red', 'in_stock', $3, 'test')
		RETURNING id`, slug, priceMDL, stock).Scan(&id)
	if err != nil {
		t.Fatalf("seedProduct: %v", err)
	}
	// Also create product_sizes rows so per-size stock decrement works.
	// Both 'M' and 'L' because tests reference both size labels (e.g.
	// TestCreateOrderRecalculatesPrice uses size 'L' for product 2).
	for _, label := range []string{"M", "L"} {
		_, err = s.pool.Exec(context.Background(), `
			INSERT INTO product_sizes (product_id, size_label, stock_quantity)
			VALUES ($1, $2, $3)`, id, label, stock)
		if err != nil {
			t.Fatalf("seedProduct sizes (%s): %v", label, err)
		}
	}
	return id
}

// TestCreateOrderRecalculatesPrice verifies that the server recalculates
// total_minor and price_minor from DB prices, ignoring client-supplied values.
func TestCreateOrderRecalculatesPrice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Seed customer
	if err := s.CreateCustomer(ctx, model.Customer{Email: "price@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "price@test.com")

	// Seed products with known prices (500 MDL and 300 MDL)
	pid1 := seedProduct(t, s, "price-test-1", 500, 10) // 50000 minor
	pid2 := seedProduct(t, s, "price-test-2", 300, 10) // 30000 minor

	order := &model.Order{
		Type:           "cart",
		City:           "Tiraspol",
		DeliveryMethod: "personal",
		PaymentMethod:  "cash",
	}
	items := []model.OrderItem{
		{ProductID: pid1, SizeLabel: "M", Quantity: 2}, // PriceMinor should be 50000, not from client
		{ProductID: pid2, SizeLabel: "L", Quantity: 1}, // PriceMinor should be 30000
	}

	created, err := s.CreateOrder(ctx, cust.ID, order, items, "price-test-key-1", "hash-1")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Total should be server-calculated: 2*50000 + 1*30000 = 130000
	wantTotal := int64(130000)
	if created.TotalMinor != wantTotal {
		t.Errorf("TotalMinor = %d, want %d (server-calculated)", created.TotalMinor, wantTotal)
	}

	if len(created.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(created.Items))
	}
	if created.Items[0].PriceMinor != 50000 {
		t.Errorf("item 0 PriceMinor = %d, want 50000", created.Items[0].PriceMinor)
	}
	if created.Items[1].PriceMinor != 30000 {
		t.Errorf("item 1 PriceMinor = %d, want 30000", created.Items[1].PriceMinor)
	}

	// OrderCode: must be non-empty and match format AA-000..ZZ-999.
	// Migration 019_order_code.sql + generateOrderCode (collision-retry loop).
	if created.OrderCode == "" {
		t.Error("OrderCode is empty; expected a human-readable code (e.g. AB-017)")
	}
	codeRx := regexp.MustCompile(`^[A-Z]{2}-\d{3}$`)
	if !codeRx.MatchString(created.OrderCode) {
		t.Errorf("OrderCode = %q, want format AA-000..ZZ-999", created.OrderCode)
	}
}

// TestCreateOrderDecrementsStock verifies that product stock is atomically
// decremented when an order is created.
func TestCreateOrderDecrementsStock(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{Email: "stock@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "stock@test.com")

	pid := seedProduct(t, s, "stock-test", 100, 5)

	order := &model.Order{
		Type:           "cart",
		City:           "Tiraspol",
		DeliveryMethod: "personal",
		PaymentMethod:  "cash",
	}
	items := []model.OrderItem{
		{ProductID: pid, SizeLabel: "M", Quantity: 3},
	}

	_, err := s.CreateOrder(ctx, cust.ID, order, items, "stock-key-1", "hash-stock-1")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Verify per-size stock decremented to 2
	var stock int
	err = s.pool.QueryRow(ctx, `SELECT stock_quantity FROM product_sizes WHERE product_id = $1 AND size_label = 'M'`, pid).Scan(&stock)
	if err != nil {
		t.Fatalf("query stock: %v", err)
	}
	if stock != 2 {
		t.Errorf("stock = %d, want 2", stock)
	}
}

// TestCreateOrderOversellFails verifies that ordering more than available
// stock fails the transaction.
func TestCreateOrderOversellFails(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{Email: "oversell@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "oversell@test.com")

	pid := seedProduct(t, s, "oversell-test", 100, 2)

	order := &model.Order{
		Type:           "cart",
		City:           "Tiraspol",
		DeliveryMethod: "personal",
		PaymentMethod:  "cash",
	}
	items := []model.OrderItem{
		{ProductID: pid, SizeLabel: "M", Quantity: 5}, // only 2 in stock
	}

	_, err := s.CreateOrder(ctx, cust.ID, order, items, "oversell-key-1", "hash-over-1")
	if err == nil {
		t.Fatal("expected error for oversell, got nil")
	}

	// Verify per-size stock was NOT decremented (transaction rolled back)
	var stock int
	err = s.pool.QueryRow(ctx, `SELECT stock_quantity FROM product_sizes WHERE product_id = $1 AND size_label = 'M'`, pid).Scan(&stock)
	if err != nil {
		t.Fatalf("query stock: %v", err)
	}
	if stock != 2 {
		t.Errorf("stock = %d, want 2 (unchanged after failed order)", stock)
	}
}

// TestCreateOrderIdempotencyReplay verifies that repeating the same
// idempotency key + request hash returns the stored order without
// re-executing.
func TestCreateOrderIdempotencyReplay(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{Email: "idem@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "idem@test.com")

	pid := seedProduct(t, s, "idem-test", 100, 10)

	order := &model.Order{
		Type:           "cart",
		City:           "Tiraspol",
		DeliveryMethod: "personal",
		PaymentMethod:  "cash",
	}
	items := []model.OrderItem{
		{ProductID: pid, SizeLabel: "M", Quantity: 1},
	}

	// First call
	created1, err := s.CreateOrder(ctx, cust.ID, order, items, "idem-replay-key", "hash-replay")
	if err != nil {
		t.Fatalf("first CreateOrder: %v", err)
	}

	// Second call with same key + hash — should return the stored order
	created2, err := s.CreateOrder(ctx, cust.ID, order, items, "idem-replay-key", "hash-replay")
	if err != nil {
		t.Fatalf("replay CreateOrder: %v", err)
	}

	if created2.ID != created1.ID {
		t.Errorf("replay returned order %d, want same as original %d", created2.ID, created1.ID)
	}
	if created2.TotalMinor != created1.TotalMinor {
		t.Errorf("replay TotalMinor = %d, want %d", created2.TotalMinor, created1.TotalMinor)
	}

	// Verify only 1 order exists in DB
	var count int
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE customer_id = $1`, cust.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if count != 1 {
		t.Errorf("order count = %d, want 1 (replay must not create duplicate)", count)
	}

	// Per-size stock decremented only once
	var stock int
	err = s.pool.QueryRow(ctx, `SELECT stock_quantity FROM product_sizes WHERE product_id = $1 AND size_label = 'M'`, pid).Scan(&stock)
	if err != nil {
		t.Fatalf("query stock: %v", err)
	}
	if stock != 9 {
		t.Errorf("stock = %d, want 9 (decremented only once)", stock)
	}
}

// TestCreateOrderIdempotencyHashMismatch verifies that reusing an
// idempotency key with a different request hash returns
// ErrIdempotencyHashMismatch.
func TestCreateOrderIdempotencyHashMismatch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{Email: "mismatch@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "mismatch@test.com")

	pid := seedProduct(t, s, "mismatch-test", 100, 10)

	order := &model.Order{
		Type:           "cart",
		City:           "Tiraspol",
		DeliveryMethod: "personal",
		PaymentMethod:  "cash",
	}
	items := []model.OrderItem{
		{ProductID: pid, SizeLabel: "M", Quantity: 1},
	}

	// First call with hash-A
	_, err := s.CreateOrder(ctx, cust.ID, order, items, "mismatch-key", "hash-A")
	if err != nil {
		t.Fatalf("first CreateOrder: %v", err)
	}

	// Second call with same key but different hash
	_, err = s.CreateOrder(ctx, cust.ID, order, items, "mismatch-key", "hash-B")
	if !errors.Is(err, ErrIdempotencyHashMismatch) {
		t.Errorf("want ErrIdempotencyHashMismatch, got %v", err)
	}
}

// TestUpdateOrderStatusValidatesEnum verifies that only valid statuses are accepted.
func TestUpdateOrderStatusValidatesEnum(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{Email: "status@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "status@test.com")

	// Create an order directly
	var orderID int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO orders (customer_id, total_minor, status, order_code) VALUES ($1, 1000, 'pending', 'Z-999') RETURNING id`,
		cust.ID).Scan(&orderID)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	// Valid transition
	if err := s.UpdateOrderStatus(ctx, orderID, "processing"); err != nil {
		t.Errorf("valid status 'processing' should succeed, got: %v", err)
	}

	// Invalid status
	if err := s.UpdateOrderStatus(ctx, orderID, "bogus"); err == nil {
		t.Error("invalid status 'bogus' should fail")
	}
}

// TestCreateOrderWithInjectedClock verifies that the idempotency expires_at
// uses the injected clock, not wall-clock time.
func TestCreateOrderWithInjectedClock(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Inject a fixed clock
	fixedTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s.clock = func() time.Time { return fixedTime }

	if err := s.CreateCustomer(ctx, model.Customer{Email: "clock@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "clock@test.com")

	pid := seedProduct(t, s, "clock-test", 100, 10)

	order := &model.Order{
		Type:           "cart",
		City:           "Tiraspol",
		DeliveryMethod: "personal",
		PaymentMethod:  "cash",
	}
	items := []model.OrderItem{
		{ProductID: pid, SizeLabel: "M", Quantity: 1},
	}

	_, err := s.CreateOrder(ctx, cust.ID, order, items, "clock-key", "clock-hash")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Verify expires_at = fixedTime + 48h
	var expiresAt time.Time
	err = s.pool.QueryRow(ctx,
		`SELECT expires_at FROM order_idempotency WHERE key = $1 AND user_id = $2`,
		"clock-key", cust.ID).Scan(&expiresAt)
	if err != nil {
		t.Fatalf("query idempotency: %v", err)
	}

	wantExpiry := fixedTime.Add(48 * time.Hour)
	if !expiresAt.Equal(wantExpiry) {
		t.Errorf("expires_at = %v, want %v", expiresAt, wantExpiry)
	}
}

// TestGetProductPriceMap verifies that prices are loaded correctly.
func TestGetProductPriceMap(t *testing.T) {
	s := testStore(t)

	pid1 := seedProduct(t, s, "pricemap-1", 500, 10) // 50000 minor
	pid2 := seedProduct(t, s, "pricemap-2", 300, 10) // 30000 minor

	prices, err := s.GetProductPriceMap(context.Background(), []int64{pid1, pid2, 99999})
	if err != nil {
		t.Fatalf("GetProductPriceMap: %v", err)
	}

	if prices[pid1] != 50000 {
		t.Errorf("price for pid1 = %d, want 50000", prices[pid1])
	}
	if prices[pid2] != 30000 {
		t.Errorf("price for pid2 = %d, want 30000", prices[pid2])
	}
	if _, ok := prices[99999]; ok {
		t.Error("non-existent product should not be in map")
	}
}

// TestGetOrderByIdempotencyKey exercises the real SQL behind the
// race-loser replay path against the migrated schema. The handler calls
// this after CreateOrder returns ErrIdempotencyRace to fetch the
// winner's stored record and decide replay (201) vs conflict (409). The
// fields it reads (request_hash, status, response_body, order_id) must
// round-trip exactly — a wrong column name here would pass the handler
// fake-store tests but fail in production.
func TestGetOrderByIdempotencyKey(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{Email: "getidem@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(ctx, "getidem@test.com")
	pid := seedProduct(t, s, "getidem-test", 100, 10)

	order := &model.Order{Type: "cart", City: "Tiraspol", DeliveryMethod: "personal", PaymentMethod: "cash"}
	items := []model.OrderItem{{ProductID: pid, SizeLabel: "M", Quantity: 1}}
	created, err := s.CreateOrder(ctx, cust.ID, order, items, "get-idem-key", "hash-xyz")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	rec, err := s.GetOrderByIdempotencyKey(ctx, "get-idem-key", cust.ID)
	if err != nil {
		t.Fatalf("GetOrderByIdempotencyKey: %v", err)
	}
	if rec.Key != "get-idem-key" {
		t.Errorf("Key = %q, want get-idem-key", rec.Key)
	}
	if rec.CustomerID != cust.ID {
		t.Errorf("CustomerID = %d, want %d", rec.CustomerID, cust.ID)
	}
	if rec.OrderID != created.ID {
		t.Errorf("OrderID = %d, want %d", rec.OrderID, created.ID)
	}
	if rec.RequestHash != "hash-xyz" {
		t.Errorf("RequestHash = %q, want hash-xyz", rec.RequestHash)
	}
	if rec.Status != 201 {
		t.Errorf("Status = %d, want 201", rec.Status)
	}
	if rec.ResponseBody == "" {
		t.Error("ResponseBody must not be empty (winner's marshalled order)")
	}

	// Scoping: another customer's lookup of the same key must miss.
	if err := s.CreateCustomer(ctx, model.Customer{Email: "other@test.com", HashedPW: "h"}); err != nil {
		t.Fatalf("CreateCustomer other: %v", err)
	}
	other, _ := s.GetCustomerByEmail(ctx, "other@test.com")
	if _, err := s.GetOrderByIdempotencyKey(ctx, "get-idem-key", other.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("cross-customer lookup err = %v, want pgx.ErrNoRows", err)
	}
}
