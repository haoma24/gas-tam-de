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

	svc := &reportService{db: db}
	subs, err := startReportConsumers(js, svc)
	if err != nil {
		slog.Error("start report consumers", "err", err)
		os.Exit(1)
	}
	defer func() {
		for _, sub := range subs {
			_ = sub.Unsubscribe()
		}
	}()

	r := httpx.NewRouter(serviceName)
	httpx.MountHealth(r, serviceName)

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
