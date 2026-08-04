package main

import (
	"database/sql"
	"embed"
	"log/slog"
	"os"
	"sync"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/natsx"
	"gas-tam-de/pkg/sqlite"

	"github.com/nats-io/nats.go"
)

const serviceName = "inventory-service"

//go:embed schema.sql
var schemaFS embed.FS

func main() {
	addr := config.ListenAddr("INVENTORY_ADDR", ":8085")
	dbPath := config.Get("INVENTORY_DB", "data/inventory.db")
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

	if err := seedInventoryDefaults(db, loadInventorySeedConfig()); err != nil {
		slog.Error("seed inventory", "err", err)
		os.Exit(1)
	}

	svc := &inventoryService{db: db}

	// The consumer attaches once JetStream is reachable; the HTTP server must
	// not wait for the broker or the container never passes its healthcheck.
	var subMu sync.Mutex
	var sub *nats.Subscription
	bus := natsx.NewBackground(natsURL)
	bus.Start(func(js nats.JetStreamContext) error {
		started, err := startOrderCompletedConsumer(js, svc)
		if err != nil {
			return err
		}
		subMu.Lock()
		sub = started
		subMu.Unlock()
		return nil
	})
	defer func() {
		bus.Close()
		subMu.Lock()
		defer subMu.Unlock()
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}()

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)
	httpx.MountReady(r, serviceName, httpx.ReadyCheck{Name: "nats", Check: bus.ReadyCheck})

	// Admin inventory (T7.1.2). Authz enforced at gateway when proxied (/v1/admin/*).
	r.Get("/v1/admin/inventory", svc.handleListStock)
	r.Post("/v1/admin/inventory", svc.handlePostMovement)
	r.Get("/v1/stock/levels", svc.handleListStockLevels)
	r.Post("/v1/internal/stock/reserve", svc.handleReserveStock)
	r.Post("/v1/internal/stock/release", svc.handleReleaseStock)

	slog.Info("upstream urls", "nats", natsURL)

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
