package collector

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/NiekKeijzer/komodo-exporter/komodo"
)

// targetGroup is an entry of Prometheus HTTP service discovery.
type targetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// ServersSD serves Prometheus HTTP service discovery with one target per
// enabled Komodo server: its name, followed by "."+domain unless domain is
// empty, labelled server=<name>. The scrape config adds the port.
func ServersSD(client *komodo.Client, domain string, timeout time.Duration, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		servers, err := client.ListServers(ctx)
		if err != nil {
			// Prometheus keeps the targets it had when discovery fails.
			logger.Error("listing Komodo servers failed", "err", err)
			http.Error(w, "listing Komodo servers failed", http.StatusBadGateway)
			return
		}
		groups := []targetGroup{}
		for _, srv := range servers {
			if srv.Info.State == komodo.ServerDisabled {
				continue
			}
			target := srv.Name
			if domain != "" {
				target += "." + domain
			}
			groups = append(groups, targetGroup{Targets: []string{target}, Labels: map[string]string{"server": srv.Name}})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(groups); err != nil {
			logger.Error("writing service discovery response failed", "err", err)
		}
	})
}
