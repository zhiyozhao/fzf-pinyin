"""Run with the wheel installed; build Swift release benchmark separately.

All timed calls include transport/materialization but exclude corpus file load.
CLI timings include process startup and stdin/stdout. Every complete output
order is checked. No selective top-K comparison hides materialization costs.
"""
import argparse
import json
import os
from pathlib import Path
import platform
import statistics
import subprocess
import tempfile
import time

import fzfp

parser = argparse.ArgumentParser()
parser.add_argument("--repetitions", type=int, default=5)
parser.add_argument("--output", type=Path)
args = parser.parse_args()
root = Path(__file__).resolve().parents[1]
cli = root / ".cache/bindings/fzfp"
swift = root / "bindings/swift/.build/release/fzfp-benchmark"
words = ["北京", "中国", "绿茶", "马超", "fooBar", "b-j", "bj", "beijing", "project", "长沙"]
candidates = [f"workspace/{i:06d}/{words[i % len(words)]}/notes-{i % 97}.md" for i in range(100_000)]
report = {"machine": platform.machine(), "system": platform.system(), "rows": len(candidates), "repetitions": args.repetitions, "positions": False, "results": []}
env = dict(os.environ, FZF_DEFAULT_OPTS="", FZF_DEFAULT_OPTS_FILE="/dev/null")
with tempfile.TemporaryDirectory(prefix="fzfp-benchmark-") as tmp:
    corpus = Path(tmp) / "corpus.txt"
    text = "\n".join(candidates) + "\n"
    corpus.write_text(text)
    for scheme in ("default", "path", "history"):
        for query in ("bj", "beijing", "fb"):
            opts = fzfp.Options(scheme=scheme, positions=False)
            times = []
            for _ in range(args.repetitions):
                start = time.perf_counter()
                hits = fzfp.filter(candidates, query, opts)
                times.append((time.perf_counter() - start)*1000)
            expected = [r.text for r in hits]
            indexes = [r.index for r in hits]
            cli_times = []
            for _ in range(args.repetitions):
                start = time.perf_counter()
                result = subprocess.run([str(cli), "--no-extended", "-i", "--scheme="+scheme, "--filter="+query], input=text, text=True, capture_output=True, check=True, env=env)
                cli_times.append((time.perf_counter() - start)*1000)
                assert result.stdout.splitlines() == expected, (scheme, query, "CLI order mismatch")
            swift_result = json.loads(subprocess.check_output([str(swift), str(corpus), query, scheme, str(args.repetitions)]))
            assert swift_result["indexes"] == indexes, (scheme, query, "Swift order mismatch")
            row = {"scheme": scheme, "query": query, "matches": len(hits), "python_ms": times, "swift_ms": swift_result["milliseconds"], "cli_ms": cli_times}
            row["medians_ms"] = {name: statistics.median(row[name+"_ms"]) for name in ("python", "swift", "cli")}
            report["results"].append(row)
            print(scheme, query, row["medians_ms"], flush=True)
if args.output:
    args.output.write_text(json.dumps(report, indent=2)+"\n")
