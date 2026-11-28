"""Compare real release and candidate CLI bytes over deterministic long texts.

Download/verify the published archive first; pass its extracted binary as --old.
Only synthetic data is used. Inputs include embedded newlines via --read0.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import random
import subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--old", required=True, type=Path)
parser.add_argument("--new", required=True, type=Path)
parser.add_argument("--output", type=Path, default=Path(".cache/bindings/release-0.2.0-diff.json"))
args = parser.parse_args()
env = dict(os.environ, FZF_DEFAULT_OPTS="", FZF_DEFAULT_OPTS_FILE="/dev/null")
assert subprocess.check_output([str(args.old.resolve()), "--version"], env=env).strip() == b"fzfp 0.2.0 (based on fzf 0.75)"
rng = random.Random(20261128)
tokens = ["收货地址", "重庆", "重行音乐", "北京", "绿女略巨嗯", "𰻝👋", "abc", "fooBar", "café", "Ü", "/", " ", "-", "\n"]
mixed = ["".join(rng.choices(tokens, k=rng.randrange(2, 18))) for _ in range(2000)]
long = []
for i in range(300):
    head = tokens[i % len(tokens)]
    tail = tokens[(i + 3) % len(tokens)]
    gap = "q09 _\n" * (700 + rng.randrange(1500))
    long.append(head + gap + tail if i % 2 else gap + head + gap + tail)
huge = "synthetic line 你好 0123456789\n" * 39999
extreme = ["收货地址\n" + huge + "END", huge + "收货地址", "收" + "0" * 100001 + "货地址",
           "重行音乐" + "0" * 90000 + "重庆", "收货地址" + "0" * 90000 + "收货地址",
           "0" * 90000 + "𰻝👋", "abc" + "0" * 70000 + "田abc"]
queries = ["s", "shd", "shdz", "bj", "beijing", "chongqing", "yue", "lv", "lue", "jv", "en", "biang", "abc", "FB", "cafe", "Ü", "不存在", ""]
extended = ["^shdz", "shdz$", "^shdz$", "'shdz", "'abc", "'chongqing", "!shdz", "shdz | bj", "shdz !abc"]
cases = []
comparisons = 0
for name, texts in [("mixed", mixed), ("long", long), ("extreme", extreme)]:
    data = b"\0".join(t.encode() for t in texts) + b"\0"
    for scheme in ["default", "path", "history"]:
        searches = [(q, ["--no-extended"]) for q in queries]
        searches += [(q, ["--no-extended", "--exact"]) for q in queries]
        searches += [(q, ["--extended"]) for q in extended]
        searches += [("shd", ["--no-extended", "--tiebreak=" + tie]) for tie in ["end,length", "begin,length", "pathname,length", "chunk,end"]]
        for query, options in searches:
            cli_args = ["--read0", "--print0", "-i", "--scheme=" + scheme, "--filter=" + query] + options
            old = subprocess.run([str(args.old.resolve())] + cli_args, input=data, capture_output=True, env=env, timeout=180)
            new = subprocess.run([str(args.new.resolve())] + cli_args, input=data, capture_output=True, env=env, timeout=180)
            if (old.returncode, old.stdout, old.stderr) != (new.returncode, new.stdout, new.stderr):
                raise AssertionError(f"CLI mismatch: {name}/{scheme}/{query!r}/{options}: exit={old.returncode}/{new.returncode}, stdout={hashlib.sha256(old.stdout).hexdigest()}/{hashlib.sha256(new.stdout).hexdigest()}, stderr={old.stderr!r}/{new.stderr!r}")
            assert old.returncode in (0, 1), old.stderr
            comparisons += len(texts)
            cases.append(dict(corpus=name, scheme=scheme, query=query, options=options, rows=len(texts), input_bytes=len(data), output_sha256=hashlib.sha256(new.stdout).hexdigest(), status=new.returncode))
        print(f"PASS {name}/{scheme}: {len(searches)} CLI comparisons, {len(texts)} candidates each", flush=True)
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text(json.dumps(dict(searches=len(cases), candidate_comparisons=comparisons, cases=cases), indent=2) + "\n")
print(f"PASS {len(cases)} old/new CLI comparisons; {comparisons} candidate/query comparisons")
