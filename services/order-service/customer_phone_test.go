package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testOrderRouterWithPhones(t *testing.T, dir phoneDirectory) (*orderService, http.Handler) {
	t.Helper()
	svc := &orderService{
		db:      openTestOrderDB(t),
		geo:     &stubGeo{},
		catalog: &stubCatalog{},
		billing: noopBillingRecorder{},
		authDir: dir,
		bus:     noopOrderPublisher{},
	}
	return svc, mountOrderTestRoutes(svc)
}

// TestAdminOrdersExposeFullPhone is the reported bug: the shop cannot deliver
// an order it has no number to call about, so admin listings carry the real
// number, not the masked one.
func TestAdminOrdersExposeFullPhone(t *testing.T) {
	dir := &stubPhoneDirectory{phones: map[string]string{"user-a": "0909777020"}}
	svc, r := testOrderRouterWithPhones(t, dir)
	insertTestOrder(t, svc, "ord-1", "user-a", orderStatusPending, "2026-08-01T09:00:00Z")

	got := listAdminOrders(t, r, "")
	if len(got) != 1 {
		t.Fatalf("want 1 order, got %d", len(got))
	}
	if got[0].CustomerPhone != "0909777020" {
		t.Fatalf("customer_phone=%q, want the full number", got[0].CustomerPhone)
	}
	// The masked value stays available as a fallback for rows with no number.
	if got[0].PhoneMasked == "" {
		t.Fatal("phone_masked should still be present")
	}
}

// TestCustomerOrdersStayMasked — widening admin access must not widen the
// customer's own API, which has always returned a masked number.
func TestCustomerOrdersStayMasked(t *testing.T) {
	dir := &stubPhoneDirectory{phones: map[string]string{"user-1": "0909777020"}}
	svc, r := testOrderRouterWithPhones(t, dir)
	insertTestOrder(t, svc, "ord-1", "user-1", orderStatusPending, "2026-08-01T09:00:00Z")
	if _, err := svc.db.Exec(
		`UPDATE orders SET customer_phone = '0909777020' WHERE id = 'ord-1'`); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/orders/me", nil)
	customerHeaders(req)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if body := rr.Body.String(); strings.Contains(body, "0909777020") {
		t.Fatalf("customer response leaked the full phone: %s", body)
	}
	var payload struct {
		Orders []orderView `json:"orders"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Orders) != 1 || payload.Orders[0].CustomerPhone != "" {
		t.Fatalf("customer_phone must be absent, got %+v", payload.Orders)
	}
}

// TestBackfillPhonesIsBatchedAndPersisted covers orders placed before the
// column existed: one lookup for the whole page, written back so the next
// listing needs no call at all.
func TestBackfillPhonesIsBatchedAndPersisted(t *testing.T) {
	dir := &stubPhoneDirectory{phones: map[string]string{
		"user-a": "0909111111",
		"user-b": "0909222222",
	}}
	svc, r := testOrderRouterWithPhones(t, dir)
	insertTestOrder(t, svc, "ord-1", "user-a", orderStatusPending, "2026-08-01T09:00:00Z")
	insertTestOrder(t, svc, "ord-2", "user-b", orderStatusPending, "2026-08-01T10:00:00Z")

	got := listAdminOrders(t, r, "")
	if len(got) != 2 || got[0].CustomerPhone != "0909111111" || got[1].CustomerPhone != "0909222222" {
		t.Fatalf("phones not backfilled: %+v", got)
	}
	if dir.calls != 1 || dir.batches[0] != 2 {
		t.Fatalf("want one batched lookup of 2 ids, got calls=%d batches=%v", dir.calls, dir.batches)
	}

	// Second listing must be served from the persisted column.
	listAdminOrders(t, r, "")
	if dir.calls != 1 {
		t.Fatalf("lookup repeated: calls=%d, want the phone persisted after the first", dir.calls)
	}

	var stored string
	if err := svc.db.QueryRow(
		`SELECT customer_phone FROM orders WHERE id = 'ord-1'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "0909111111" {
		t.Fatalf("stored customer_phone=%q", stored)
	}
}

// TestBackfillSurvivesAuthOutage — the Order Desk must keep working when
// auth-service is unreachable; it just shows no full number.
func TestBackfillSurvivesAuthOutage(t *testing.T) {
	svc, r := testOrderRouterWithPhones(t, failingPhoneDirectory{})
	insertTestOrder(t, svc, "ord-1", "user-a", orderStatusPending, "2026-08-01T09:00:00Z")

	got := listAdminOrders(t, r, "")
	if len(got) != 1 {
		t.Fatalf("want 1 order, got %d", len(got))
	}
	if got[0].CustomerPhone != "" {
		t.Fatalf("customer_phone=%q, want empty", got[0].CustomerPhone)
	}
	if got[0].PhoneMasked == "" {
		t.Fatal("masked fallback missing")
	}
}

// TestDisplayPhoneUsesLocalForm — a shop dials 09…, not +849….
func TestDisplayPhoneUsesLocalForm(t *testing.T) {
	cases := map[string]string{
		"+84909777020": "0909777020",
		"0909777020":   "0909777020",
		"":             "",
		"  ":           "",
	}
	for in, want := range cases {
		if got := displayPhone(in); got != want {
			t.Fatalf("displayPhone(%q)=%q, want %q", in, got, want)
		}
	}
}

type failingPhoneDirectory struct{}

func (failingPhoneDirectory) PhonesByUserID(_ context.Context, _ []string) (map[string]string, error) {
	return nil, errString("auth unreachable")
}
