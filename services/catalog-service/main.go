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

const serviceName = "catalog-service"

//go:embed schema.sql
var schemaFS embed.FS

func main() {
	addr := config.ListenAddr("CATALOG_ADDR", ":8082")
	dbPath := config.Get("CATALOG_DB", "data/catalog.db")
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

	// NATS connects in the background so /healthz answers even if the broker
	// is still starting; publish failures are logged, not fatal.
	bus := natsx.NewBackground(natsURL)
	bus.Start(nil)
	defer bus.Close()

	svc := &catalogService{db: db, bus: newJSProductPublisher(bus)}

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)
	httpx.MountReady(r, serviceName, httpx.ReadyCheck{Name: "nats", Check: bus.ReadyCheck})

	// Public browse — active products only (T2.2.1); gateway mounts as public.
	r.Get("/v1/products", svc.handleListActiveProducts)

	// Admin CRUD (architecture §4.4). Authz enforced at gateway when proxied.
	r.Get("/v1/admin/products", svc.handleListAdminProducts)
	r.Post("/v1/admin/products", svc.handleCreateProduct)
	r.Get("/v1/admin/products/{id}", svc.handleGetAdminProduct)
	r.Patch("/v1/admin/products/{id}", svc.handlePatchProduct)

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
