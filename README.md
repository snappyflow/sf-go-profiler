# sf-go-profiler

sf-apm-profiler enables collecting supported profiles types by golang along with runtime metrics
and sends them to snappyflow-apm for further visualization and analysis.

supported profiles: cpu, heap, block, mutex, goroutine, allocs, threadcreate

**cpu** and **heap** profiles are enabled always other types can be enabled as required.

## getting started

- **pre-requisite**

install and configure snappyflow agent on vm or as a sidecar in the container, as it is required to send data to snappyflow-apm

- **simple example**

```go
import "github.com/snappyflow/sf-go-profiler/profiler"

main(){
    profile := profiler.NewProfilerConfig("server")
    profile.Start()
    defer profile.Stop()
    // rest of the application code
}
```

- profiling can conditionally enabled when required using golang flags

```go
import (
    "github.com/snappyflow/sf-go-profiler/profiler"
    "flag"
)

main(){
    enableprofile := flag.Bool("profile",false,"enable profiler")
    if *enableprofile {
        profile := profiler.NewProfilerConfig("server")
        // below line disables collection of go runtime metrics
        // profile.DisableRuntimeMetrics()
        // below line disables profiling
        // profile.DisableProfiles()
        profile.Start()
        defer profile.Stop()
    }
    // rest of the application code
}
```

- runtime metrics can be disable by calling **DisableRuntimeMetrics()** similarly profiling can be disabled by calling **DisableProfiles()** on profile config object.

```go
    profile := profiler.NewProfilerConfig("server")
    // below line disables collection of go runtime metrics
    profile.DisableRuntimeMetrics()
    // below line disables profiling
    profile.DisableProfiles()
    profile.Start()
    defer profile.Stop()
```

## godoc

- <https://pkg.go.dev/github.com/snappyflow/sf-go-profiler/profiler>

## sample code

- <https://github.com/snappyflow/sf-go-profiler/tree/main/example/main.go>

## sample runtime metrics collected

- reference: <https://pkg.go.dev/runtime#MemStats>

```json
{
  "_documentType": "runtime_metrics",
  "_hostname": "ip-172-31-88-98",
  "_plugin": "go_profiler",
  "_tag_Name": "dev",
  "_tag_appName": "profiler-dev",
  "_tag_projectName": "app",
  "_tag_uuid": "02d03d81e525",
  "alloc_mb": 8.4275,
  "frees": 28575,
  "gc_cpu_fraction": 0.0001,
  "go_version": "go1.16.4",
  "interval": 60,
  "language": "golang",
  "last_gc": 1631099627396,
  "live_objects": 27146,
  "mallocs": 27361,
  "max_pause_gc_ms": 0.1033,
  "min_pause_gc_ms": 0.0366,
  "num_cpu": 2,
  "num_gc": 2,
  "num_goroutines": 23,
  "pid": 23201,
  "service": "test",
  "sys_mb": 71.5791,
  "time": 1631099686505,
  "total_alloc_mb": 8.8994,
  "total_pause_gc_ms": 0.14,
  "type": "metrics"
}
```
