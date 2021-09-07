# sf-go-profiler

sf-apm-profiler enables collecting supported profiles types by golang along with runtime metrics
and sends them to SnappyFlowAPM for further visualization and analysis.

supported profiles: cpu, heap, block, mutex, goroutine, allocs, threadcreate

cpu and heap profiles are enabled always other types can be enabled as required.

## getting started

- simple example

```go
import "github.com/snappyflow/sf-go-profiler/profiler"

main(){
    profile := profiler.NewProfilerConfig("server")
    profile.Start()
    defer profile.Stop()
    // rest of the application code
}
```

- profiling can conditionally enabled when required using flags

```go
import (
    "github.com/snappyflow/sf-go-profiler/profiler"
    "flag"
)

main(){
    enableprofile := flag.Bool("debug",false,"enable profiler")
    if *enableprofile {
        profile := profiler.NewProfilerConfig("server")
        profile.Start()
        defer profile.Stop()
    }
    // rest of the application code
}
```

- runtime metrics can be disable by calling **DisableRuntimeMetrics()** similarly profiling can be disabled by calling **DisableProfiles()** on profile config object.

## documentation

- <https://pkg.go.dev/github.com/snappyflow/sf-go-profiler/profiler>
- sample code  <https://github.com/snappyflow/sf-go-profiler/tree/main/example>
