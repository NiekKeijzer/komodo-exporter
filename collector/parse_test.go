package collector

import "testing"

func TestHealthStatus(t *testing.T) {
	for status, want := range map[string]string{
		"Up 2 hours (healthy)":            "healthy",
		"Up 3 minutes (unhealthy)":        "unhealthy",
		"Up 5 seconds (health: starting)": "starting",
		"Up 2 hours":                      "none",
		"Exited (0) 2 hours ago":          "none",
		"":                                "none",
	} {
		if got := healthStatus(status); got != want {
			t.Errorf("healthStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestParsePercent(t *testing.T) {
	for in, want := range map[string]float64{"1.50%": 1.5, "0.00%": 0, "250.3%": 250.3} {
		if got, ok := parsePercent(in); !ok || got != want {
			t.Errorf("parsePercent(%q) = %v, %v, want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"--", ""} {
		if _, ok := parsePercent(in); ok {
			t.Errorf("parsePercent(%q) succeeded, want failure", in)
		}
	}
}

func TestParseMemUsage(t *testing.T) {
	for in, want := range map[string]float64{
		"62.5MiB / 7.637GiB": 62.5 * (1 << 20),
		"1.2GiB / 7.637GiB":  1.2 * (1 << 30),
		"512KiB / 1GiB":      512 * (1 << 10),
		"900B / 1GiB":        900,
		"12.5MB / 1GB":       12.5e6,
	} {
		if got, ok := parseMemUsage(in); !ok || got != want {
			t.Errorf("parseMemUsage(%q) = %v, %v, want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"-- / --", "", "12XB / 1GiB", "MiB / 1GiB"} {
		if _, ok := parseMemUsage(in); ok {
			t.Errorf("parseMemUsage(%q) succeeded, want failure", in)
		}
	}
}
