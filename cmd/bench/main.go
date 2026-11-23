// Command bench compares complete processes on reproducible synthetic inputs.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/junegunn/fzf/internal/fixture"
)

func environment() []string {
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "FZF_") {
			env = append(env, e)
		}
	}
	return env
}

type counter struct{ lines int }

func (c *counter) Write(b []byte) (int, error) {
	for _, x := range b {
		if x == '\n' {
			c.lines++
		}
	}
	return len(b), nil
}

func measure(bin, corpus, query string) (time.Duration, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--filter", query)
	cmd.Env, cmd.Stdin = environment(), strings.NewReader(corpus)
	count := &counter{}
	cmd.Stdout, cmd.Stderr = count, io.Discard
	start := time.Now()
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
		err = nil
	}
	return time.Since(start), count.lines, err
}

func main() {
	adapted, native := os.Getenv("FZFP_BIN"), os.Getenv("FZF_REFERENCE_BIN")
	if adapted == "" || native == "" {
		fmt.Fprintln(os.Stderr, "run make bench")
		os.Exit(2)
	}
	fmt.Printf("100,000 lines; 50%% Han on mixed input; 5 alternating measured runs after one warm-up; median wall time including startup, ingestion, matching, sorting and output.\n\n%s %s/%s, %d logical CPUs\n\n", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	fmt.Println("| Input | Query | fzf ms | fzfp ms | Ratio | Matches fzf / fzfp |")
	fmt.Println("|---|---|---:|---:|---:|---:|")
	for _, mixed := range []bool{false, true} {
		corpus := fixture.Corpus(100000, mixed)
		name := "non-Han"
		if mixed {
			name = "mixed"
		}
		for _, query := range []string{"pr", "zg", "zhongguo", "zhonggu", "zzzz"} {
			for _, bin := range []string{native, adapted} {
				if _, _, err := measure(bin, corpus, query); err != nil {
					panic(err)
				}
			}
			nt, at := []time.Duration{}, []time.Duration{}
			nc, ac := 0, 0
			for run := 0; run < 5; run++ {
				bins := []string{native, adapted}
				if run%2 == 1 {
					slices.Reverse(bins)
				}
				for _, bin := range bins {
					duration, count, err := measure(bin, corpus, query)
					if err != nil {
						panic(err)
					}
					if bin == native {
						nt = append(nt, duration)
						nc = count
					} else {
						at = append(at, duration)
						ac = count
					}
				}
			}
			slices.Sort(nt)
			slices.Sort(at)
			n, a := float64(nt[2])/1e6, float64(at[2])/1e6
			fmt.Printf("| %s | `%s` | %.2f | %.2f | %.2fx | %d / %d |\n", name, query, n, a, a/n, nc, ac)
		}
	}
}
