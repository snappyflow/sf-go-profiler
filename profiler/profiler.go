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

type pprofData struct {
	Timestamp int64  `json:"timestamp,omitempty"`
	Type      string `json:"type,omitempty"`
	Data      []byte `json:"data,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Service   string `json:"service,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
}

func sleepWithContext(ctx context.Context, delay time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
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

func (cfg *Config) run(ctx context.Context) {
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
					p   pprofData
					err error
				)

				p.Timestamp = time.Now().UnixNano()
				p.Type = t
				p.PID = pid
				p.GoVersion = v
				p.Hostname = hostname
				p.Service = cfg.service

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
				p.Data = buff.Bytes()
				// send data
				cfg.out <- p
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
	go cfg.run(ctx)
}

// Stop profile collection
func (cfg *Config) Stop() {
	cfg.cancel()
}
