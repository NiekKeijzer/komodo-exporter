package collector

import (
	"strconv"
	"strings"
)

// healthStatus reads the health check status from Docker's status text,
// such as "Up 2 hours (healthy)" or "Up 5 seconds (health: starting)".
func healthStatus(status string) string {
	switch {
	case strings.Contains(status, "(unhealthy)"):
		return "unhealthy"
	case strings.Contains(status, "(health: starting)"):
		return "starting"
	case strings.Contains(status, "(healthy)"):
		return "healthy"
	default:
		return "none"
	}
}

// parsePercent parses a docker stats percentage such as "1.25%".
func parsePercent(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	return v, err == nil
}

// sizeUnits are the units of docker stats sizes: binary for memory,
// decimal for network and block I/O.
var sizeUnits = map[string]float64{
	"B":   1,
	"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40,
	"kB": 1e3, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12,
}

// parseMemUsage parses the usage half of a docker stats memory column such
// as "12.5MiB / 7.6GiB" into bytes.
func parseMemUsage(s string) (float64, bool) {
	usage, _, _ := strings.Cut(s, "/")
	usage = strings.TrimSpace(usage)
	i := strings.IndexFunc(usage, func(r rune) bool { return (r < '0' || r > '9') && r != '.' })
	if i <= 0 {
		return 0, false
	}
	unit, ok := sizeUnits[usage[i:]]
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(usage[:i], 64)
	if err != nil {
		return 0, false
	}
	return v * unit, true
}
