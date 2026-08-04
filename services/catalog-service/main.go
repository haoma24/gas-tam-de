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

	nc, js, err := natsx.ConnectJS(natsURL)
	if err != nil {
		slog.Error("nats connect", "url", natsURL, "err", err)
		os.Exit(1)
	}
	defer nc.Close()

	if err := natsx.EnsureStreams(js); err != nil {
		slog.Error("ensure jetstream streams", "err", err)
		os.Exit(1)
	}

	svc := &catalogService{db: db, bus: newJSProductPublisher(js)}

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)

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
