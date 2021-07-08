# sf-go-profiler

sf-apm-profiler enables collecting supported profiles types by golang
and sends them to SnappyFlowAPM for further visualization and analysis.

supported profiles: cpu, heap, block, mutex, goroutine, allocs, threadcreate

cpu and heap profiles are enabled always other types can be enabled as required.

## enabling profiling

```
 profile := profiler.NewProfilerConfig("server")
 profile.Start()
 defer profile.Stop()
```

## documentation

- <https://pkg.go.dev/github.com/snappyflow/sf-go-profiler>
- sample code  <https://github.com/snappyflow/sf-go-profiler/example>
