// Command komodo-exporter exposes the state of a Komodo Core as Prometheus
// metrics on /metrics, and its servers as HTTP service discovery on
// /sd/servers.
//
//	komodo-exporter [-listen addr] [-version]
//
// It reads KOMODO_URL, KOMODO_API_KEY and KOMODO_API_SECRET, and optionally
// KOMODO_EXPORTER_SD_DOMAIN, from the environment.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"

	"github.com/NiekKeijzer/komodo-exporter/collector"
	"github.com/NiekKeijzer/komodo-exporter/komodo"
)

// queryTimeout bounds the Komodo queries of one scrape or discovery request.
const queryTimeout = 10 * time.Second

// errUsage marks errors in the command line.
var errUsage = errors.New("usage")

// errVersion marks a request for the version, which is printed instead of
// serving.
var errVersion = errors.New("version")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, lookupEnv func(string) (string, bool), stdout, stderr io.Writer) int {
	logger := slog.New(slog.NewJSONHandler(stderr, nil))
	h, listen, err := setup(args, lookupEnv, logger, stderr)
	if err == nil {
		logger.Info("listening", "addr", listen)
		err = serve(ctx, listen, h, logger)
	}
	switch {
	case errors.Is(err, errVersion):
		fmt.Fprintln(stdout, version.Print("komodo-exporter"))
	case errors.Is(err, errUsage):
		fmt.Fprintln(stderr, "usage: komodo-exporter [-listen addr] [-version]")
		return 2
	case err != nil:
		fmt.Fprintf(stderr, "komodo-exporter: %v\n", err)
		return 1
	}
	return 0
}

// setup parses the command line and environment into the HTTP handler and
// the address to serve it on.
func setup(args []string, lookupEnv func(string) (string, bool), logger *slog.Logger, stderr io.Writer) (http.Handler, string, error) {
	fs := flag.NewFlagSet("komodo-exporter", flag.ContinueOnError)
	fs.SetOutput(stderr)
	listen := fs.String("listen", ":9821", "address to serve HTTP on")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if fs.Parse(args) != nil || fs.NArg() > 0 {
		return nil, "", errUsage
	}
	if *showVersion {
		return nil, "", errVersion
	}
	env := map[string]string{}
	for _, name := range []string{"KOMODO_URL", "KOMODO_API_KEY", "KOMODO_API_SECRET"} {
		v, _ := lookupEnv(name)
		if v == "" {
			return nil, "", fmt.Errorf("%s is not set", name)
		}
		env[name] = v
	}
	sdDomain, _ := lookupEnv("KOMODO_EXPORTER_SD_DOMAIN")

	client := komodo.NewClient(env["KOMODO_URL"], env["KOMODO_API_KEY"], env["KOMODO_API_SECRET"], &http.Client{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.New(client, queryTimeout, logger), versioncollector.NewCollector("komodo_exporter"))

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError)}))
	mux.Handle("GET /sd/servers", collector.ServersSD(client, sdDomain, queryTimeout, logger))
	return mux, *listen, nil
}

// serve serves h on addr until ctx is cancelled, then shuts down gracefully.
func serve(ctx context.Context, addr string, h http.Handler, logger *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
