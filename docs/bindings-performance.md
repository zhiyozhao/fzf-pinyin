# Binding benchmark, 2026-11-25

The first binding implementation is correct but slower than the CLI on this
100,000-row batch. It prioritizes a small, stable ABI and exact alignment.
Do not infer that an in-process call is automatically faster than `--filter`.

Apple M1 Pro, arm64, Go 1.27.4, Swift 6.2.4 release, Python 3.12.16.
Five full calls per case, no discarded warmup; table shows median milliseconds.
Corpus creation and file loading are outside timing. Binding import and runtime
initialization are outside timing; JSON conversion, Go matching and sorting,
and complete host result materialization are included. CLI includes process
startup and stdin/stdout. Positions arrays are off; bindings still trace exact
bounds. CLI returns text only, bindings also materialize scores and bounds.
CLI and bindings have different execution pipelines, so these are end-to-end
consumer costs, not isolated FFI overhead measurements.

| Scheme | Query | Matches | Swift ms | Python ms | CLI ms |
|---|---|---:|---:|---:|---:|
| default | bj | 40,000 | 371.0 | 317.6 | 80.0 |
| default | beijing | 20,000 | 282.2 | 272.3 | 67.6 |
| default | fb | 10,000 | 220.5 | 190.1 | 58.5 |
| path | bj | 40,000 | 368.7 | 351.0 | 88.3 |
| path | beijing | 20,000 | 295.4 | 330.5 | 85.2 |
| path | fb | 10,000 | 224.7 | 183.2 | 62.4 |
| history | bj | 40,000 | 360.0 | 324.8 | 86.4 |
| history | beijing | 20,000 | 273.7 | 260.3 | 73.7 |
| history | fb | 10,000 | 217.2 | 183.4 | 57.3 |

Every full sorted result was checked: Python text output equals CLI output for
each repetition, and Swift input-index output equals Python's complete sequence.
The deterministic corpus contains 100,000 distinct path strings, mixing 北京,
中国, 绿茶, 马超, fooBar, b-j, bj, beijing, project, and 长沙. No top-K truncation
is used. All raw samples and environment metadata are in
[`bindings-benchmark.json`](bindings-benchmark.json).

Reproduce after building and installing a wheel into a virtual environment:

```sh
bash scripts/build-bindings.sh all
swift build -c release --package-path bindings/swift
python scripts/benchmark-bindings.py --output .cache/bindings/benchmark.json
```

Likely future improvements, requiring profiling: a typed C batch transport to
avoid JSON, a prepared-corpus handle to avoid copying unchanged candidates,
top-K results to reduce host allocations, and parallel candidate matching under
one selected scheme. These are deferred API changes. This benchmark does not
isolate which component dominates and does not promise any specific speedup.

Local artifacts: a universal macOS static archive of about 19 MiB before zip;
compressed Python wheels about 1.45 MiB arm64 / 1.55 MiB x86_64, including Go
runtime and full license notices. arm64 was executed; x86_64 was compiled and
inspected, not run. Linux CI is authored but was not executed in this session.

## Current prepared/typed three-language results

The original JSON benchmark above is retained as historical evidence. The new
Swift/Python/Node matrix, phase breakdown, full/top-50 and 1/4-worker timings
are documented in [bindings-performance-v2.md](bindings-performance-v2.md).
