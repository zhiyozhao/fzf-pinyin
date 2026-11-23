// Command perf measures normal --filter processes, and optionally verifies bytes
// and probes allocations with a separate perfcli build from each revision.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/junegunn/fzf/internal/fixture"
)

type engine struct{ label, bin, alloc string }
type record struct {
	Case, Engine    string
	MedianMS, P95MS float64
	Bytes, Allocs   uint64
	OutputSHA256    string
	Exit            int
}

func main() {
	bins := flag.String("bins", "", "label=normal-binary,...")
	allocBins := flag.String("alloc-bins", "", "label=perfcli-binary,...")
	count := flag.Int("runs", 21, "measured alternating runs after warmup")
	pattern := flag.String("cases", "", "case name substring")
	compare := flag.Bool("compare", false, "require identical output hashes and exit status")
	scheme := flag.String("scheme", "default", "scoring scheme")
	flag.Parse()
	alloc := map[string]string{}
	if *allocBins != "" {
		for _, pair := range strings.Split(*allocBins, ",") {
			p := strings.SplitN(pair, "=", 2)
			alloc[p[0]] = p[1]
		}
	}
	engines := []engine{}
	for _, pair := range strings.Split(*bins, ",") {
		p := strings.SplitN(pair, "=", 2)
		if len(p) != 2 {
			panic("expected -bins label=path,...")
		}
		engines = append(engines, engine{p[0], p[1], alloc[p[0]]})
	}
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "FZF_") && !strings.HasPrefix(e, "FZFP_") {
			env = append(env, e)
		}
	}
	for _, c := range fixture.PerfCases() {
		if c.Positions || !strings.Contains(c.Name, *pattern) {
			continue
		}
		run := func(bin string, probe bool) (float64, record) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "--scheme="+*scheme, "--filter", c.Query)
			cmd.Env, cmd.Stdin = env, strings.NewReader(c.Corpus)
			hash := sha256.New()
			var stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = hash, &stderr
			start := time.Now()
			err := cmd.Run()
			ms := float64(time.Since(start)) / 1e6
			code := 0
			if err != nil {
				if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
					code = 1
				} else {
					panic(fmt.Sprintf("%s: %v %s", c.Name, err, stderr.String()))
				}
			}
			r := record{Case: c.Name, OutputSHA256: fmt.Sprintf("%x", hash.Sum(nil)), Exit: code}
			if probe {
				if _, err := fmt.Sscanf(stderr.String(), "PERF %d %d", &r.Bytes, &r.Allocs); err != nil {
					panic(stderr.String())
				}
			} else if stderr.Len() != 0 {
				panic(stderr.String())
			}
			return ms, r
		}
		durations := make([][]float64, len(engines))
		records := make([]record, len(engines))
		for i, e := range engines {
			_, records[i] = run(e.bin, false)
			if *compare && i > 0 && (records[0].OutputSHA256 != records[i].OutputSHA256 || records[0].Exit != records[i].Exit) {
				panic("output differs: " + c.Name)
			}
		}
		for i := 0; i < *count; i++ {
			for k := range engines {
				j := k
				if i%2 != 0 {
					j = len(engines) - 1 - k
				}
				ms, _ := run(engines[j].bin, false)
				durations[j] = append(durations[j], ms)
			}
		}
		for i, e := range engines {
			slices.Sort(durations[i])
			r := records[i]
			r.Engine = e.label
			r.MedianMS = durations[i][len(durations[i])/2]
			r.P95MS = durations[i][(len(durations[i])*95+99)/100-1]
			if e.alloc != "" {
				for j := 0; j < 3; j++ {
					_, a := run(e.alloc, true)
					r.Bytes += a.Bytes / 3
					r.Allocs += a.Allocs / 3
				}
			}
			data, _ := json.Marshal(r)
			fmt.Println(string(data))
		}
	}
}
