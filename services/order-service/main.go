package main

import (
	"context"
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
	authURL := config.Get("AUTH_SERVICE_URL", "http://127.0.0.1:8081")
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
		authDir:   newHTTPAuthClient(authURL, nil),
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
	// Static segment before the {id} param — chi prefers the literal route.
	r.Get("/v1/admin/orders/customers", svc.handleListCustomerStats)
	r.Get("/v1/admin/orders/{id}", svc.handleGetAdminOrder)
	r.Post("/v1/admin/orders/{id}/complete", svc.handleCompleteOrder)
	r.Get("/v1/admin/delivery-fee", svc.handleGetAdminDeliveryFee)
	r.Put("/v1/admin/delivery-fee", svc.handlePutAdminDeliveryFee)
	r.Get("/v1/admin/desk-settings", svc.handleGetDeskSettings)
	r.Put("/v1/admin/desk-settings", svc.handlePutDeskSettings)

	slog.Info("upstream urls", "auth", authURL, "geo", geoURL, "catalog", catalogURL, "billing", billingURL, "inventory", inventoryURL, "nats", natsURL)

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
	if _, err = db.Exec(string(sqlBytes)); err != nil {
		return err
	}
	// CREATE TABLE IF NOT EXISTS does not evolve databases already deployed.
	// There is no migration tool in this repo, so additive columns are applied
	// here and must stay idempotent.
	for _, column := range []struct {
		table, name, definition string
	}{
		{"order_items", "unit_cost", "INTEGER NOT NULL DEFAULT 0"},
		{"orders", "customer_phone", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := ensureColumn(context.Background(), db, column.table, column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

// ensureColumn adds a column when an already-deployed table lacks it.
func ensureColumn(ctx context.Context, db *sql.DB, table, name, definition string) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid        int
			colName    string
			colType    string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if colName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+name+" "+definition)
	return err
}
