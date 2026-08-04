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

const serviceName = "report-service"

//go:embed schema.sql
var schemaFS embed.FS

func main() {
	addr := config.ListenAddr("REPORT_ADDR", ":8087")
	dbPath := config.Get("REPORT_DB", "data/report.db")
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

	svc := &reportService{db: db}

	// Consumers attach once JetStream is reachable; the HTTP server must not
	// wait for the broker or the container never passes its healthcheck.
	var subsMu sync.Mutex
	var subs []*nats.Subscription
	bus := natsx.NewBackground(natsURL)
	bus.Start(func(js nats.JetStreamContext) error {
		started, err := startReportConsumers(js, svc)
		if err != nil {
			return err
		}
		subsMu.Lock()
		subs = started
		subsMu.Unlock()
		return nil
	})
	defer func() {
		bus.Close()
		subsMu.Lock()
		defer subsMu.Unlock()
		for _, sub := range subs {
			_ = sub.Unsubscribe()
		}
	}()

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)
	httpx.MountReady(r, serviceName, httpx.ReadyCheck{Name: "nats", Check: bus.ReadyCheck})

	// Admin dashboard summary (T8.1.2). Authz enforced at gateway when proxied (/v1/admin/*).
	r.Get("/v1/admin/dashboard/summary", svc.handleDashboardSummary)

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
