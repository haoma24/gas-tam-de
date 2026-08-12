// Package deploy holds guards over the compose files. There is no Go code to
// ship here — the tests exist so `make test` catches deploy wiring mistakes
// that otherwise only surface as runtime auth failures on the VPS.
package deploy

import (
	"os"
	"strings"
	"testing"
)

// Services that sign or verify the access JWT. auth-service signs, api-gateway
// verifies, and `web` runs an embedded api-gateway (docker-entrypoint.web.sh).
var jwtServices = []string{"auth-service", "api-gateway", "web"}

// TestJWTSecretWiredConsistently pins the fix for customers seeing
// "invalid or expired access token" on the profile screen and
// "Phiên đăng nhập hết hạn" on the orders screen: auth-service used to sign
// with the compiled-in default while the gateways verified with the deployed
// JWT_SECRET, so no amount of client-side token refresh could recover.
func TestJWTSecretWiredConsistently(t *testing.T) {
	services := parseComposeEnv(t, "docker-compose.yml")

	var want string
	for _, name := range jwtServices {
		env, ok := services[name]
		if !ok {
			t.Fatalf("service %q missing from docker-compose.yml", name)
		}
		got, ok := env["JWT_SECRET"]
		if !ok {
			t.Errorf("service %q does not receive JWT_SECRET: it would fall back to the compiled-in default and break every authenticated route", name)
			continue
		}
		if want == "" {
			want = got
			continue
		}
		if got != want {
			t.Errorf("service %q resolves JWT_SECRET as %q, want %q — signer and verifier must share one value", name, got, want)
		}
	}
}

// TestAuthServiceHasTokenLifetimes keeps refresh/access lifetimes tunable from
// deploy/.env instead of silently using the compiled-in defaults.
func TestAuthServiceHasTokenLifetimes(t *testing.T) {
	env := parseComposeEnv(t, "docker-compose.yml")["auth-service"]
	for _, key := range []string{"JWT_ACCESS_TTL_SEC", "JWT_REFRESH_TTL_SEC"} {
		if _, ok := env[key]; !ok {
			t.Errorf("auth-service does not receive %s", key)
		}
	}
}

// TestOrderServiceUsesComposeInventoryURL prevents order-service from falling
// back to 127.0.0.1:8085 inside its own container. That fallback makes every
// checkout fail with INVENTORY_UNAVAILABLE even while inventory-service is up.
func TestOrderServiceUsesComposeInventoryURL(t *testing.T) {
	env := parseComposeEnv(t, "docker-compose.yml")["order-service"]
	const want = "http://inventory-service:8085"
	if got := env["INVENTORY_SERVICE_URL"]; got != want {
		t.Errorf("order-service INVENTORY_SERVICE_URL=%q, want %q", got, want)
	}
}

// TestOTPDevRevealOffByDefault keeps the deploy file from handing out OTP codes
// in API responses. With OTP_DEV_REVEAL=1, /v1/auth/otp/request returns the
// code, so anyone who can reach the endpoint logs in as any phone number —
// real SMS delivery does not close that hole, only this default does.
func TestOTPDevRevealOffByDefault(t *testing.T) {
	env := parseComposeEnv(t, "docker-compose.yml")["auth-service"]
	const want = "${OTP_DEV_REVEAL:-0}"
	if got := env["OTP_DEV_REVEAL"]; got != want {
		t.Errorf("auth-service OTP_DEV_REVEAL=%q, want %q — the deploy default must not reveal codes", got, want)
	}
}

// TestOTPDevRevealOnLocally pins the other half: `make compose-up` keeps
// returning dev_code, so day-to-day development needs no SMS vendor.
//
// Note this override is not local-only — the GCP stag deploy merges this file
// as well, so staging reveals codes until deploy/.env on the VM sets 0.
func TestOTPDevRevealOnLocally(t *testing.T) {
	env := parseComposeEnv(t, "docker-compose.local.yml")["auth-service"]
	const want = "${OTP_DEV_REVEAL:-1}"
	if got := env["OTP_DEV_REVEAL"]; got != want {
		t.Errorf("local override OTP_DEV_REVEAL=%q, want %q", got, want)
	}
}

// TestPhoneSecretsNotWired guards the deliberate omission documented in
// docker-compose.yml: phone_hash is the customer identity key, so wiring a
// pepper that differs from the one already used by a live auth.db would orphan
// every existing user and their order history.
func TestPhoneSecretsNotWired(t *testing.T) {
	env := parseComposeEnv(t, "docker-compose.yml")["auth-service"]
	for _, key := range []string{"PHONE_HASH_PEPPER", "PHONE_ENC_KEY"} {
		if _, ok := env[key]; ok {
			t.Errorf("auth-service now receives %s — re-keying phone_hash on a live auth.db orphans existing users; add a migration first", key)
		}
	}
}

// parseComposeEnv reads the `environment:` mapping of every compose service.
// Compose files here always use the nested-mapping form with two-space indents,
// which keeps this simple enough to avoid a YAML dependency.
func parseComposeEnv(t *testing.T, path string) map[string]map[string]string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	services := map[string]map[string]string{}
	inServices := false
	current := ""
	inEnv := false

	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))

		if indent == 0 {
			inServices = strings.HasPrefix(line, "services:")
			current, inEnv = "", false
			continue
		}
		if !inServices {
			continue
		}
		if indent == 2 {
			current = strings.TrimSuffix(strings.TrimSpace(line), ":")
			services[current] = map[string]string{}
			inEnv = false
			continue
		}
		if indent == 4 {
			inEnv = strings.TrimSpace(line) == "environment:"
			continue
		}
		if indent == 6 && inEnv && current != "" {
			key, value, found := strings.Cut(strings.TrimSpace(line), ":")
			if !found {
				continue
			}
			services[current][key] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}

	if len(services) == 0 {
		t.Fatalf("no services parsed from %s", path)
	}
	return services
}
