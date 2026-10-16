// Package collector exposes Komodo's state as Prometheus metrics.
package collector

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/NiekKeijzer/komodo-exporter/komodo"
)

// bytesPerGB converts Komodo's GiB to bytes.
const bytesPerGB = 1 << 30

// States are exported one-hot: one series per known state, 1 for the
// current one. The lists follow Komodo v2.3, in snake_case. A state missing
// from a list is exported too, so a newer Komodo shows up rather than vanishes.
var (
	serverStates    = []string{"ok", "not_ok", "disabled"}
	stackStates     = []string{"deploying", "running", "paused", "stopped", "created", "restarting", "dead", "removing", "unhealthy", "down", "unknown"}
	containerStates = []string{"running", "created", "restarting", "stopping", "removing", "paused", "exited", "dead", "unknown"}
	healthStatuses  = []string{"healthy", "unhealthy", "starting", "none"}
	syncStates      = []string{"syncing", "pending", "ok", "failed", "unknown"}
	procedureStates = []string{"running", "ok", "failed", "unknown"}
	repoStates      = []string{"cloning", "pulling", "building", "ok", "failed", "unknown"}
)

var (
	upDesc             = desc("up", "Whether the last query of the Komodo API succeeded.")
	scrapeDurationDesc = desc("scrape_duration_seconds", "How long the last query of the Komodo API took.")

	serverStateDesc     = desc("server_state", "Server state, one series per state.", "server", "state")
	serverInfoDesc      = desc("server_info", "Periphery version of the server.", "server", "version")
	serverCPUDesc       = desc("server_cpu_usage_percent", "CPU usage of the server.", "server")
	serverMemUsedDesc   = desc("server_memory_used_bytes", "Memory used on the server.", "server")
	serverMemTotalDesc  = desc("server_memory_total_bytes", "Memory of the server.", "server")
	serverDiskUsedDesc  = desc("server_disk_used_bytes", "Disk space used on a mount of the server.", "server", "mount")
	serverDiskTotalDesc = desc("server_disk_total_bytes", "Disk space of a mount of the server.", "server", "mount")

	stackStateDesc          = desc("stack_state", "Stack state, one series per state.", "stack", "server", "state")
	stackProjectMissingDesc = desc("stack_project_missing", "Whether the stack's compose project is missing from its server.", "stack", "server")

	containerStateDesc  = desc("container_state", "Container state, one series per state.", "server", "container", "state")
	containerHealthDesc = desc("container_health_status", `Container health check status, one series per status; "none" without a health check.`, "server", "container", "status")
	containerCPUDesc    = desc("container_cpu_usage_percent", "CPU usage of the container, as docker stats reports it.", "server", "container")
	containerMemDesc    = desc("container_memory_usage_bytes", "Memory used by the container, as docker stats reports it.", "server", "container")

	syncStateDesc      = desc("resource_sync_state", "Resource sync state, one series per state.", "sync", "state")
	procedureStateDesc = desc("procedure_state", "State of the procedure's last run, one series per state.", "procedure", "state")
	repoStateDesc      = desc("repo_state", "Repo state, one series per state.", "repo", "state")
)

func desc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc("komodo_"+name, help, labels, nil)
}

// Collector queries the Komodo API on every scrape.
type Collector struct {
	client  *komodo.Client
	timeout time.Duration
	logger  *slog.Logger
}

// New returns a collector that gives each scrape timeout to query Komodo.
func New(client *komodo.Client, timeout time.Duration, logger *slog.Logger) *Collector {
	return &Collector{client: client, timeout: timeout, logger: logger}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// snapshot is everything one scrape reads from Komodo.
type snapshot struct {
	servers    []komodo.Server
	stats      map[string]komodo.SystemStats // by server name
	stacks     []komodo.Stack
	containers []komodo.Container
	syncs      []komodo.Resource
	procedures []komodo.Resource
	repos      []komodo.Resource
}

// Collect implements prometheus.Collector. It exports nothing but
// komodo_up 0 when a query fails, rather than a partial view.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	s, err := c.fetch(ctx)
	up := 1.0
	if err != nil {
		c.logger.Error("querying Komodo failed", "err", err)
		up = 0
	} else {
		export(ch, s)
	}
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, up)
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(start).Seconds())
}

