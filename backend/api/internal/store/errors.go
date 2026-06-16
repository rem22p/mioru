// Package store contains all the data-access sentinels that the handler
// layer branches on. Keeping them in their own file (rather than scattered
// across the per-table *.go files) makes it easy to audit the public
// contract: every error a handler can intentionally respond to with a
// specific status code lives here.
//
// Any error that should be returned to the HTTP client with a specific
// status code (e.g. 400 PRODUCT_NOT_FOUND, 409 INSUFFICIENT_STOCK) MUST
// be a sentinel. Generic `fmt.Errorf("...")` errors are always treated
// as 500 ISE in the handler's catch-all, per the project logging
// standard (CLAUDE.md: "ERROR means operation failed").
package store

import "errors"

// ErrInsufficientStock is returned when a customer tries to buy more of
// a product than is available. The handler maps this to
// 409 INSUFFICIENT_STOCK. Defined here so that the same sentinel
// historically living in order_postgres.go is reachable from the
// handler tests without an import cycle (store tests import
// order_postgres_test.go which itself imports this file).
var ErrInsufficientStock = errors.New("insufficient stock")

// ErrIdempotencyHashMismatch is returned when an Idempotency-Key is
// reused for a different request body. The handler maps this to
// 409 IDEMPOTENCY_REPLAY.
var ErrIdempotencyHashMismatch = errors.New("idempotency hash mismatch")

// ErrIdempotencyRace is returned when a concurrent first submit with
// the same Idempotency-Key wins the race. The handler re-fetches the
// stored record and returns 201 (true replay) or 409 (real conflict).
var ErrIdempotencyRace = errors.New("idempotency race")

// ErrProductNotFound is returned by:
//
//   - SaveCustomerCart: when the cart contains a product_id that no
//     longer exists in `products` (e.g. admin deleted it, or the
//     client built a cart from a stale frontend cache).
//   - CreateOrder: when one or more line items reference a missing
//     product.
//
// The handler maps this to 400 PRODUCT_NOT_FOUND so the client knows
// the problem is a stale cart and the user has to refresh — not a
// transient server failure.
var ErrProductNotFound = errors.New("product not found")
