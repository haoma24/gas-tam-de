package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gas-tam-de/pkg/events"
	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/natsx"

	"github.com/nats-io/nats-server/v2/server"
)

func TestCreateAndPatchPublishProductUpdated(t *testing.T) {
	bus := &recordingProductPublisher{}
	svc := &catalogService{db: openTestDB(t), bus: bus}
	r := httpx.NewRouter("catalog-event-test")
	r.Post("/v1/admin/products", svc.handleCreateProduct)
	r.Patch("/v1/admin/products/{id}", svc.handlePatchProduct)

	createBody, _ := json.Marshal(map[string]any{
		"sku":        "GAS12",
		"name":       "Gas 12kg",
		"sale_price": 450000,
		"active":     true,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/products", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created product
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(bus.events) != 1 {
		t.Fatalf("events after create=%d want 1", len(bus.events))
	}
	got := bus.events[0]
	if got.ID != created.ID || got.SKU != "GAS12" || got.SalePrice != 450000 || !got.Active {
		t.Fatalf("create event=%+v", got)
	}

	patchBody, _ := json.Marshal(map[string]any{"active": false, "sale_price": 460000})
	req = httptest.NewRequest(http.MethodPatch, "/v1/admin/products/"+created.ID, bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(bus.events) != 2 {
		t.Fatalf("events after patch=%d want 2", len(bus.events))
	}
	got = bus.events[1]
	if got.ID != created.ID || got.SalePrice != 460000 || got.Active {
		t.Fatalf("hide/update event=%+v", got)
	}
}

func TestJSProductPublisherPublishesEnvelope(t *testing.T) {
	opts := &server.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats server not ready")
	}
	defer ns.Shutdown()

	nc, js, err := natsx.ConnectJS(ns.ClientURL())
	if err != nil {
		t.Fatalf("ConnectJS: %v", err)
	}
	defer nc.Close()
	if err := natsx.EnsureStreams(js); err != nil {
		t.Fatalf("EnsureStreams: %v", err)
	}

	sub, err := js.SubscribeSync(events.CatalogProductUpdated)
	if err != nil {
		t.Fatalf("SubscribeSync: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	pub := newJSProductPublisher(natsx.Static(js))
	p := product{ID: "prod-1", SKU: "GAS12", Name: "Gas 12kg", SalePrice: 450000, Active: true}
	if err := pub.PublishProductUpdated(p); err != nil {
		t.Fatalf("PublishProductUpdated: %v", err)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	var env events.Envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Subject != events.CatalogProductUpdated {
		t.Fatalf("subject=%q", env.Subject)
	}
	if env.EventID == "" || env.SchemaVersion != 1 {
		t.Fatalf("envelope=%+v", env)
	}
	if env.Payload["product_id"] != "prod-1" {
		t.Fatalf("payload=%v", env.Payload)
	}
	if env.Payload["sku"] != "GAS12" {
		t.Fatalf("payload sku=%v", env.Payload["sku"])
	}
	// inventory-service creates its stock row from this payload; without the
	// name every synced row would be labelled with the bare SKU.
	if env.Payload["name"] != "Gas 12kg" {
		t.Fatalf("payload name=%v", env.Payload["name"])
	}
	// JSON numbers decode as float64
	if sp, ok := env.Payload["sale_price"].(float64); !ok || int64(sp) != 450000 {
		t.Fatalf("sale_price=%v", env.Payload["sale_price"])
	}
	if active, ok := env.Payload["active"].(bool); !ok || !active {
		t.Fatalf("active=%v", env.Payload["active"])
	}
	info, err := js.StreamInfo("CATALOG")
	if err != nil {
		t.Fatalf("StreamInfo: %v", err)
	}
	if info.State.Msgs < 1 {
		t.Fatalf("CATALOG msgs=%d want >=1", info.State.Msgs)
	}
}
