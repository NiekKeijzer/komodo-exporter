package collector

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func serveSD(t *testing.T, h http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sd/servers", nil))
	return rec
}

func TestServersSD(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for domain, want := range map[string]string{
		"node.example.com": `[{"targets":["komodo-01.node.example.com"],"labels":{"server":"komodo-01"}},` +
			`{"targets":["periphery-01.node.example.com"],"labels":{"server":"periphery-01"}},` +
			`{"targets":["periphery-02.node.example.com"],"labels":{"server":"periphery-02"}}]`,
		"": `[{"targets":["komodo-01"],"labels":{"server":"komodo-01"}},` +
			`{"targets":["periphery-01"],"labels":{"server":"periphery-01"}},` +
			`{"targets":["periphery-02"],"labels":{"server":"periphery-02"}}]`,
	} {
		rec := serveSD(t, ServersSD(fakeKomodo(t), domain, 5*time.Second, logger))
		if rec.Code != http.StatusOK {
			t.Fatalf("domain %q: status %d, want 200", domain, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("domain %q: Content-Type %q, want application/json", domain, ct)
		}
		// The disabled old-01 is left out; the unreachable periphery-02 is not.
		if got := strings.TrimSpace(rec.Body.String()); got != want {
			t.Errorf("domain %q:\ngot  %s\nwant %s", domain, got, want)
		}
	}
}

func TestServersSDFailedQuery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := serveSD(t, ServersSD(fakeKomodo(t, "ListServers"), "", 5*time.Second, logger))
	if rec.Code != http.StatusBadGateway {
		t.Errorf("status %d, want 502", rec.Code)
	}
}
