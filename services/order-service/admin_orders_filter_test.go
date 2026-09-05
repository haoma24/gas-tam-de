package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func listAdminOrders(t *testing.T, r http.Handler, query string) []orderView {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders"+query, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Orders []orderView `json:"orders"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Orders
}

func seedMixedStatusOrders(t *testing.T, svc *orderService) {
	t.Helper()
	insertTestOrder(t, svc, "ord-pending-old", "user-a", orderStatusPending, "2026-08-01T09:00:00Z")
	insertTestOrder(t, svc, "ord-pending-new", "user-b", orderStatusPending, "2026-08-03T09:00:00Z")
	insertTestOrder(t, svc, "ord-done-old", "user-c", orderStatusCompleted, "2026-08-02T09:00:00Z")
	insertTestOrder(t, svc, "ord-done-new", "user-d", orderStatusCompleted, "2026-08-04T09:00:00Z")
	insertTestOrder(t, svc, "ord-void", "user-e", orderStatusCancelled, "2026-08-05T09:00:00Z")
}

// TestAdminOrdersStatusAll is the fix for "đơn hoàn tất rồi thì không xem lại
// được": the desk endpoint has to be able to answer for every status, not only
// the pending queue.
func TestAdminOrdersStatusAll(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	seedMixedStatusOrders(t, svc)

	got := listAdminOrders(t, r, "?status=ALL")
	if len(got) != 5 {
		t.Fatalf("status=ALL returned %d orders, want 5", len(got))
	}
	// History reads newest first.
	if got[0].ID != "ord-void" {
		t.Fatalf("first=%q, want the newest order ord-void", got[0].ID)
	}
	if got[len(got)-1].ID != "ord-pending-old" {
		t.Fatalf("last=%q, want the oldest order", got[len(got)-1].ID)
	}
}

func TestAdminOrdersCompletedNewestFirst(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	seedMixedStatusOrders(t, svc)

	got := listAdminOrders(t, r, "?status=COMPLETED")
	if len(got) != 2 {
		t.Fatalf("want 2 completed, got %d", len(got))
	}
	if got[0].ID != "ord-done-new" || got[1].ID != "ord-done-old" {
		t.Fatalf("order=%q,%q want newest first", got[0].ID, got[1].ID)
	}
	// A FIFO position is meaningless in a history listing.
	if got[0].Stt != 0 {
		t.Fatalf("stt=%d, want unset for history rows", got[0].Stt)
	}
}

// TestAdminOrdersPendingStaysFIFO — the filter must not disturb the desk queue,
// which is the one place the oldest order has to come first.
func TestAdminOrdersPendingStaysFIFO(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	seedMixedStatusOrders(t, svc)

	for _, query := range []string{"", "?status=PENDING", "?status=pending"} {
		got := listAdminOrders(t, r, query)
		if len(got) != 2 {
			t.Fatalf("query %q: want 2 pending, got %d", query, len(got))
		}
		if got[0].ID != "ord-pending-old" || got[0].Stt != 1 {
			t.Fatalf("query %q: first=%q stt=%d, want ord-pending-old/1",
				query, got[0].ID, got[0].Stt)
		}
	}
}

func TestAdminOrdersRejectsUnknownStatus(t *testing.T) {
	_, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders?status=SHIPPED", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}
}

// TestAdminOrderDetailShowsSettlement — reopening a finished order has to show
// what was collected and what is still owed, or the history is unusable.
func TestAdminOrderDetailShowsSettlement(t *testing.T) {
	svc, r := testOrderRouter(t, &stubGeo{}, &stubCatalog{})
	insertTestOrder(t, svc, "ord-1", "user-a", orderStatusPending, "2026-08-01T09:00:00Z")

	if _, err := svc.db.Exec(`
		UPDATE orders SET status = 'COMPLETED', completed_at = ?, payment_type = 'PARTIAL', amount_paid = ?
		WHERE id = 'ord-1'`, "2026-08-01T10:00:00Z", 40000); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orders/ord-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got orderView
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.CompletedAt != "2026-08-01T10:00:00Z" {
		t.Fatalf("completed_at=%q", got.CompletedAt)
	}
	if got.PaymentType != "PARTIAL" || got.AmountPaid != 40000 {
		t.Fatalf("payment=%q paid=%d", got.PaymentType, got.AmountPaid)
	}
}

// stubPhoneDirectory answers phone lookups without an auth-service.
type stubPhoneDirectory struct {
	phones map[string]string
	calls  int
	// batches records the id-count of each call, so a test can prove the
	// backfill is batched rather than one request per order.
	batches []int
}

func (s *stubPhoneDirectory) PhonesByUserID(_ context.Context, ids []string) (map[string]string, error) {
	s.calls++
	s.batches = append(s.batches, len(ids))
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		if p, ok := s.phones[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}
