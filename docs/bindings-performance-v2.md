# Prepared typed bindings: 100k-row benchmark

Historical measurements of `662cfd7b`: its path alignment direction was later
found incorrect on tie-heavy inputs. See [the correction and live regression](bindings-path-fix.md)
and `bindings-benchmark-path-fix.json` for the corrected targeted rerun. The
old corpus's CLI agreement did not establish correctness for all alignments.

Measured 2026-11-25 on Apple M1 Pro arm64, Go 1.27.4, Swift 6.2.4 release, Python 3.12.16, Node 22.22.0.

Same deterministic corpus as the original benchmark, plus low-selectivity `b`. All twelve cases validate full index/score order against legacy exact results, and text order against the CLI. Every top-K is checked against the full prefix. No incremental cache is used.

Corpus file read/import/runtime startup are excluded. Prepared rows exclude corpus construction, measured separately. One cold query is recorded and excluded from warm statistics. Full/convenience/JSON use 7 warm samples; highlighted top-50/4 workers uses 21. CLI includes process startup, ingestion and stdin/stdout, returns text only. Legacy JSON/convenience always compute exact bounds even with positions off; new rank-only full results intentionally do less work, so the speed ratio is not a claim of equivalent exact-alignment work.

This was a shared desktop with ongoing browser/emulator activity; no user processes were stopped. Cases/languages/configurations run sequentially, not interleaved, so differences between languages are not a controlled ranking of runtimes. No outliers are discarded; p95 is the nearest-rank sample percentile, not a production latency guarantee.

Reproduce after building all artifacts, installing the wheel into a venv, and building Swift release:

```sh
python scripts/benchmark-bindings-v2.py --output docs/bindings-benchmark-v2.json
python scripts/summarize-bindings-v2.py
```

## Warm medians: complete matrix

Full results are unbounded; top50 returns at most 50. `positions=true` includes host UTF-8/UTF-16 conversion for returned hits. All times below are milliseconds.