func (c *Collector) fetch(ctx context.Context) (*snapshot, error) {
	s := &snapshot{stats: map[string]komodo.SystemStats{}}
	var err error
	if s.servers, err = c.client.ListServers(ctx); err != nil {
		return nil, err
	}
	for _, srv := range s.servers {
		if srv.Info.State != komodo.ServerOk {
			continue
		}
		// A server can disconnect after ListServers: skip its stats, its
		// state shows it on the next scrape.
		stats, err := c.client.GetSystemStats(ctx, srv.ID)
		if err != nil {
			c.logger.Warn("no system stats", "server", srv.Name, "err", err)
			continue
		}
		s.stats[srv.Name] = stats
	}
	if s.stacks, err = c.client.ListStacks(ctx); err != nil {
		return nil, err
	}
	if s.containers, err = c.client.ListAllContainers(ctx); err != nil {
		return nil, err
	}
	if s.syncs, err = c.client.ListResourceSyncs(ctx); err != nil {
		return nil, err
	}
	if s.procedures, err = c.client.ListProcedures(ctx); err != nil {
		return nil, err
	}
	if s.repos, err = c.client.ListRepos(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func export(ch chan<- prometheus.Metric, s *snapshot) {
	gauge := func(d *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
	}
	oneHot := func(d *prometheus.Desc, states []string, current string, labels ...string) {
		current = snakeCase(current)
		if current == "" {
			current = "unknown"
		}
		known := false
		for _, state := range states {
			v := 0.0
			if state == current {
				v, known = 1, true
			}
			gauge(d, v, append(labels, state)...)
		}
		if !known {
			gauge(d, 1, append(labels, current)...)
		}
	}

	for _, srv := range s.servers {
		oneHot(serverStateDesc, serverStates, srv.Info.State, srv.Name)
		gauge(serverInfoDesc, 1, srv.Name, srv.Info.Version)
		stats, ok := s.stats[srv.Name]
		if !ok {
			continue
		}
		gauge(serverCPUDesc, stats.CPUPerc, srv.Name)
		gauge(serverMemUsedDesc, stats.MemUsedGB*bytesPerGB, srv.Name)
		gauge(serverMemTotalDesc, stats.MemTotalGB*bytesPerGB, srv.Name)
		for _, d := range stats.Disks {
			gauge(serverDiskUsedDesc, d.UsedGB*bytesPerGB, srv.Name, d.Mount)
			gauge(serverDiskTotalDesc, d.TotalGB*bytesPerGB, srv.Name, d.Mount)
		}
	}

	for _, st := range s.stacks {
		oneHot(stackStateDesc, stackStates, st.Info.State, st.Name, st.Info.ServerName)
		gauge(stackProjectMissingDesc, boolValue(st.Info.ProjectMissing), st.Name, st.Info.ServerName)
	}

	for _, ct := range s.containers {
		oneHot(containerStateDesc, containerStates, ct.State, ct.ServerName, ct.Name)
		oneHot(containerHealthDesc, healthStatuses, healthStatus(ct.Status), ct.ServerName, ct.Name)
		if ct.Stats == nil {
			continue
		}
		if v, ok := parsePercent(ct.Stats.CPUPerc); ok {
			gauge(containerCPUDesc, v, ct.ServerName, ct.Name)
		}
		if v, ok := parseMemUsage(ct.Stats.MemUsage); ok {
			gauge(containerMemDesc, v, ct.ServerName, ct.Name)
		}
	}

	for _, r := range s.syncs {
		oneHot(syncStateDesc, syncStates, r.Info.State, r.Name)
	}
	for _, r := range s.procedures {
		oneHot(procedureStateDesc, procedureStates, r.Info.State, r.Name)
	}
	for _, r := range s.repos {
		oneHot(repoStateDesc, repoStates, r.Info.State, r.Name)
	}
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// snakeCase turns Komodo's variant names ("NotOk") into label values
// ("not_ok"). Values that are already snake_case pass unchanged.
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
