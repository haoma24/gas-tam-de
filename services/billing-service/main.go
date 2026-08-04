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

const serviceName = "billing-service"

//go:embed schema.sql
var schemaFS embed.FS

type billingService struct {
	db  *sql.DB
	bus billingEventPublisher
}

func main() {
	addr := config.ListenAddr("BILLING_ADDR", ":8086")
	dbPath := config.Get("BILLING_DB", "data/billing.db")
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

	svc := &billingService{db: db, bus: newJSBillingPublisher(js)}

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)

	// Internal: order-service records payment after admin complete (T6.1.2).
	// Publishes billing.payment.recorded + billing.debt.updated (T6.1.3).
	r.Post("/v1/internal/payments", svc.handleRecordPayment)
	// Admin: list/aggregate customer debts (T6.2.1). Gateway RBAC: /v1/admin/*.
	r.Get("/v1/admin/debts", svc.handleListDebts)

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
