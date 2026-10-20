package syncclient

import "testing"

func TestServerURLTakesHTTPSAndLoopbackHTTPOnly(t *testing.T) {
	ok := map[string]string{
		"https://todo.sri-ai.xyz/": "https://todo.sri-ai.xyz",
		" https://a.example:8443 ": "https://a.example:8443",
		"http://127.0.0.1:18080":   "http://127.0.0.1:18080",
		"http://localhost:8080/":   "http://localhost:8080",
		"http://[::1]:9000":        "http://[::1]:9000",
	}
	for in, want := range ok {
		got, err := ServerURL(in)
		if err != nil || got != want {
			t.Errorf("ServerURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "todo.sri-ai.xyz", "http://todo.sri-ai.xyz", "http://192.168.1.5", "ftp://a.example", "https://"} {
		if got, err := ServerURL(in); err == nil {
			t.Errorf("ServerURL(%q) = %q, want an error", in, got)
		}
	}
}
