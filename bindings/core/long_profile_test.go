package core

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkMaclipLong(b *testing.B) {
	text := "收货地址\n" + strings.Repeat("synthetic line 你好 0123456789\n", 39999) + "END"
	x, err := NewIndex([]string{text}, Options{Scheme: "history"}, 1)
	if err != nil {
		b.Fatal(err)
	}
	for _, q := range []string{"s", "shdz", "shd"} {
		for _, positions := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/positions=%t", q, positions), func(b *testing.B) {
				var scan, trace float64
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					hits, t, err := x.FindMode(q, -1, positions, false, false)
					if err != nil || len(hits) != 1 {
						b.Fatalf("hits=%v err=%v", hits, err)
					}
					scan += t.Scan
					trace += t.Trace
				}
				b.ReportMetric(scan/float64(b.N), "scan-ms")
				b.ReportMetric(trace/float64(b.N), "trace-ms")
			})
		}
	}
}
