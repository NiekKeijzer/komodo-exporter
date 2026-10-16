package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func lookup(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := env[name]
		return v, ok
	}
}

var fullEnv = map[string]string{
	"KOMODO_URL":        "http://komodo:9120",
	"KOMODO_API_KEY":    "key",
	"KOMODO_API_SECRET": "secret",
}

func TestSetup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h, listen, err := setup([]string{"-listen", ":1234"}, lookup(fullEnv), logger, io.Discard)
	if err != nil || listen != ":1234" {
		t.Fatalf("setup = %q, %v, want :1234, nil", listen, err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(rec.Body.String(), "komodo_exporter_build_info{") {
		t.Errorf("/metrics lacks komodo_exporter_build_info:\n%s", rec.Body)
	}

	if _, _, err := setup([]string{"extra"}, lookup(fullEnv), logger, io.Discard); !errors.Is(err, errUsage) {
		t.Errorf("setup with an argument: err = %v, want errUsage", err)
	}

	for name := range fullEnv {
		env := map[string]string{}
		for k, v := range fullEnv {
			if k != name {
				env[k] = v
			}
		}
		if _, _, err := setup(nil, lookup(env), logger, io.Discard); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("setup without %s: err = %v, want it named", name, err)
		}
	}
}

func TestVersion(t *testing.T) {
	var stdout strings.Builder
	if code := run(context.Background(), []string{"-version"}, lookup(nil), &stdout, io.Discard); code != 0 {
		t.Errorf("run -version = %d, want 0", code)
	}
	if !strings.HasPrefix(stdout.String(), "komodo-exporter, version") {
		t.Errorf("run -version printed %q", stdout.String())
	}
}
