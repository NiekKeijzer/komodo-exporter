package collector

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"

	"github.com/NiekKeijzer/komodo-exporter/komodo"
)

// fakeKomodo serves testdata/<Request>.json for POST /read/<Request>, or
// testdata/GetSystemStats-<server>.json, after checking the API key. A
// request named in fail gets a 500.
func fakeKomodo(t *testing.T, fail ...string) *komodo.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request, ok := strings.CutPrefix(r.URL.Path, "/read/")
		if r.Method != http.MethodPost || !ok {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Api-Key") != "key" || r.Header.Get("X-Api-Secret") != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var params map[string]any
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Errorf("%s: decoding params: %v", request, err)
		}
		name := request
		if request == "GetSystemStats" {
			name += "-" + params["server"].(string)
		} else if params["limit"] != 0.0 {
			t.Errorf("%s: limit = %v, want 0 for all results", request, params["limit"])
		}
		for _, f := range fail {
			if f == name {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
		}
		data, err := os.ReadFile(filepath.Join("testdata", name+".json"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return komodo.NewClient(srv.URL+"/", "key", "secret", srv.Client())
}

func newCollector(client *komodo.Client) *Collector {
	return New(client, 5*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestCollect(t *testing.T) {
	c := newCollector(fakeKomodo(t))
	const want = `
# HELP komodo_up Whether the last query of the Komodo API succeeded.
# TYPE komodo_up gauge
komodo_up 1
# HELP komodo_server_state Server state, one series per state.
# TYPE komodo_server_state gauge
komodo_server_state{server="komodo-01",state="disabled"} 0
komodo_server_state{server="komodo-01",state="not_ok"} 0
komodo_server_state{server="komodo-01",state="ok"} 1
komodo_server_state{server="old-01",state="disabled"} 1
komodo_server_state{server="old-01",state="not_ok"} 0
komodo_server_state{server="old-01",state="ok"} 0
komodo_server_state{server="periphery-01",state="disabled"} 0
komodo_server_state{server="periphery-01",state="not_ok"} 0
komodo_server_state{server="periphery-01",state="ok"} 1
komodo_server_state{server="periphery-02",state="disabled"} 0
komodo_server_state{server="periphery-02",state="not_ok"} 1
komodo_server_state{server="periphery-02",state="ok"} 0
# HELP komodo_server_info Periphery version of the server.
# TYPE komodo_server_info gauge
komodo_server_info{server="komodo-01",version="v2.3.3"} 1
komodo_server_info{server="old-01",version=""} 1
komodo_server_info{server="periphery-01",version="v2.3.3"} 1
komodo_server_info{server="periphery-02",version=""} 1
# HELP komodo_server_cpu_usage_percent CPU usage of the server.
# TYPE komodo_server_cpu_usage_percent gauge
komodo_server_cpu_usage_percent{server="komodo-01"} 12.5
komodo_server_cpu_usage_percent{server="periphery-01"} 50
# HELP komodo_server_memory_total_bytes Memory of the server.
# TYPE komodo_server_memory_total_bytes gauge
komodo_server_memory_total_bytes{server="komodo-01"} 4.294967296e+09
komodo_server_memory_total_bytes{server="periphery-01"} 8.589934592e+09
# HELP komodo_server_disk_used_bytes Disk space used on a mount of the server.
# TYPE komodo_server_disk_used_bytes gauge
komodo_server_disk_used_bytes{mount="/etc/hostname",server="komodo-01"} 3.221225472e+09
komodo_server_disk_used_bytes{mount="/mnt/data",server="komodo-01"} 8.589934592e+09
komodo_server_disk_used_bytes{mount="/mnt/data",server="periphery-01"} 1.7179869184e+10
# HELP komodo_stack_project_missing Whether the stack's compose project is missing from its server.
# TYPE komodo_stack_project_missing gauge
komodo_stack_project_missing{server="periphery-01",stack="edge"} 0
komodo_stack_project_missing{server="periphery-01",stack="monitoring"} 1
komodo_stack_project_missing{server="periphery-01",stack="security"} 0
# HELP komodo_container_cpu_usage_percent CPU usage of the container, as docker stats reports it.
# TYPE komodo_container_cpu_usage_percent gauge
komodo_container_cpu_usage_percent{container="edge-traefik-1",server="periphery-01"} 1.5
# HELP komodo_container_memory_usage_bytes Memory used by the container, as docker stats reports it.
# TYPE komodo_container_memory_usage_bytes gauge
komodo_container_memory_usage_bytes{container="edge-traefik-1",server="periphery-01"} 6.5536e+07
# HELP komodo_resource_sync_state Resource sync state, one series per state.
# TYPE komodo_resource_sync_state gauge
komodo_resource_sync_state{state="failed",sync="homelab"} 0
komodo_resource_sync_state{state="ok",sync="homelab"} 0
komodo_resource_sync_state{state="pending",sync="homelab"} 1
komodo_resource_sync_state{state="syncing",sync="homelab"} 0
komodo_resource_sync_state{state="unknown",sync="homelab"} 0
# HELP komodo_procedure_state State of the procedure's last run, one series per state.
# TYPE komodo_procedure_state gauge
komodo_procedure_state{procedure="homelab-reconcile",state="failed"} 1
komodo_procedure_state{procedure="homelab-reconcile",state="ok"} 0
komodo_procedure_state{procedure="homelab-reconcile",state="running"} 0
komodo_procedure_state{procedure="homelab-reconcile",state="unknown"} 0
# HELP komodo_repo_state Repo state, one series per state.
# TYPE komodo_repo_state gauge
komodo_repo_state{repo="homelab",state="building"} 0
komodo_repo_state{repo="homelab",state="cloning"} 0
komodo_repo_state{repo="homelab",state="failed"} 0
komodo_repo_state{repo="homelab",state="ok"} 1
komodo_repo_state{repo="homelab",state="pulling"} 0
komodo_repo_state{repo="homelab",state="unknown"} 0
`
	err := testutil.CollectAndCompare(c, strings.NewReader(want),
		"komodo_up",
		"komodo_server_state", "komodo_server_info", "komodo_server_cpu_usage_percent",
		"komodo_server_memory_total_bytes", "komodo_server_disk_used_bytes",
		"komodo_stack_project_missing",
		"komodo_container_cpu_usage_percent", "komodo_container_memory_usage_bytes",
		"komodo_resource_sync_state", "komodo_procedure_state", "komodo_repo_state",
	)
	if err != nil {
		t.Error(err)
	}
}

// TestCollectStates checks the current state of each stack and container,
// leaving out the zero series of the other states.
func TestCollectStates(t *testing.T) {
	c := newCollector(fakeKomodo(t))
	out, err := testutil.CollectAndFormat(c, expfmt.TypeTextPlain, "komodo_stack_state", "komodo_container_state", "komodo_container_health_status")
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{
		`komodo_stack_state{server="periphery-01",stack="edge",state="running"} 1`,
		`komodo_stack_state{server="periphery-01",stack="security",state="unhealthy"} 1`,
		`komodo_stack_state{server="periphery-01",stack="monitoring",state="down"} 1`,
		`komodo_container_state{container="edge-traefik-1",server="periphery-01",state="running"} 1`,
		`komodo_container_state{container="security-authentik-db-init-1",server="periphery-01",state="exited"} 1`,
		`komodo_container_state{container="edge-whoami-1",server="periphery-01",state="restarting"} 1`,
		`komodo_container_health_status{container="edge-traefik-1",server="periphery-01",status="healthy"} 1`,
		`komodo_container_health_status{container="security-authentik-1",server="periphery-01",status="unhealthy"} 1`,
		`komodo_container_health_status{container="edge-whoami-1",server="periphery-01",status="none"} 1`,
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("missing %s", want)
		}
	}
	if n := strings.Count(got, "} 1\n"); n != 11 {
		t.Errorf("%d series are 1, want 11: one state per stack and container, one health status per container\n%s", n, got)
	}
}

func TestCollectFailedQuery(t *testing.T) {
	c := newCollector(fakeKomodo(t, "ListStacks"))
	const want = `
# HELP komodo_up Whether the last query of the Komodo API succeeded.
# TYPE komodo_up gauge
komodo_up 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(want), "komodo_up", "komodo_server_state"); err != nil {
		t.Error(err)
	}
}

func TestCollectMissingSystemStats(t *testing.T) {
	c := newCollector(fakeKomodo(t, "GetSystemStats-srv2"))
	const want = `
# HELP komodo_up Whether the last query of the Komodo API succeeded.
# TYPE komodo_up gauge
komodo_up 1
# HELP komodo_server_cpu_usage_percent CPU usage of the server.
# TYPE komodo_server_cpu_usage_percent gauge
komodo_server_cpu_usage_percent{server="komodo-01"} 12.5
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(want), "komodo_up", "komodo_server_cpu_usage_percent"); err != nil {
		t.Error(err)
	}
}

// snapshotCollector exports a fixed snapshot.
type snapshotCollector struct{ s *snapshot }

func (c snapshotCollector) Describe(ch chan<- *prometheus.Desc) { prometheus.DescribeByCollect(c, ch) }
func (c snapshotCollector) Collect(ch chan<- prometheus.Metric) { export(ch, c.s) }

func TestCollectUnknownState(t *testing.T) {
	s := &snapshot{repos: []komodo.Resource{{Name: "homelab"}}}
	s.repos[0].Info.State = "Migrating"
	const want = `
# HELP komodo_repo_state Repo state, one series per state.
# TYPE komodo_repo_state gauge
komodo_repo_state{repo="homelab",state="building"} 0
komodo_repo_state{repo="homelab",state="cloning"} 0
komodo_repo_state{repo="homelab",state="failed"} 0
komodo_repo_state{repo="homelab",state="migrating"} 1
komodo_repo_state{repo="homelab",state="ok"} 0
komodo_repo_state{repo="homelab",state="pulling"} 0
komodo_repo_state{repo="homelab",state="unknown"} 0
`
	if err := testutil.CollectAndCompare(snapshotCollector{s}, strings.NewReader(want), "komodo_repo_state"); err != nil {
		t.Error(err)
	}
}

func TestSnakeCase(t *testing.T) {
	for in, want := range map[string]string{"NotOk": "not_ok", "Ok": "ok", "running": "running", "not_ok": "not_ok", "": ""} {
		if got := snakeCase(in); got != want {
			t.Errorf("snakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}
