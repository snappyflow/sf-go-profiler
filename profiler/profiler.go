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
	Profile     []byte `json:"data,omitempty"`
	ProfileType string `json:"profile_type,omitempty"`
}

func sleepWithContext(ctx context.Context, delay time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(delay):
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

// func heap(buff *bytes.Buffer) error {
// 	err := pprof.WriteHeapProfile(buff)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

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
			for _, t := range cfg.profileTypes {
				var (
					p   profileData
					err error
				)

				p.Timestamp = unixMillNow()
				p.Type = profile
				p.DocType = profile
				p.Plugin = goProfiler
				p.PID = pid
				p.GoVersion = v
				p.Hostname = hostname
				p.Service = cfg.service
				p.ProfileType = t

				buff.Reset()

				// collect profile
				switch t {
				case cpu:
					err = cpuprofile(ctx, cfg.duration, buff)
				// case Heap:
				// 	err = heap(buffer)
				case heap, block, mutex, goroutine, allocs, threadcreate:
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
