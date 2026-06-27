package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

func TestCreateOrderWithMeasurementsRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	pid, err := s.CreateProduct(ctx, model.Product{
		Slug:       "measure-rt",
		CategoryID: 2,
		Brand:      "TestBrand",
		Name:       "Measure RT",
		Price:      990,
		Color:       "blue",
		Status:      "in_stock",
		InStock:     true,
		StockQty:    5,
		CreatedBy:   "test",
		Sizes: []model.ProductSize{
			{Label: "M", StockQuantity: 5},
		},
	})
	if err != nil {
		t.Skipf("CreateProduct: %v (no test DB?)", err)
		return
	}

	// Create a customer first (FK required)
	err = s.CreateCustomer(ctx, model.Customer{
		Email: "measure-rt@test.com", HashedPW: "test123",
		FirstName: "Test", LastName: "User",
	})
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	// Look up the ID
	cust, err := s.GetCustomerByEmail(ctx, "measure-rt@test.com")
	if err != nil {
		t.Fatalf("GetCustomerByEmail: %v", err)
	}
	custID := cust.ID

	order := &model.Order{
		CustomerID: custID, Phone: "+37369123456", City: "Tiraspol",
		DeliveryMethod: "pickup",
		PaymentMethod:  "cod",
		Status:     "pending",
		TotalMinor:  990,
		Items: []model.OrderItem{{
			ProductID:    pid,
			SizeLabel:    "M",
			Quantity:     1,
			PriceMinor:   990,
			Measurements: map[string]interface{}{
				"height": 175.0,
				"weight": 70.0,
				"foot_length": 27.0,
			},
		}},
	}

	created, err := s.CreateOrder(ctx, 1, order, order.Items, "k1", "h1")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("order ID is 0")
	}

	// Read back
	orders, _, err := s.ListCustomerOrders(ctx, custID, 1, 10)
	if err != nil {
		t.Fatalf("ListCustomerOrders: %v", err)
	}
	if len(orders) == 0 {
		t.Fatal("no orders returned")
	}

	item := orders[0].Items[0]
	if item.Measurements == nil {
		t.Fatal("measurements is nil")
	}
	if h, ok := item.Measurements["height"].(float64); !ok || h != 175.0 {
		t.Errorf("height = %v", h)
	}
	if w, ok := item.Measurements["weight"].(float64); !ok || w != 70.0 {
		t.Errorf("weight = %v", w)
	}
	if fl, ok := item.Measurements["foot_length"].(float64); !ok || fl != 27.0 {
		t.Errorf("foot_length = %v", fl)
	}
}

func TestUpdateProductSizeStockRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	pid, err := s.CreateProduct(ctx, model.Product{
		Slug: "stock-rt", CategoryID: 2, Brand: "Test", Name: "Stock RT",
		Price: 500, Color: "red", Status: "in_stock", InStock: true,
		StockQty: 10, CreatedBy: "test",
		Sizes: []model.ProductSize{
			{Label: "S", StockQuantity: 2},
			{Label: "M", StockQuantity: 3},
			{Label: "L", StockQuantity: 1},
		},
	})
	if err != nil {
		t.Skipf("CreateProduct: %v", err)
		return
	}

	// Update: M stock +5, drop L, add XL=4
	err = s.UpdateProduct(ctx, "stock-rt", model.Product{
		ID: pid, Slug: "stock-rt", CategoryID: 2, Brand: "Test",
		Name: "Stock RT", Price: 600, Color: "red", Status: "in_stock",
		InStock: true, StockQty: 10, CreatedBy: "test",
		Sizes: []model.ProductSize{
			{Label: "S", StockQuantity: 2},
			{Label: "M", StockQuantity: 5},
			{Label: "XL", StockQuantity: 4},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	got, err := s.GetProduct(ctx, "stock-rt")
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}

	if len(got.Sizes) != 3 {
		t.Fatalf("got %d sizes, want 3", len(got.Sizes))
	}

	find := func(label string) int {
		for _, sz := range got.Sizes {
			if sz.Label == label {
				return sz.StockQuantity
			}
		}
		return -1
	}

	if q := find("S"); q != 2 {
		t.Errorf("S = %d", q)
	}
	if q := find("M"); q != 5 {
		t.Errorf("M = %d", q)
	}
	if q := find("XL"); q != 4 {
		t.Errorf("XL = %d", q)
	}
	if q := find("L"); q != -1 {
		t.Errorf("L should be gone, got %d", q)
	}
}
