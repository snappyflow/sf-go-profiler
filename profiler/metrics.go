package profiler

import (
	"context"
	"os"
	"runtime"
	"time"
)

type metricsData struct {
	commonData
	NumGoroutines int              `json:"num_goroutines,omitempty"`
	MemStats      runtime.MemStats `json:"mem_stats,omitempty"`
}

func (cfg *Config) collectRuntimeMetrics(ctx context.Context) {
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	pid := os.Getpid()
	v := runtime.Version()
	hostname, err := os.Hostname()
	if err != nil {
		cfg.logf("failed to get hostname %s", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var d metricsData
			d.Timestamp = time.Now().UnixNano()
			d.Type = metrics
			d.PID = pid
			d.GoVersion = v
			d.Hostname = hostname
			d.Service = cfg.service
			d.NumGoroutines = runtime.NumGoroutine()
			runtime.ReadMemStats(&d.MemStats)
			cfg.outMetrics <- d
		}
	}
}
