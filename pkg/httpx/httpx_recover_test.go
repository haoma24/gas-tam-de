package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gas-tam-de/pkg/httpx"
)

func TestSafeRecover_ReturnsGenericJSON(t *testing.T) {
	r := httpx.NewRouter("test-svc")
	r.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("secret internal detail /tmp/stack")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	var parsed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v body=%s", err, body)
	}
	errObj, _ := parsed["error"].(map[string]any)
	if errObj["code"] != "INTERNAL_ERROR" {
		t.Fatalf("code=%v", errObj["code"])
	}
	if errObj["message"] != "internal server error" {
		t.Fatalf("message=%v", errObj["message"])
	}
	lower := strings.ToLower(body)
	for _, leak := range []string{"secret internal", "/tmp/stack", "goroutine", "panic:"} {
		if strings.Contains(lower, leak) {
			t.Fatalf("response leaked %q: %s", leak, body)
		}
	}
}
