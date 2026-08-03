package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
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
func MountHealth(r chi.Router, serviceName string) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		JSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": serviceName,
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

// ListenAndServe starts HTTP and logs the address.
func ListenAndServe(addr, serviceName string, handler http.Handler) error {
	slog.Info("listening", "service", serviceName, "addr", addr)
	return http.ListenAndServe(addr, handler)
}
