"""Render tables from measured samples; never substitutes targets for results."""
import argparse
import json
from pathlib import Path
import statistics

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--input", type=Path, default=root / ".cache/bindings/benchmark-v2.json")
parser.add_argument("--output", type=Path, default=root / ".cache/bindings/performance-v2.md")
args = parser.parse_args()
data = json.loads(args.input.read_text())
languages = ("swift", "python", "js")
lines = ["# Prepared typed bindings: 100k-row benchmark", "",
         "The committed performance summary preserves historical measurements of `662cfd7b`, whose path direction was later corrected. See docs/bindings-path-fix.md for corrected measurements and regression coverage. Raw samples and regenerated tables stay in ignored .cache/bindings.", "",
         "Measured 2026-11-25 on Apple M1 Pro arm64, Go 1.27.4, Swift 6.2.4 release, Python 3.12.16, Node 22.22.0.", "",
         "Same deterministic corpus as the original benchmark, plus low-selectivity `b`. All twelve cases validate full index/score order against legacy exact results, and text order against the CLI. Every top-K is checked against the full prefix. No incremental cache is used.", "",
         "Corpus file read/import/runtime startup are excluded. Prepared rows exclude corpus construction, measured separately. One cold query is recorded and excluded from warm statistics. Full/convenience/JSON use 7 warm samples; highlighted top-50/4 workers uses 21. CLI includes process startup, ingestion and stdin/stdout, returns text only. Legacy JSON/convenience always compute exact bounds even with positions off; new rank-only full results intentionally do less work, so the speed ratio is not a claim of equivalent exact-alignment work.", "",
         "This was a shared desktop with ongoing browser/emulator activity; no user processes were stopped. Cases/languages/configurations run sequentially, not interleaved, so differences between languages are not a controlled ranking of runtimes. No outliers are discarded; p95 is the nearest-rank sample percentile, not a production latency guarantee.", "",
         "Reproduce after building all artifacts, installing the wheel into a venv, and building Swift release:", "",
         "```sh", "python scripts/benchmark-bindings-v2.py --output .cache/bindings/benchmark-v2.json", "python scripts/summarize-bindings-v2.py", "```", "",
         "## Warm medians: complete matrix", "",
         "Full results are unbounded; top50 returns at most 50. `positions=true` includes host UTF-8/UTF-16 conversion for returned hits. All times below are milliseconds.", "",
         "| Scheme/query | Mode | Workers | Positions | Swift | Python | JS | CLI |",
         "|---|---|---:|---|---:|---:|---:|---:|"]


def pick(case, mode, workers=4, positions=False, language="swift"):
    return next(r for r in case["results"] if (r["mode"], r["workers"], r["positions"], r["language"]) == (mode, workers, positions, language))


for c in data["cases"]:
    for mode, workers, pos in [("json", 1, False), ("convenience", 4, False)] + [(m, w, p) for m in ("full", "top50") for w in (1, 4) for p in (False, True)]:
        values = [pick(c, mode, workers, pos, lang)["median_ms"] for lang in languages]
        lines.append(f"| {c['scheme']}/{c['query']} | {mode} | {workers} | {str(pos).lower()} | " + " | ".join(f"{v:.2f}" for v in values) + f" | {c['cli_median_ms']:.2f} |")
lines += ["", "## Highlighted top-50 latency", "", "| Scheme/query | Swift median/p95 | Python median/p95 | JS median/p95 |", "|---|---:|---:|---:|"]
for c in data["cases"]:
    values = [pick(c, "top50", 4, True, lang) for lang in languages]
    lines.append(f"| {c['scheme']}/{c['query']} | " + " | ".join(f"{r['median_ms']:.2f} / {r['p95_ms']:.2f}" for r in values) + " |")
lines += ["", "## Phase breakdown: default/bj", "",
          "Phase values are medians of wall time. `scan` includes conservative rejection and worker-local heap admission; `select` is merge/final sort. Worker scan/trace values are joined wall times, not summed CPU times. `native` also includes lock wait and query/result setup. Median components need not sum exactly to the median total. Host is per-sample total-native, covering query transport, FFI, row decoding, Unicode conversion and objects. JSON ABI 1 has only end-to-end timings; no invented JSON subphase allocation is reported.", "",
          "| Mode/workers/positions | Language | Create | Prepare | Scan | Select | Trace | C copy | Native | Host | Total |",
          "|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|"]
c = data["cases"][0]
for r in c["results"]:
    if "median_phases" not in r:
        continue
    p = r["median_phases"]
    values = [r["create_ms"], p["prepare"], p["scan"], p["select"], p["trace"], p["copy"], p["native"], r["host_median_ms"], r["median_ms"]]
    lines.append(f"| {r['mode']}/{r['workers']}/{str(r['positions']).lower()} | {r['language']} | " + " | ".join(f"{v:.3f}" for v in values) + " |")
lines += ["", "## Goal audit", "", "Goals came from the approved API survey, section 6; unmet cases are reported, not silently relaxed.", ""]
for lang in languages:
    full = [(c, pick(c, "full", 4, False, lang)) for c in data["cases"]]
    top = [(c, pick(c, "top50", 4, True, lang)) for c in data["cases"]]
    misses = [f"{c['scheme']}/{c['query']} ({r['median_ms']/c['cli_median_ms']:.2f}x)" for c, r in full if r["median_ms"] > 1.25*c["cli_median_ms"]]
    slow = [f"{c['scheme']}/{c['query']} ({r['median_ms']:.2f}/{r['p95_ms']:.2f}ms)" for c, r in top if r["median_ms"] > 50 or r["p95_ms"] > 100]
    host = max(r["host_median_ms"] + r["median_phases"]["copy"] for _, r in top)
    lines += [f"- {lang}: prepared full <=1.25x CLI: {len(full)-len(misses)}/{len(full)} cases; exceptions: {', '.join(misses) or 'none'}.",
              f"- {lang}: top50+highlight median<=50ms and p95<=100ms: {len(top)-len(slow)}/{len(top)} cases; exceptions: {', '.join(slow) or 'none'}.",
              f"- {lang}: max case-median top50 boundary + objects + C copy = {host:.3f}ms (goal <=5ms)."]
lines += ["", "The first nine scheme/query cases reproduce the original corpus queries; `b` is an extra initial-input case. All full-position paths and cold Index construction remain O(corpus/matches) work and must not be compared to rank-only/top-K as equivalent output. Retain an Index for interactive input. Prefix incremental work is deferred, so there is no incremental-vs-cold benchmark in this phase.", "",
          "The raw JSON preserves all samples, cold/create timings and per-call phase metrics. No isolated profile proves a single cause for every latency outlier; phase metrics identify native-vs-host costs, not system scheduler/thermal causality.", ""]
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text("\n".join(lines))
