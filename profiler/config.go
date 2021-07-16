package profiler

import (
	"context"
	"log"
	"os"
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
	// default directory where profiles are stored while writing to file
	DefaultProfilesDir = "./profiles"

	// time to preserve old profile files
	DefaultProfilesAge = 900 * time.Second

	// default url to send profiles to agent
	DefaultAgentURL = "http://127.0.0.1:8588"

	// default url to send profiles to agent
	DefaultClusterForwarderURL = "http://forwarder.sfagent.svc"

	// default cpu profile duration
	DefaultCPUProfileDuration = 10 * time.Second

	// default intervals at which profiles are collected
	DefaultProfileInterval = 60 * time.Second
)

var (
	allProfiles     = []string{threadcreate, block, mutex, goroutine, allocs, heap, cpu}
	defaultProfiles = []string{heap, cpu}
	logger          = log.New(os.Stdout, "[go profiler] ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmicroseconds)
	defaultlogf     = func(format string, v ...interface{}) { logger.Printf(format+"\n", v...) }
)

type commonData struct {
	PID       int    `json:"pid,omitempty"`
	Timestamp int64  `json:"time,omitempty"`
	Type      string `json:"type,omitempty"`
	Service   string `json:"service,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
	Hostname  string `json:"_hostname,omitempty"`
	DocType   string `json:"_documentType,omitempty"`
	Plugin    string `json:"_plugin,omitempty"`
}

type Config struct {
	collectProfiles bool
	collectMetrics  bool
	duration        time.Duration
	interval        time.Duration
	profileTypes    []string
	cancel          context.CancelFunc
	outProfile      chan profileData
	outMetrics      chan metricsData
	service         string
	dumpToFile      bool
	targetURL       string
	customTarget    bool
	logf            func(format string, v ...interface{})
}

// NewProfilerConfig returns profiler config
//
// Accepts service name as argument, service name is required for identification
func NewProfilerConfig(service string) *Config {
	return &Config{
		collectProfiles: true,
		collectMetrics:  true,
		service:         service,
		duration:        DefaultCPUProfileDuration,
		interval:        DefaultProfileInterval,
		profileTypes:    defaultProfiles,
		outProfile:      make(chan profileData, len(allProfiles)+1),
		outMetrics:      make(chan metricsData, 1),
		dumpToFile:      false,
		targetURL:       DefaultAgentURL,
		customTarget:    false,
		logf:            defaultlogf,
	}
}

// DisableProfiles disables all profile collection
func (cfg *Config) DisableProfiles() {
	cfg.collectProfiles = false
}

// DisableRuntimeMetrics disables runtime metric collection
func (cfg *Config) DisableRuntimeMetrics() {
	cfg.collectMetrics = false
}

// SetInterval sets interval between profiles collection
func (cfg *Config) SetInterval(i int) {
	cfg.interval = time.Duration(i) * time.Second
}

// SetCPUProfileDuration sets duration in seconds for which cpu profile is collected
func (cfg *Config) SetCPUProfileDuration(i int) {
	cfg.duration = time.Duration(i) * time.Second
}

// EnableBlockProfile enables block profile
func (cfg *Config) EnableBlockProfile() {
	cfg.profileTypes = append(cfg.profileTypes, block)
}

// EnableMutexProfile enables mutex profile
func (cfg *Config) EnableMutexProfile() {
	cfg.profileTypes = append(cfg.profileTypes, mutex)
}

// EnableGoRoutineProfile enables goroutine profile
func (cfg *Config) EnableGoRoutineProfile() {
	cfg.profileTypes = append(cfg.profileTypes, goroutine)
}

// EnableThreadCreateProfile enables threadcreate profile
func (cfg *Config) EnableThreadCreateProfile() {
	cfg.profileTypes = append(cfg.profileTypes, threadcreate)
}

// EnableAllProfiles enables all currently supported profile types
func (cfg *Config) EnableAllProfiles() {
	cfg.profileTypes = allProfiles
}

// WriteProfileToFile writes all collected profiles to file to DefaultProfilesDir directory,
// with file name formatted as service_timestamp_pid.profiletype
func (cfg *Config) WriteProfileToFile() {
	cfg.dumpToFile = true
}

// SetTargetURL sets target url to given string, useful for changing where profiles are sent
func (cfg *Config) SetTargetURL(url string) {
	cfg.customTarget = true
	cfg.targetURL = url
}

// SetLogger allows to set custom logger
// logger function format func(format string, v ...interface{})
func (cfg *Config) SetLogger(logf func(format string, v ...interface{})) {
	cfg.logf = logf
}