| Scheme/query | Mode | Workers | Positions | Swift | Python | JS | CLI |
|---|---|---:|---|---:|---:|---:|---:|
| default/bj | json | 1 | false | 408.09 | 358.77 | 208.45 | 131.97 |
| default/bj | convenience | 4 | false | 90.40 | 295.53 | 151.61 | 131.97 |
| default/bj | full | 1 | false | 39.51 | 78.45 | 64.63 | 131.97 |
| default/bj | full | 1 | true | 90.20 | 309.55 | 170.91 | 131.97 |
| default/bj | full | 4 | false | 27.08 | 83.56 | 45.48 | 131.97 |
| default/bj | full | 4 | true | 62.19 | 335.32 | 149.43 | 131.97 |
| default/bj | top50 | 1 | false | 26.18 | 56.00 | 30.13 | 131.97 |
| default/bj | top50 | 1 | true | 24.91 | 34.27 | 29.18 | 131.97 |
| default/bj | top50 | 4 | false | 11.87 | 11.76 | 9.30 | 131.97 |
| default/bj | top50 | 4 | true | 8.92 | 7.78 | 12.64 | 131.97 |
| default/beijing | json | 1 | false | 349.35 | 325.36 | 283.88 | 115.22 |
| default/beijing | convenience | 4 | false | 85.21 | 188.99 | 136.94 | 115.22 |
| default/beijing | full | 1 | false | 40.95 | 61.06 | 52.98 | 115.22 |
| default/beijing | full | 1 | true | 98.64 | 181.30 | 150.20 | 115.22 |
| default/beijing | full | 4 | false | 17.30 | 35.80 | 26.28 | 115.22 |
| default/beijing | full | 4 | true | 47.44 | 161.55 | 91.15 | 115.22 |
| default/beijing | top50 | 1 | false | 45.43 | 40.93 | 36.17 | 115.22 |
| default/beijing | top50 | 1 | true | 34.43 | 41.67 | 51.07 | 115.22 |
| default/beijing | top50 | 4 | false | 12.26 | 14.46 | 10.12 | 115.22 |
| default/beijing | top50 | 4 | true | 17.13 | 10.08 | 15.10 | 115.22 |
| default/fb | json | 1 | false | 299.35 | 314.49 | 188.97 | 120.07 |
| default/fb | convenience | 4 | false | 44.55 | 91.87 | 71.01 | 120.07 |
| default/fb | full | 1 | false | 3.09 | 10.86 | 6.81 | 120.07 |
| default/fb | full | 1 | true | 13.14 | 70.60 | 29.04 | 120.07 |
| default/fb | full | 4 | false | 2.23 | 10.23 | 6.30 | 120.07 |
| default/fb | full | 4 | true | 9.63 | 54.95 | 27.36 | 120.07 |
| default/fb | top50 | 1 | false | 1.23 | 1.14 | 1.26 | 120.07 |
| default/fb | top50 | 1 | true | 0.97 | 1.38 | 1.74 | 120.07 |
| default/fb | top50 | 4 | false | 0.45 | 0.51 | 0.98 | 120.07 |
| default/fb | top50 | 4 | true | 0.52 | 0.71 | 0.67 | 120.07 |
| default/b | json | 1 | false | 436.29 | 453.16 | 219.57 | 120.91 |
| default/b | convenience | 4 | false | 85.22 | 268.72 | 141.60 | 120.91 |
| default/b | full | 1 | false | 37.81 | 100.08 | 62.81 | 120.91 |
| default/b | full | 1 | true | 240.84 | 343.68 | 233.13 | 120.91 |
| default/b | full | 4 | false | 30.85 | 131.21 | 71.81 | 120.91 |
| default/b | full | 4 | true | 65.28 | 421.48 | 227.70 | 120.91 |
| default/b | top50 | 1 | false | 22.24 | 23.56 | 25.41 | 120.91 |
| default/b | top50 | 1 | true | 23.01 | 25.79 | 24.15 | 120.91 |
| default/b | top50 | 4 | false | 7.16 | 8.03 | 8.43 | 120.91 |
| default/b | top50 | 4 | true | 7.02 | 7.74 | 7.31 | 120.91 |
| path/bj | json | 1 | false | 433.60 | 421.73 | 218.39 | 134.53 |
| path/bj | convenience | 4 | false | 93.11 | 254.11 | 141.50 | 134.53 |
| path/bj | full | 1 | false | 37.64 | 82.15 | 58.33 | 134.53 |
| path/bj | full | 1 | true | 87.13 | 322.73 | 152.30 | 134.53 |
| path/bj | full | 4 | false | 21.36 | 62.03 | 38.42 | 134.53 |
| path/bj | full | 4 | true | 56.74 | 263.26 | 126.29 | 134.53 |
| path/bj | top50 | 1 | false | 27.82 | 28.08 | 27.49 | 134.53 |
| path/bj | top50 | 1 | true | 26.42 | 28.84 | 28.46 | 134.53 |
| path/bj | top50 | 4 | false | 8.59 | 8.49 | 8.60 | 134.53 |
| path/bj | top50 | 4 | true | 7.99 | 11.98 | 16.71 | 134.53 |
| path/beijing | json | 1 | false | 373.61 | 284.31 | 232.67 | 115.49 |
| path/beijing | convenience | 4 | false | 78.77 | 196.25 | 129.65 | 115.49 |
| path/beijing | full | 1 | false | 40.94 | 74.65 | 54.06 | 115.49 |
| path/beijing | full | 1 | true | 88.23 | 185.39 | 122.97 | 115.49 |
| path/beijing | full | 4 | false | 21.49 | 36.14 | 36.02 | 115.49 |
| path/beijing | full | 4 | true | 53.17 | 195.47 | 179.33 | 115.49 |
| path/beijing | top50 | 1 | false | 38.80 | 56.23 | 36.77 | 115.49 |
| path/beijing | top50 | 1 | true | 39.57 | 37.35 | 44.33 | 115.49 |
| path/beijing | top50 | 4 | false | 10.94 | 11.55 | 14.17 | 115.49 |
| path/beijing | top50 | 4 | true | 12.69 | 11.83 | 11.28 | 115.49 |
| path/fb | json | 1 | false | 245.98 | 267.46 | 175.35 | 82.33 |
| path/fb | convenience | 4 | false | 46.91 | 96.82 | 85.43 | 82.33 |
| path/fb | full | 1 | false | 4.61 | 12.37 | 8.33 | 82.33 |
| path/fb | full | 1 | true | 10.31 | 57.78 | 26.80 | 82.33 |
| path/fb | full | 4 | false | 4.22 | 12.11 | 7.27 | 82.33 |
| path/fb | full | 4 | true | 8.24 | 55.00 | 21.83 | 82.33 |
| path/fb | top50 | 1 | false | 1.69 | 1.59 | 2.18 | 82.33 |
| path/fb | top50 | 1 | true | 1.70 | 1.81 | 1.84 | 82.33 |
| path/fb | top50 | 4 | false | 0.75 | 0.83 | 0.92 | 82.33 |
| path/fb | top50 | 4 | true | 0.73 | 0.90 | 0.78 | 82.33 |
| path/b | json | 1 | false | 427.00 | 408.16 | 213.56 | 103.97 |
| path/b | convenience | 4 | false | 83.53 | 266.72 | 138.02 | 103.97 |
| path/b | full | 1 | false | 39.55 | 186.85 | 73.77 | 103.97 |
| path/b | full | 1 | true | 158.98 | 743.48 | 198.56 | 103.97 |
| path/b | full | 4 | false | 25.98 | 103.35 | 52.76 | 103.97 |
| path/b | full | 4 | true | 68.80 | 542.10 | 155.72 | 103.97 |
| path/b | top50 | 1 | false | 30.71 | 27.58 | 71.94 | 103.97 |
| path/b | top50 | 1 | true | 31.73 | 28.38 | 28.73 | 103.97 |
| path/b | top50 | 4 | false | 8.69 | 9.03 | 10.44 | 103.97 |
| path/b | top50 | 4 | true | 8.13 | 9.42 | 13.88 | 103.97 |
| history/bj | json | 1 | false | 798.24 | 418.51 | 301.53 | 112.18 |
| history/bj | convenience | 4 | false | 125.18 | 367.40 | 184.69 | 112.18 |
| history/bj | full | 1 | false | 49.84 | 192.41 | 69.85 | 112.18 |
| history/bj | full | 1 | true | 84.21 | 440.58 | 161.63 | 112.18 |
| history/bj | full | 4 | false | 22.53 | 72.48 | 43.52 | 112.18 |
| history/bj | full | 4 | true | 65.83 | 534.19 | 168.65 | 112.18 |
| history/bj | top50 | 1 | false | 27.35 | 27.93 | 27.76 | 112.18 |
| history/bj | top50 | 1 | true | 30.82 | 26.84 | 26.76 | 112.18 |
| history/bj | top50 | 4 | false | 17.95 | 8.58 | 11.91 | 112.18 |
| history/bj | top50 | 4 | true | 12.16 | 10.74 | 11.66 | 112.18 |
| history/beijing | json | 1 | false | 391.50 | 367.39 | 605.23 | 127.84 |
| history/beijing | convenience | 4 | false | 88.77 | 265.25 | 113.04 | 127.84 |
| history/beijing | full | 1 | false | 53.06 | 95.20 | 58.77 | 127.84 |
| history/beijing | full | 1 | true | 133.58 | 224.92 | 134.67 | 127.84 |
| history/beijing | full | 4 | false | 47.75 | 44.64 | 26.76 | 127.84 |
| history/beijing | full | 4 | true | 55.21 | 163.89 | 69.43 | 127.84 |
| history/beijing | top50 | 1 | false | 33.17 | 35.28 | 32.16 | 127.84 |
| history/beijing | top50 | 1 | true | 32.66 | 34.26 | 36.12 | 127.84 |
| history/beijing | top50 | 4 | false | 13.60 | 18.44 | 11.69 | 127.84 |
| history/beijing | top50 | 4 | true | 10.72 | 10.41 | 13.55 | 127.84 |
| history/fb | json | 1 | false | 248.54 | 278.22 | 179.57 | 118.50 |
| history/fb | convenience | 4 | false | 44.24 | 92.06 | 74.89 | 118.50 |
| history/fb | full | 1 | false | 1.57 | 11.13 | 6.72 | 118.50 |
| history/fb | full | 1 | true | 7.46 | 56.04 | 24.22 | 118.50 |
| history/fb | full | 4 | false | 1.09 | 10.34 | 7.47 | 118.50 |
| history/fb | full | 4 | true | 5.68 | 57.84 | 23.49 | 118.50 |
| history/fb | top50 | 1 | false | 1.18 | 1.25 | 1.33 | 118.50 |
| history/fb | top50 | 1 | true | 3.35 | 1.50 | 1.22 | 118.50 |
| history/fb | top50 | 4 | false | 0.92 | 0.45 | 0.78 | 118.50 |
| history/fb | top50 | 4 | true | 0.78 | 1.06 | 0.89 | 118.50 |
| history/b | json | 1 | false | 686.83 | 617.62 | 291.44 | 157.32 |
| history/b | convenience | 4 | false | 84.11 | 296.07 | 148.57 | 157.32 |
| history/b | full | 1 | false | 42.92 | 106.26 | 65.47 | 157.32 |
| history/b | full | 1 | true | 78.69 | 368.39 | 153.71 | 157.32 |
| history/b | full | 4 | false | 20.79 | 104.84 | 52.61 | 157.32 |
| history/b | full | 4 | true | 53.04 | 346.44 | 128.28 | 157.32 |
| history/b | top50 | 1 | false | 22.22 | 23.52 | 21.59 | 157.32 |
| history/b | top50 | 1 | true | 23.20 | 23.79 | 22.93 | 157.32 |
| history/b | top50 | 4 | false | 5.76 | 10.26 | 6.41 | 157.32 |
| history/b | top50 | 4 | true | 6.73 | 6.93 | 6.93 | 157.32 |

