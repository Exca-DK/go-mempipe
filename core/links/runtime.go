package links

import (
	"runtime"
	"unsafe"
	_ "unsafe"
)

//go:linkname mcall runtime.mcall
func mcall(fn func(unsafe.Pointer))

//go:linkname gosched_m runtime.gosched_m
func gosched_m(gp unsafe.Pointer)

//go:linkname spin runtime.procyield
func spin(cycles uint32)

// whether the system has multiple cores or a single core
var multicore = runtime.NumCPU() > 1

func Wait() {
	if multicore {
		spin(100)
	} else {
		mcall(gosched_m)
	}
}
