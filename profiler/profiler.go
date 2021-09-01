// Package profiler enables collecting supported profiles types by golang
// and sends them to SnappyFlowAPM for further visualization and analysis.
//
// supported profiles: cpu, heap, block, mutex, goroutine, allocs, threadcreate
//
// cpu and heap profiles are enabled always other types can be enabled as required.
package profiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

type profileData struct {
	commonData
	Duration    int    `json:"duration,omitempty"`
	Profile     []byte `json:"pprof,omitempty"`
	ProfileType string `json:"profile_type,omitempty"`
}

func sleepWithContext(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		timer.Stop()
	case <-timer.C:
		timer.Stop()
	}
}

func unixMillNow() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func cpuprofile(ctx context.Context, duration time.Duration, buff *bytes.Buffer) error {
	err := pprof.StartCPUProfile(buff)
	if err != nil {
		return err
	}

	sleepWithContext(ctx, duration)

	pprof.StopCPUProfile()

	return nil
}

func getProfile(name string, buff *bytes.Buffer) error {
	prof := pprof.Lookup(name)
	if prof == nil {
		return fmt.Errorf("%s profile not found", name)
	}

	return prof.WriteTo(buff, 0)
}

func (cfg *Config) gatherProfiles(ctx context.Context) {
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	pid := os.Getpid()
	v := runtime.Version()

	hostname, err := os.Hostname()
	if err != nil {
		cfg.logf("failed to get hostname %s", err)
	}

	buff := new(bytes.Buffer)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			for _, t := range allProfiles {

				// skip profiles not enabled
				if enabled := cfg.enabled[t]; !enabled {
					continue
				}

				var (
					p   profileData
					err error
				)

				p.Language = golang
				p.Timestamp = unixMillNow()
				p.Type = profile
				p.DocType = profile
				p.Plugin = goProfiler
				p.PID = pid
				p.GoVersion = v
				p.Hostname = hostname
				p.Service = cfg.service
				p.ProfileType = t
				p.Interval = int(cfg.interval / time.Second)

				buff.Reset()

				// collect profile
				switch t {
				case cpu:
					p.Duration = int(cfg.duration / time.Second)
					err = cpuprofile(ctx, cfg.duration, buff)
				case heap, block, mutex, goroutine, threadcreate:
					err = getProfile(t, buff)
				}

				if err != nil {
					cfg.logf("failed to collect %s profile, %s", t, err)
					continue
				}
				// add collected data
				p.Profile = buff.Bytes()
				// send data
				cfg.outProfile <- p
				// wait till profile is processed
				<-cfg.ackProfile
			}
		}
	}
}

// Start profile collection routines.
func (cfg *Config) Start() {
	var ctx context.Context
	ctx, cfg.cancel = context.WithCancel(context.Background())
	// start publish
	if cfg.dumpToFile {
		go cfg.writeToFile(ctx)
	} else {
		go cfg.sendToAgent(ctx)
	}
	// start profiler
	if cfg.collectProfiles {
		go cfg.gatherProfiles(ctx)
	}
	// start runtime metrics collector
	if cfg.collectMetrics {
		go cfg.collectRuntimeMetrics(ctx)
	}
}

// Stop profile collection.
func (cfg *Config) Stop() {
	cfg.cancel()
}
