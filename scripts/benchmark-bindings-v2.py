"""Same-session 100k differential benchmark; install wheel/build Swift+JS first.

Warm results exclude one explicitly recorded warmup and corpus construction.
Native phase metrics are wall times; scan includes local heap admission. Host
overhead = total - native, including FFI, decode, coordinate conversion and
objects. No incremental cache is used. Raw samples are preserved.
"""
import argparse
import json
import math
import os
from pathlib import Path
import platform
import statistics
import subprocess
import tempfile
import time

import fzfp

parser = argparse.ArgumentParser()
parser.add_argument("--repetitions", type=int, default=7)
parser.add_argument("--top-repetitions", type=int, default=21)
parser.add_argument("--output", type=Path, default=Path(__file__).resolve().parents[1] / ".cache/bindings/benchmark-v2.json")
parser.add_argument("--schemes", nargs="+", choices=("default", "path", "history"), default=["default", "path", "history"])
parser.add_argument("--queries", nargs="+", default=["bj", "beijing", "fb", "b"])
parser.add_argument("--top-only", action="store_true", help="measure only prepared top-50, 4 workers, highlights")
args = parser.parse_args()
args.output.parent.mkdir(parents=True, exist_ok=True)
root = Path(__file__).resolve().parents[1]
cli = root / ".cache/bindings/fzfp"
swift = root / "bindings/swift/.build/release/fzfp-benchmark"
node_driver = root / "bindings/javascript/benchmark.js"
words = ["北京", "中国", "绿茶", "马超", "fooBar", "b-j", "bj", "beijing", "project", "长沙"]
candidates = [f"workspace/{i:06d}/{words[i % 10]}/notes-{i % 97}.md" for i in range(100_000)]
text = "\n".join(candidates) + "\n"
report = {"machine": platform.machine(), "system": platform.system(), "rows": len(candidates),
          "repetitions": args.repetitions, "top_repetitions": args.top_repetitions, "cases": []}
env = dict(os.environ, FZF_DEFAULT_OPTS="", FZF_DEFAULT_OPTS_FILE="/dev/null")


def summary(data):
    samples = data["milliseconds"]
    data["median_ms"] = statistics.median(samples)
    data["p95_ms"] = sorted(samples)[math.ceil(len(samples) * .95) - 1]
    if data["phases"]:
        data["median_phases"] = {key: statistics.median(p[key] for p in data["phases"]) for key in data["phases"][0]}
        data["host_ms"] = [t - p["native"] for t, p in zip(samples, data["phases"])]
        data["host_median_ms"] = statistics.median(data["host_ms"])
    return data


def python_run(query, scheme, repetitions, mode, workers, positions):
    options = fzfp.Options(scheme=scheme, positions=positions)
    start = time.perf_counter()
    index = None if mode in ("json", "convenience") else fzfp.Index(candidates, options, workers=workers)
    create = (time.perf_counter() - start) * 1000
    def run():
        start = time.perf_counter()
        if mode == "json":
            rows = fzfp._call(dict(operation="filter", candidates=candidates, query=query, options=vars(options))).get("results", [])
            hits = [fzfp.Result(**h, text=candidates[h["index"]]) for h in rows]
        elif mode == "convenience":
            hits = fzfp.filter(candidates, query, options)
        else:
            hits = index.find(query, 50 if mode == "top50" else None, positions)
        duration = (time.perf_counter() - start) * 1000
        return hits, duration
    _, cold = run()
    milliseconds, phases = [], []
    for _ in range(repetitions):
        hits, duration = run()
        milliseconds.append(duration)
        if index:
            phases.append(dict(index.timings))
    if index:
        index.close()
    return dict(milliseconds=milliseconds, phases=phases, create_ms=create, cold_ms=cold,
                indexes=[h.index for h in hits], scores=[h.score for h in hits])


with tempfile.TemporaryDirectory(prefix="fzfp-benchmark-v2-") as tmp:
    corpus = Path(tmp) / "corpus.txt"
    corpus.write_text(text)
    for scheme in args.schemes:
        for query in args.queries:
            cli_times = []
            for _ in range(args.repetitions):
                start = time.perf_counter()
                result = subprocess.run([str(cli), "--no-extended", "-i", "--scheme=" + scheme, "--filter=" + query],
                                        input=text, text=True, capture_output=True, check=True, env=env)
                cli_times.append((time.perf_counter() - start) * 1000)
            expected_texts = result.stdout.splitlines()
            case = dict(scheme=scheme, query=query, matches=len(expected_texts), cli_ms=cli_times,
                        cli_median_ms=statistics.median(cli_times), results=[])
            configs = [("json", 1, False), ("convenience", 4, False)]
            configs += [(mode, w, pos) for mode in ("full", "top50") for w in (1, 4) for pos in (False, True)]
            if args.top_only:
                configs = [("top50", 4, True)]
                # Independent full baseline remains outside the top-K timing.
                baseline = fzfp.filter(candidates, query, fzfp.Options(scheme=scheme))
                oracle = [(h.index, h.score) for h in baseline]
                assert [h.text for h in baseline] == expected_texts, (scheme, query, "CLI full order")
            else:
                oracle = None
            for mode, workers, positions in configs:
                repetitions = args.top_repetitions if mode == "top50" and workers == 4 and positions else args.repetitions
                for language in ("python", "swift", "js"):
                    if language == "python":
                        data = python_run(query, scheme, repetitions, mode, workers, positions)
                    else:
                        command = [str(swift)] if language == "swift" else ["node", str(node_driver)]
                        data = json.loads(subprocess.check_output(command + [str(corpus), query, scheme, str(repetitions), mode, str(workers), str(positions).lower()]))
                    indexes, scores = data.pop("indexes"), data.pop("scores")
                    if oracle is None:
                        oracle = list(zip(indexes, scores))
                        assert [candidates[i] for i in indexes] == expected_texts, (scheme, query, "CLI order")
                    want = oracle[:50] if mode == "top50" else oracle
                    assert list(zip(indexes, scores)) == want, (scheme, query, mode, language, workers, positions)
                    summary(data)
                    case["results"].append(dict(language=language, mode=mode, workers=workers, positions=positions, **data))
                print(scheme, query, mode, workers, positions, {r["language"]: round(r["median_ms"], 2) for r in case["results"][-3:]}, flush=True)
            report["cases"].append(case)
            args.output.write_text(json.dumps(report, indent=2) + "\n")
