package httpx

import "testing"

func TestNormalizeListenAddr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{":8083", "0.0.0.0:8083"},
		{"0.0.0.0:8083", "0.0.0.0:8083"},
		{"127.0.0.1:8083", "127.0.0.1:8083"},
		{"", "0.0.0.0:8080"},
		{"  :8081  ", "0.0.0.0:8081"},
	}
	for _, tc := range cases {
		if got := NormalizeListenAddr(tc.in); got != tc.want {
			t.Fatalf("NormalizeListenAddr(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