## Highlighted top-50 latency

| Scheme/query | Swift median/p95 | Python median/p95 | JS median/p95 |
|---|---:|---:|---:|
| default/bj | 8.92 / 12.40 | 7.78 / 11.58 | 12.64 / 21.40 |
| default/beijing | 17.13 / 42.88 | 10.08 / 15.41 | 15.10 / 19.95 |
| default/fb | 0.52 / 0.96 | 0.71 / 1.10 | 0.67 / 1.43 |
| default/b | 7.02 / 13.92 | 7.74 / 12.09 | 7.31 / 8.67 |
| path/bj | 7.99 / 10.82 | 11.98 / 16.18 | 16.71 / 46.71 |
| path/beijing | 12.69 / 16.24 | 11.83 / 13.81 | 11.28 / 12.49 |
| path/fb | 0.73 / 1.50 | 0.90 / 1.27 | 0.78 / 1.07 |
| path/b | 8.13 / 12.70 | 9.42 / 13.38 | 13.88 / 29.20 |
| history/bj | 12.16 / 31.72 | 10.74 / 29.15 | 11.66 / 33.77 |
| history/beijing | 10.72 / 18.72 | 10.41 / 11.63 | 13.55 / 26.21 |
| history/fb | 0.78 / 2.43 | 1.06 / 4.31 | 0.89 / 1.45 |
| history/b | 6.73 / 8.33 | 6.93 / 7.96 | 6.93 / 8.80 |

