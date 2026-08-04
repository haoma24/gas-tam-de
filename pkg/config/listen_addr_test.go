package config

import "testing"

func TestListenAddrOrder(t *testing.T) {
	t.Setenv("GEO_ADDR", "")
	t.Setenv("PORT", "")

	if got := ListenAddr("GEO_ADDR", ":8083"); got != ":8083" {
		t.Fatalf("fallback: got %q", got)
	}

	t.Setenv("PORT", "9090")
	if got := ListenAddr("GEO_ADDR", ":8083"); got != ":9090" {
		t.Fatalf("PORT bare: got %q", got)
	}

	t.Setenv("PORT", ":9091")
	if got := ListenAddr("GEO_ADDR", ":8083"); got != ":9091" {
		t.Fatalf("PORT with colon: got %q", got)
	}

	t.Setenv("GEO_ADDR", ":8083")
	t.Setenv("PORT", "9090")
	if got := ListenAddr("GEO_ADDR", ":8080"); got != ":8083" {
		t.Fatalf("primary wins: got %q", got)
	}
}
