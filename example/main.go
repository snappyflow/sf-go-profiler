package main

import (
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/snappyflow/sf-go-profiler/profiler"
)

var mem [][]byte

//go:noinline
func fibonacci1(n int) {
	for j := 0; j <= n; j++ {
		if j <= 1 {
			continue
		}

		n2, n1 := big.NewInt(0), big.NewInt(1)

		for i := 1; i < j; i++ {
			n2.Add(n2, n1)
			n1, n2 = n2, n1
		}
		fmt.Println(j, n1)
	}
}

//go:noinline
func fibonacci(value int) int {
	if value <= 1 {
		return value
	}
	return fibonacci(value-2) + fibonacci(value-1)
}

//go:noinline
func allocate3() {
	for i := 0; i < 64*8; i++ {
		mem = append(mem, make([]byte, 64*1024))
	}
	fmt.Println("allocate 3")
}

//go:noinline
func allocate2() {
	for i := 0; i < 64*4; i++ {
		mem = append(mem, make([]byte, 64*1024))
	}
	fmt.Println("allocate 2")
	allocate3()
}

//go:noinline
func allocate1() {
	for i := 0; i < 64*2; i++ {
		mem = append(mem, make([]byte, 64*1024))
	}
	fmt.Println("allocate 1")
	allocate2()
}

func deallocate() {
	mem = [][]byte{}
	fmt.Println("deallocate")
}

func main() {
	profile := profiler.NewProfilerConfig("test")
	// profile.SetInterval(30)
	// profile.SetCPUProfileDuration(5)
	// profile.EnableAllProfiles()
	// // profile.WriteProfileToFile()
	// profile.SetLogger(func(format string, v ...interface{}) {
	// 	fmt.Printf(format+"\n", v...)
	// })
	profile.Start()
	defer profile.Stop()

	killSignal := make(chan os.Signal, 1)

	signal.Notify(killSignal, os.Interrupt, os.Kill, syscall.SIGTERM)

	// c, err := os.Create("cpu.pb.gz")
	// if err != nil {
	// 	log.Fatal("could not create file for cpu profile: ", err)
	// }
	// defer c.Close()
	// if err := pprof.StartCPUProfile(c); err != nil {
	// 	log.Fatal("could not start CPU profile: ", err)
	// }
	// defer pprof.StopCPUProfile()

	done := make(chan struct{})

	go func(done chan struct{}) {
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				allocate1()
				allocate2()
				allocate3()
			case <-done:
				return
			}
		}
	}(done)

	go func(done chan struct{}) {
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				// allocate1()
				// allocate2()
				allocate3()
			case <-done:
				return
			}
		}
	}(done)

	go func(done chan struct{}) {
		timer := time.NewTicker(5 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				fibonacci1(100000000)
			case <-done:
				return
			}
		}
	}(done)

	<-killSignal
	close(done)

	// h, err := os.Create("heap.pb.gz")
	// if err != nil {
	// 	log.Fatal("could not create file for heap profile: ", err)
	// }
	// defer h.Close()
	// runtime.GC() // get up-to-date statistics
	// if err := pprof.WriteHeapProfile(h); err != nil {
	// 	log.Fatal("could not write memory profile: ", err)
	// }
}
