package httpx

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter returns a Chi router with request-id, real-ip, safe recover, and request logging.
func NewRouter(serviceName string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(SafeRecover)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)
			// Container health probes hit /healthz every few seconds; logging
			// the successful ones buries real traffic. Failures still surface.
			if req.URL.Path == "/healthz" && ww.Status() == http.StatusOK {
				return
			}
			slog.Info("http",
				"service", serviceName,
				"method", req.Method,
				"path", req.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(req.Context()),
			)
		})
	})
	return r
}

// SafeRecover recovers panics, logs the stack server-side only, and returns a generic JSON 500.
// Replaces chi's Recoverer so clients never see stack traces or panic strings.
func SafeRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				slog.Error("panic recovered",
					"err", rec,
					"request_id", middleware.GetReqID(r.Context()),
					"stack", string(debug.Stack()),
				)
				if r.Header.Get("Connection") != "Upgrade" {
					Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// MountHealth registers GET /healthz.
//
// This is a liveness probe: it answers as soon as the process serves HTTP and
// never depends on brokers or other services, so container healthchecks do not
// fail because a dependency is still starting.
func MountHealth(r chi.Router, serviceName string) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		JSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": serviceName,
		})
	})
}

// ReadyCheck reports whether one dependency is usable. A nil error means ready.
type ReadyCheck struct {
	Name  string
	Check func() error
}

// MountReady registers GET /readyz, which reports dependency state:
// 200 when every check passes, else 503 with the failing dependency names.
func MountReady(r chi.Router, serviceName string, checks ...ReadyCheck) {
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		deps := make(map[string]string, len(checks))
		ready := true
		for _, c := range checks {
			if c.Check == nil {
				continue
			}
			if err := c.Check(); err != nil {
				ready = false
				deps[c.Name] = err.Error()
				continue
			}
			deps[c.Name] = "ok"
		}
		status := http.StatusOK
		state := "ready"
		if !ready {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}
		JSON(w, status, map[string]any{
			"status":       state,
			"service":      serviceName,
			"dependencies": deps,
		})
	})
}

// JSON writes a JSON response.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Error writes a JSON error payload.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// NormalizeListenAddr rewrites bare ":port" to "0.0.0.0:port".
//
// Go's default Listen(":port") can bind IPv6-only on hosts with
// net.ipv6.bindv6only=1. Docker healthchecks probe 127.0.0.1, so the
// container looks unhealthy even though the process is up. Binding IPv4
// explicitly keeps compose probes working on those VPS kernels.
func NormalizeListenAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "0.0.0.0:8080"
	}
	if strings.HasPrefix(addr, ":") && !strings.HasPrefix(addr, "::") {
		return "0.0.0.0" + addr
	}
	return addr
}

// ListenAndServe starts HTTP and logs the address.
// Addresses normalized to 0.0.0.0:* use the tcp4 network so probes to
// 127.0.0.1 succeed even when the host has net.ipv6.bindv6only=1.
func ListenAndServe(addr, serviceName string, handler http.Handler) error {
	addr = NormalizeListenAddr(addr)
	network := "tcp"
	if strings.HasPrefix(addr, "0.0.0.0:") {
		network = "tcp4"
	}
	ln, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	slog.Info("listening", "service", serviceName, "addr", addr, "network", network)
	return http.Serve(ln, handler)
}
