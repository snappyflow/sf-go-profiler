// profiler enables collecting supported profiles types by golang
// and sends them to SnappyFlowAPM for further visualization and analysis.
//
// supported profiles: cpu, heap, block, mutex, goroutine, allocs, threadcreate
//
// cpu and heap profiles are enabled always other types can be enabled as required.
package profiler

import (
	"bytes"
	"context"
	"errors"
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

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
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
		return errors.New("not found")
	}
	err := prof.WriteTo(buff, 0)
	if err != nil {
		return err
	}
	return nil
}

func (cfg *Config) collectProfiles(ctx context.Context) {
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

				var p profileData
				var err error

				p.Timestamp = unixMillNow()
				p.Type = profile
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
			}
		}
	}
}

// Start profile collection routines
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
	go cfg.collectProfiles(ctx)
	// start runtime metrics collector
	go cfg.collectRuntimeMetrics(ctx)
}

// Stop profile collection
func (cfg *Config) Stop() {
	cfg.cancel()
}
