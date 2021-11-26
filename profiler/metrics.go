package profiler

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"
)

type floatZero float64

func (f floatZero) MarshalJSON() ([]byte, error) {
	if float64(f) == float64(int(f)) {
		return []byte(strconv.FormatFloat(float64(f), 'f', 1, 64)), nil
	}

	return []byte(strconv.FormatFloat(float64(f), 'f', 4, 64)), nil
}

type metricsData struct {
	commonData

	NumCPU        int `json:"num_cpu,omitempty"`
	NumGoroutines int `json:"num_goroutines,omitempty"`

	AllocMB      floatZero `json:"alloc_mb,omitempty"`
	TotalAllocMB floatZero `json:"total_alloc_mb,omitempty"`
	SysMB        floatZero `json:"sys_mb,omitempty"`
	Mallocs      uint64    `json:"mallocs,omitempty"`
	Frees        uint64    `json:"frees,omitempty"`
	LiveObjects  uint64    `json:"live_objects,omitempty"`

	NumGC          uint32    `json:"num_gc,omitempty"`
	NumForcedGC    uint32    `json:"num_forced_gc,omitempty"`
	LastGC         uint64    `json:"last_gc,omitempty"`
	TotalPauseGcMs floatZero `json:"total_pause_gc_ms,omitempty"`
	MaxPauseGcMs   floatZero `json:"max_pause_gc_ms,omitempty"`
	MinPauseGcMs   floatZero `json:"min_pause_gc_ms,omitempty"`
	GcCPUFraction  floatZero `json:"gc_cpu_fraction,omitempty"`
}

func minmaxPauseNs(pauseNs []uint64, prev, cur uint32) (uint64, uint64) {
	start := (prev + 1 + 255) % 256
	pause := pauseNs[start]
	min, max := pause, pause

	for i := start; i <= cur; i++ {
		if max < pauseNs[i] {
			max = pauseNs[i]
		}
		if (pauseNs[i] > 0) && min > pauseNs[i] {
			min = pauseNs[i]
		}
	}

	return min, max
}

func bytesToMB(n uint64) float64 {
	return float64(n) / 1024.0 / 1024.0
}

func (cfg *Config) collectRuntimeMetrics(ctx context.Context) {
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	var prev runtime.MemStats

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
			d.Language = golang
			d.Type = metrics
			d.PID = pid
			d.GoVersion = v
			d.Hostname = hostname
			d.Service = cfg.service
			d.DocType = runTimeMetrics
			d.Plugin = goProfiler
			d.Timestamp = unixMillNow()
			d.Interval = int(cfg.interval / time.Second)

			d.NumGoroutines = runtime.NumGoroutine()
			d.NumCPU = runtime.NumCPU()

			var cur runtime.MemStats

			runtime.ReadMemStats(&cur)
			// heap
			d.AllocMB = floatZero(bytesToMB(cur.Alloc))
			d.SysMB = floatZero(bytesToMB(cur.Sys))
			d.TotalAllocMB = floatZero(bytesToMB(cur.TotalAlloc - prev.TotalAlloc))
			d.Mallocs = cur.Mallocs - prev.Mallocs
			d.Frees = cur.Frees - prev.Frees
			d.LiveObjects = cur.Mallocs - cur.Frees
			// gc
			d.NumGC = cur.NumGC - prev.NumGC
			d.NumForcedGC = cur.NumForcedGC - prev.NumForcedGC
			d.GcCPUFraction = floatZero(cur.GCCPUFraction)
			d.LastGC = cur.LastGC / 1e6
			d.TotalPauseGcMs = floatZero(cur.PauseTotalNs-prev.PauseTotalNs) / 1e6
			// find min and max gc pause duration
			min, max := minmaxPauseNs(cur.PauseNs[:], prev.NumGC, cur.NumGC)
			d.MinPauseGcMs = floatZero(float64(min) / 1e6)
			d.MaxPauseGcMs = floatZero(float64(max) / 1e6)
			// save current memstat values
			prev = cur

			cfg.outMetrics <- d
		}
	}
}
