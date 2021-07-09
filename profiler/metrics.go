package profiler

import (
	"context"
	"os"
	"runtime"
	"time"
)

type metricsData struct {
	commonData
	Alloc         uint64 `json:"alloc,omitempty"`
	TotalAlloc    uint64 `json:"total_alloc,omitempty"`
	Sys           uint64 `json:"sys,omitempty"`
	Mallocs       uint64 `json:"mallocs,omitempty"`
	Frees         uint64 `json:"frees,omitempty"`
	LiveObjects   uint64 `json:"live_objects,omitempty"`
	PauseTotalNs  uint64 `json:"pause_total_ns,omitempty"`
	NumGC         uint32 `json:"num_gc,omitempty"`
	NumCPU        int    `json:"num_cpu,omitempty"`
	NumCgoCall    int64  `json:"num_cgo_call,omitempty"`
	NumGoroutines int    `json:"num_goroutines,omitempty"`
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
			d.NumCPU = runtime.NumCPU()
			d.NumCgoCall = runtime.NumCgoCall()

			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			d.Alloc = m.Alloc
			d.TotalAlloc = m.TotalAlloc
			d.Sys = m.Sys
			d.Mallocs = m.Mallocs
			d.Frees = m.Frees
			d.LiveObjects = m.Mallocs - m.Frees
			d.PauseTotalNs = m.PauseTotalNs
			d.NumGC = m.NumGC

			cfg.outMetrics <- d
		}
	}
}
