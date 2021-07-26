package profiler

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"
)

const (
	cpu            = "cpu"
	heap           = "heap"
	block          = "block"
	mutex          = "mutex"
	goroutine      = "goroutine"
	allocs         = "allocs"
	threadcreate   = "threadcreate"
	metrics        = "metrics"
	profile        = "profile"
	goProfiler     = "go_profiler"
	runTimeMetrics = "runtime_metrics"
)

const (
	// DefaultProfilesDir is the default directory where profiles are stored while writing to file.
	DefaultProfilesDir = "./profiles"

	// DefaultProfilesAge is the time to preserve old profile files.
	DefaultProfilesAge = 900 * time.Second

	// DefaultAgentURL default url to send profiles to agent.
	DefaultAgentURL = "http://127.0.0.1:8588"

	// DefaultClusterForwarderURL default url to send profiles to agent.
	DefaultClusterForwarderURL = "http://forwarder.sfagent.svc"

	// DefaultCPUProfileDuration is the default cpu profile duration in seconds.
	DefaultCPUProfileDuration = 10 * time.Second

	// DefaultProfileInterval is the  default intervals at which profiles are collected.
	DefaultProfileInterval = 60 * time.Second

	// DefaultMutexProfileFraction
	// check https://pkg.go.dev/runtime#SetMutexProfileFraction
	DefaultMutexProfileFraction = 100

	// DefaultBlockProfileRate
	// check https://pkg.go.dev/runtime#SetBlockProfileRate
	DefaultBlockProfileRate = 10000
)

var (
	allProfiles     = []string{threadcreate, block, mutex, goroutine, heap, cpu}
	defaultProfiles = map[string]bool{
		threadcreate: false,
		block:        false,
		mutex:        false,
		goroutine:    false,
		heap:         true,
		cpu:          true,
	}
	logger      = log.New(os.Stdout, "[go profiler] ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmicroseconds)
	defaultlogf = func(format string, v ...interface{}) { logger.Printf(format+"\n", v...) }
)

type commonData struct {
	Timestamp int64  `json:"time,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Type      string `json:"type,omitempty"`
	Service   string `json:"service,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
	Hostname  string `json:"_hostname,omitempty"`
	DocType   string `json:"_documentType,omitempty"`
	Plugin    string `json:"_plugin,omitempty"`
}

// Config for profiling.
type Config struct {
	service         string
	targetURL       string
	customTarget    bool
	dumpToFile      bool
	collectMetrics  bool
	collectProfiles bool
	enabled         map[string]bool
	ackProfile      chan struct{}
	outProfile      chan profileData
	outMetrics      chan metricsData
	duration        time.Duration
	interval        time.Duration
	cancel          context.CancelFunc
	logf            func(format string, v ...interface{})
}

// NewProfilerConfig returns profiler config.
//
// Accepts service name as argument, service name is required for identification.
func NewProfilerConfig(service string) *Config {
	return &Config{
		collectProfiles: true,
		collectMetrics:  true,
		ackProfile:      make(chan struct{}, 1),
		service:         service,
		duration:        DefaultCPUProfileDuration,
		interval:        DefaultProfileInterval,
		enabled:         defaultProfiles,
		outProfile:      make(chan profileData, 1),
		outMetrics:      make(chan metricsData, 1),
		dumpToFile:      false,
		targetURL:       DefaultAgentURL,
		customTarget:    false,
		logf:            defaultlogf,
	}
}

// DisableProfiles disables all profile collection.
func (cfg *Config) DisableProfiles() {
	cfg.collectProfiles = false
}

// DisableRuntimeMetrics disables runtime metric collection.
func (cfg *Config) DisableRuntimeMetrics() {
	cfg.collectMetrics = false
}

// SetInterval sets interval in seconds between profiles collection.
func (cfg *Config) SetInterval(i int) {
	cfg.interval = time.Duration(i) * time.Second
}

// SetCPUProfileDuration sets duration in seconds for which cpu profile is collected.
func (cfg *Config) SetCPUProfileDuration(i int) {
	cfg.duration = time.Duration(i) * time.Second
}

// EnableBlockProfile enables block profile.
//
// https://pkg.go.dev/runtime#SetBlockProfileRate
func (cfg *Config) EnableBlockProfile(rate int) {
	runtime.SetBlockProfileRate(rate)
	cfg.enabled[block] = true
}

// EnableMutexProfile enables mutex profile.
//
// https://pkg.go.dev/runtime#SetMutexProfileFraction
func (cfg *Config) EnableMutexProfile(rate int) {
	runtime.SetMutexProfileFraction(rate)
	cfg.enabled[mutex] = true
}

// EnableGoRoutineProfile enables goroutine profile.
func (cfg *Config) EnableGoRoutineProfile() {
	cfg.enabled[goroutine] = true
}

// EnableThreadCreateProfile enables threadcreate profile.
func (cfg *Config) EnableThreadCreateProfile() {
	cfg.enabled[threadcreate] = true
}

// EnableAllProfiles enables all currently supported profile types.
func (cfg *Config) EnableAllProfiles() {
	runtime.SetBlockProfileRate(DefaultBlockProfileRate)
	runtime.SetMutexProfileFraction(DefaultMutexProfileFraction)
	for k, _ := range cfg.enabled {
		cfg.enabled[k] = true
	}
}

// WriteProfileToFile writes all collected profiles to file to DefaultProfilesDir directory,
// with file name formatted as service_timestamp_pid.profiletype .
func (cfg *Config) WriteProfileToFile() {
	cfg.dumpToFile = true
}

// SetTargetURL sets target url to given string, useful for changing where profiles are sent.
func (cfg *Config) SetTargetURL(url string) {
	cfg.customTarget = true
	cfg.targetURL = url
}

// SetLogger allows to set custom logger,
// logger function format func(format string, v ...interface{}) .
func (cfg *Config) SetLogger(logf func(format string, v ...interface{})) {
	cfg.logf = logf
}
