package main

import (
	"database/sql"
	"embed"
	"log/slog"
	"net/http"
	"os"

	"gas-tam-de/pkg/config"
	"gas-tam-de/pkg/httpx"
	"gas-tam-de/pkg/sqlite"

	"github.com/go-chi/chi/v5"
)

const serviceName = "api-gateway"

const defaultCORSOrigins = "http://localhost:*,http://127.0.0.1:*"

//go:embed schema.sql
var schemaFS embed.FS

type upstreams struct {
	auth      string
	catalog   string
	geo       string
	order     string
	inventory string
	billing   string
	report    string
}

func main() {
	addr := config.ListenAddr("GATEWAY_ADDR", ":8080")
	jwtSecret := config.Get("JWT_SECRET", "dev-jwt-secret-change-me")
	corsOrigins := parseCORSOrigins(config.Get("CORS_ORIGINS", defaultCORSOrigins))
	rlCfg := rateLimitConfig{
		OTPPerIPPerMinute:     config.GetInt("RATE_LIMIT_OTP_PER_IP_MINUTE", 10),
		LoginPerIPPerMinute:   config.GetInt("RATE_LIMIT_LOGIN_PER_IP_MINUTE", 10),
		OrderPerIPPerMinute:   config.GetInt("RATE_LIMIT_ORDER_PER_IP_MINUTE", 30),
		OrderPerUserPerMinute: config.GetInt("RATE_LIMIT_ORDER_PER_USER_MINUTE", 10),
	}

	dbPath := config.Get("GATEWAY_DB", "data/gateway.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := migrateGateway(db); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	audit := newMultiAuditRecorder(slogAuditRecorder{}, newSQLiteAuditRecorder(db))

	u := upstreams{
		auth:      config.Get("AUTH_SERVICE_URL", "http://127.0.0.1:8081"),
		catalog:   config.Get("CATALOG_SERVICE_URL", "http://127.0.0.1:8082"),
		geo:       config.Get("GEO_SERVICE_URL", "http://127.0.0.1:8083"),
		order:     config.Get("ORDER_SERVICE_URL", "http://127.0.0.1:8084"),
		inventory: config.Get("INVENTORY_SERVICE_URL", "http://127.0.0.1:8085"),
		billing:   config.Get("BILLING_SERVICE_URL", "http://127.0.0.1:8086"),
		report:    config.Get("REPORT_SERVICE_URL", "http://127.0.0.1:8087"),
	}

	r := newGatewayRouter(jwtSecret, corsOrigins, u, newRateLimiters(rlCfg), audit)

	slog.Info("upstream urls",
		"auth", u.auth,
		"catalog", u.catalog,
		"geo", u.geo,
		"order", u.order,
		"inventory", u.inventory,
		"billing", u.billing,
		"report", u.report,
		"cors_origins", corsOrigins,
		"gateway_db", dbPath,
		"rate_limit_otp_per_ip_min", rlCfg.OTPPerIPPerMinute,
		"rate_limit_login_per_ip_min", rlCfg.LoginPerIPPerMinute,
		"rate_limit_order_per_ip_min", rlCfg.OrderPerIPPerMinute,
		"rate_limit_order_per_user_min", rlCfg.OrderPerUserPerMinute,
	)

	if err := httpx.ListenAndServe(addr, serviceName, r); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func migrateGateway(db *sql.DB) error {
	sqlBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}

func newGatewayRouter(jwtSecret string, corsOrigins []string, u upstreams, rl *rateLimiters, audit AuditRecorder) http.Handler {
	if rl == nil {
		rl = newRateLimiters(defaultRateLimitConfig())
	}
	if audit == nil {
		audit = slogAuditRecorder{}
	}

	r := httpx.NewRouter(serviceName)
	r.Use(SecurityHeaders)
	r.Use(CORS(corsOrigins))
	r.Use(stripInboundIdentityHeaders)
	httpx.MountHealth(r, serviceName)

	// Sprint 0: hello endpoint for smoke test (public).
	r.Get("/v1/hello", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"message": "Gas Tam Đệ API Gateway",
			"status":  "ok",
		})
	})

	authProxy := reverseProxy(u.auth)
	catalogProxy := reverseProxy(u.catalog)
	geoProxy := reverseProxy(u.geo)
	orderProxy := reverseProxy(u.order)
	inventoryProxy := reverseProxy(u.inventory)
	billingProxy := reverseProxy(u.billing)
	reportProxy := reverseProxy(u.report)

	r.Route("/v1", func(v1 chi.Router) {
		// Public auth (OTP, admin login, refresh) — no JWT; edge rate limit OTP + login.
		v1.With(RateLimitOTPAndLogin(rl)).Handle("/auth/*", authProxy)

		// Public catalog browse + geo read/search (architecture §4.4).
		v1.Handle("/products", catalogProxy)
		v1.Handle("/products/*", catalogProxy)
		v1.Get("/geo/store", geoProxy.ServeHTTP)
		v1.Get("/geo/search", geoProxy.ServeHTTP)
		v1.Get("/stock/levels", inventoryProxy.ServeHTTP)

		// Customer-authenticated geo check + orders (place-order rate limited by IP + user).
		v1.Group(func(cust chi.Router) {
			cust.Use(RequireJWT(jwtSecret), RequireRole(roleCustomer), RateLimitPlaceOrder(rl))
			cust.Get("/me", authProxy.ServeHTTP)
			cust.Patch("/me", authProxy.ServeHTTP)
			cust.Post("/geo/check", geoProxy.ServeHTTP)
			cust.Handle("/orders", orderProxy)
			cust.Handle("/orders/*", orderProxy)
		})

		// Admin-only — JWT role=admin; audit mutating actions; split by upstream service.
		v1.Group(func(admin chi.Router) {
			admin.Use(RequireJWT(jwtSecret), RequireRole(roleAdmin), AuditAdminMutations(audit))

			admin.Handle("/admin/products", catalogProxy)
			admin.Handle("/admin/products/*", catalogProxy)

			admin.Handle("/admin/geo", geoProxy)
			admin.Handle("/admin/geo/*", geoProxy)

			admin.Handle("/admin/orders", orderProxy)
			admin.Handle("/admin/orders/*", orderProxy)
			admin.Handle("/admin/delivery-fee", orderProxy)
			admin.Handle("/admin/desk-settings", orderProxy)

			admin.Handle("/admin/inventory", inventoryProxy)
			admin.Handle("/admin/inventory/*", inventoryProxy)

			admin.Handle("/admin/debts", billingProxy)
			admin.Handle("/admin/debts/*", billingProxy)

			admin.Handle("/admin/dashboard", reportProxy)
			admin.Handle("/admin/dashboard/*", reportProxy)
		})
	})

	return r
}