## Phase breakdown: default/bj

Phase values are medians of wall time. `scan` includes conservative rejection and worker-local heap admission; `select` is merge/final sort. Worker scan/trace values are joined wall times, not summed CPU times. `native` also includes lock wait and query/result setup. Median components need not sum exactly to the median total. Host is per-sample total-native, covering query transport, FFI, row decoding, Unicode conversion and objects. JSON ABI 1 has only end-to-end timings; no invented JSON subphase allocation is reported.

| Mode/workers/positions | Language | Create | Prepare | Scan | Select | Trace | C copy | Native | Host | Total |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| full/1/false | python | 81.747 | 0.002 | 26.176 | 9.771 | 0.000 | 0.395 | 36.602 | 43.012 | 78.450 |
| full/1/false | swift | 41.960 | 0.004 | 27.315 | 10.831 | 0.000 | 0.446 | 38.619 | 0.539 | 39.509 |
| full/1/false | js | 79.958 | 0.004 | 29.284 | 13.608 | 0.000 | 0.407 | 43.891 | 20.738 | 64.629 |
| full/1/true | python | 85.772 | 0.002 | 27.117 | 10.491 | 37.036 | 0.548 | 77.572 | 239.232 | 309.547 |
| full/1/true | swift | 101.841 | 0.002 | 28.799 | 11.254 | 36.035 | 0.501 | 76.379 | 13.445 | 90.195 |
| full/1/true | js | 78.262 | 0.002 | 27.032 | 10.568 | 36.422 | 0.576 | 73.769 | 97.948 | 170.907 |
| full/4/false | python | 75.752 | 0.004 | 12.690 | 11.429 | 0.000 | 0.455 | 24.571 | 60.242 | 83.561 |
| full/4/false | swift | 48.097 | 0.004 | 12.427 | 11.154 | 0.000 | 0.476 | 26.620 | 0.570 | 27.083 |
| full/4/false | js | 96.178 | 0.005 | 11.759 | 12.607 | 0.000 | 0.501 | 25.491 | 23.533 | 45.482 |
| full/4/true | python | 65.247 | 0.002 | 11.673 | 10.539 | 28.762 | 0.563 | 58.791 | 266.036 | 335.322 |
| full/4/true | swift | 64.148 | 0.002 | 8.478 | 13.688 | 25.006 | 0.635 | 50.591 | 13.853 | 62.190 |
| full/4/true | js | 102.772 | 0.001 | 7.924 | 9.510 | 26.189 | 0.603 | 46.259 | 86.793 | 149.434 |
| top50/1/false | python | 93.258 | 0.001 | 55.847 | 0.013 | 0.000 | 0.016 | 55.879 | 0.159 | 55.996 |
| top50/1/false | swift | 106.978 | 0.001 | 26.136 | 0.020 | 0.000 | 0.005 | 26.161 | 0.020 | 26.182 |
| top50/1/false | js | 125.885 | 0.001 | 30.006 | 0.014 | 0.000 | 0.006 | 30.037 | 0.104 | 30.126 |
| top50/1/true | python | 74.918 | 0.001 | 33.897 | 0.014 | 0.022 | 0.010 | 33.989 | 0.277 | 34.266 |
| top50/1/true | swift | 55.176 | 0.001 | 24.816 | 0.011 | 0.024 | 0.011 | 24.869 | 0.043 | 24.907 |
| top50/1/true | js | 84.027 | 0.001 | 28.907 | 0.014 | 0.023 | 0.008 | 28.953 | 0.292 | 29.184 |
| top50/4/false | python | 69.784 | 0.001 | 11.625 | 0.014 | 0.000 | 0.009 | 11.653 | 0.102 | 11.755 |
| top50/4/false | swift | 97.709 | 0.001 | 11.830 | 0.013 | 0.000 | 0.005 | 11.858 | 0.021 | 11.872 |
| top50/4/false | js | 104.727 | 0.001 | 8.949 | 0.024 | 0.000 | 0.005 | 9.008 | 0.096 | 9.300 |
| top50/4/true | python | 70.060 | 0.001 | 7.440 | 0.014 | 0.046 | 0.006 | 7.511 | 0.272 | 7.776 |
| top50/4/true | swift | 54.252 | 0.001 | 8.785 | 0.019 | 0.050 | 0.006 | 8.875 | 0.045 | 8.918 |
| top50/4/true | js | 99.180 | 0.001 | 12.263 | 0.020 | 0.061 | 0.005 | 12.372 | 0.251 | 12.639 |

