// Command perfcli runs the real filter pipeline and reports process allocations.
// Wall times must be measured with the normal binary, not this instrumented one.
package main

import (
	"fmt"
	"os"
	"runtime"

	fzf "github.com/junegunn/fzf/src"
)

func main() {
	opts, err := fzf.ParseOptions(true, os.Args[1:])
	if err != nil {
		panic(err)
	}
	code, err := fzf.Run(opts)
	if err != nil {
		panic(err)
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	fmt.Fprintf(os.Stderr, "PERF %d %d\n", stats.TotalAlloc, stats.Mallocs)
	os.Exit(code)
}
