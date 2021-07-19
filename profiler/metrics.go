package profiler

import (
	"context"
	"math"
	"os"
	"runtime"
	"strconv"
	"time"
)

type FloatZero float64

func (f FloatZero) MarshalJSON() ([]byte, error) {
	if float64(f) == float64(int(f)) {
		return []byte(strconv.FormatFloat(float64(f), 'f', 1, 64)), nil
	}
	return []byte(strconv.FormatFloat(float64(f), 'f', 4, 64)), nil
}

type metricsData struct {
	commonData

	NumCPU        int `json:"num_cpu,omitempty"`
	NumGoroutines int `json:"num_goroutines,omitempty"`

	AllocMB      FloatZero `json:"alloc_mb,omitempty"`
	TotalAllocMB FloatZero `json:"total_alloc_mb,omitempty"`
	SysMB        FloatZero `json:"sys_mb,omitempty"`
	Mallocs      uint64    `json:"mallocs,omitempty"`
	Frees        uint64    `json:"frees,omitempty"`
	LiveObjects  uint64    `json:"live_objects,omitempty"`

	NumGC          uint32    `json:"num_gc,omitempty"`
	NumForcedGC    uint32    `json:"num_forced_gc,omitempty"`
	LastGC         uint64    `json:"last_gc,omitempty"`
	TotalPauseGcMs FloatZero `json:"total_pause_gc_ms,omitempty"`
	MaxPauseGcMs   FloatZero `json:"max_pause_gc_ms,omitempty"`
	MinPauseGcMs   FloatZero `json:"min_pause_gc_ms,omitempty"`
}

func minmaxPauseNs(pauseNs []uint64, prev, cur uint32) (uint64, uint64) {
	pause := pauseNs[(prev+1+255)%256]
	var min uint64 = pause
	var max uint64 = pause
	for i := prev + 1; i <= cur; i++ {
		if max < pauseNs[i] {
			max = pauseNs[i]
		}
		if (pauseNs[i] > 0) && min > pauseNs[i] {
			min = pauseNs[i]
		}
	}
	return min, max
}

func round(n float64, i int) float64 {
	pow := math.Pow10(i)
	return math.Round(n*pow) / pow
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
			d.Type = metrics
			d.PID = pid
			d.GoVersion = v
			d.Hostname = hostname
			d.Service = cfg.service
			d.DocType = runTimeMetrics
			d.Plugin = goProfiler
			d.Timestamp = unixMillNow()

			d.NumGoroutines = runtime.NumGoroutine()
			d.NumCPU = runtime.NumCPU()

			var cur runtime.MemStats
			runtime.ReadMemStats(&cur)
			// heap
			d.AllocMB = FloatZero(bytesToMB(cur.Alloc))
			d.SysMB = FloatZero(bytesToMB(cur.Sys))
			d.TotalAllocMB = FloatZero(bytesToMB(cur.TotalAlloc - prev.TotalAlloc))
			d.Mallocs = cur.Mallocs - prev.Mallocs
			d.Frees = cur.Frees - prev.Frees
			d.LiveObjects = cur.Mallocs - cur.Frees
			// gc
			d.NumGC = cur.NumGC - prev.NumGC
			d.NumForcedGC = cur.NumForcedGC - prev.NumForcedGC
			d.LastGC = cur.LastGC / 1e6
			d.TotalPauseGcMs = FloatZero(cur.PauseTotalNs-prev.PauseTotalNs) / 1e6
			// find min and max gc pause duration
			min, max := minmaxPauseNs(cur.PauseNs[:], prev.NumGC, cur.NumGC)
			d.MinPauseGcMs = FloatZero(float64(min) / 1e6)
			d.MaxPauseGcMs = FloatZero(float64(max) / 1e6)
			// save current memstat values
			prev = cur

			cfg.outMetrics <- d
		}
	}
}
