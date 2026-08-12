package main

import (
	"database/sql"
	"embed"
	"log/slog"
	"os"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/natsx"
	"gas-tam-de/pkg/sqlite"
)

const serviceName = "order-service"

//go:embed schema.sql
var schemaFS embed.FS

func main() {
	addr := config.ListenAddr("ORDER_ADDR", ":8084")
	dbPath := config.Get("ORDER_DB", "data/order.db")
	geoURL := config.Get("GEO_SERVICE_URL", "http://127.0.0.1:8083")
	catalogURL := config.Get("CATALOG_SERVICE_URL", "http://127.0.0.1:8082")
	billingURL := config.Get("BILLING_SERVICE_URL", "http://127.0.0.1:8086")
	inventoryURL := config.Get("INVENTORY_SERVICE_URL", "http://127.0.0.1:8085")
	natsURL := config.Get("NATS_URL", "nats://127.0.0.1:4222")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	if err := seedDeliveryFee(db, loadDeliveryFeeSeedConfig()); err != nil {
		slog.Error("seed delivery fee", "err", err)
		os.Exit(1)
	}
	if err := seedDeskSettings(db); err != nil {
		slog.Error("seed desk settings", "err", err)
		os.Exit(1)
	}

	// NATS connects in the background so /healthz answers even if the broker
	// is still starting; publish failures are logged, not fatal.
	bus := natsx.NewBackground(natsURL)
	bus.Start(nil)
	defer bus.Close()

	svc := &orderService{
		db:        db,
		geo:       newHTTPGeoClient(geoURL, nil),
		catalog:   newHTTPCatalogClient(catalogURL, nil),
		billing:   newHTTPBillingClient(billingURL, nil),
		inventory: newHTTPInventoryClient(inventoryURL, nil),
		bus:       newJSOrderPublisher(bus),
	}

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)
	// Checkout calls these four over HTTP, so an unreachable one must be
	// diagnosable from /readyz instead of only from a customer's failed order.
	httpx.MountReady(r, serviceName,
		httpx.ReadyCheck{Name: "nats", Check: bus.ReadyCheck},
		upstreamReadyCheck("geo", geoURL),
		upstreamReadyCheck("catalog", catalogURL),
		upstreamReadyCheck("billing", billingURL),
		upstreamReadyCheck("inventory", inventoryURL),
	)

	r.Post("/v1/orders/quote", svc.handleQuoteOrder)
	r.Post("/v1/orders", svc.handleCreateOrder)
	r.Post("/v1/orders/{id}/cancel", svc.handleCancelMyOrder)

	r.Get("/v1/orders/me/defaults", svc.handleGetMyOrderDefaults)
	r.Get("/v1/orders/me", svc.handleListMyOrders)
	r.Get("/v1/admin/orders", svc.handleListAdminOrders)
	r.Get("/v1/admin/orders/{id}", svc.handleGetAdminOrder)
	r.Post("/v1/admin/orders/{id}/complete", svc.handleCompleteOrder)
	r.Get("/v1/admin/delivery-fee", svc.handleGetAdminDeliveryFee)
	r.Put("/v1/admin/delivery-fee", svc.handlePutAdminDeliveryFee)
	r.Get("/v1/admin/desk-settings", svc.handleGetDeskSettings)
	r.Put("/v1/admin/desk-settings", svc.handlePutDeskSettings)

	slog.Info("upstream urls", "geo", geoURL, "catalog", catalogURL, "billing", billingURL, "inventory", inventoryURL, "nats", natsURL)

	if err := httpx.ListenAndServe(addr, serviceName, r); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func migrate(db *sql.DB) error {
	sqlBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}