## Goal audit

Goals came from the approved API survey, section 6; unmet cases are reported, not silently relaxed.

- swift: prepared full <=1.25x CLI: 12/12 cases; exceptions: none.
- swift: top50+highlight median<=50ms and p95<=100ms: 12/12 cases; exceptions: none.
- swift: max case-median top50 boundary + objects + C copy = 0.070ms (goal <=5ms).
- python: prepared full <=1.25x CLI: 12/12 cases; exceptions: none.
- python: top50+highlight median<=50ms and p95<=100ms: 12/12 cases; exceptions: none.
- python: max case-median top50 boundary + objects + C copy = 0.298ms (goal <=5ms).
- js: prepared full <=1.25x CLI: 12/12 cases; exceptions: none.
- js: top50+highlight median<=50ms and p95<=100ms: 12/12 cases; exceptions: none.
- js: max case-median top50 boundary + objects + C copy = 0.256ms (goal <=5ms).

The first nine scheme/query cases reproduce the original corpus queries; `b` is an extra initial-input case. All full-position paths and cold Index construction remain O(corpus/matches) work and must not be compared to rank-only/top-K as equivalent output. Retain an Index for interactive input. Prefix incremental work is deferred, so there is no incremental-vs-cold benchmark in this phase.

The raw JSON preserves all samples, cold/create timings and per-call phase metrics. No isolated profile proves a single cause for every latency outlier; phase metrics identify native-vs-host costs, not system scheduler/thermal causality.
