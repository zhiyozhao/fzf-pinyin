import concurrent.futures
import ctypes
from dataclasses import asdict
import json
import os
from pathlib import Path
import subprocess
import unittest

import fzfp


class BindingTests(unittest.TestCase):
    def test_golden(self):
        path = Path(os.environ.get("FZFP_GOLDEN", Path(__file__).resolve().parents[2] / "testdata/golden.json"))
        fixture = json.loads(path.read_text())
        self.assertEqual(fzfp.version()["core_revision"], fixture["core_revision"])
        self.assertEqual(fzfp.version()["abi_version"], fixture["abi_version"])
        for vector in fixture["vectors"]:
            with self.subTest(vector=vector["name"]):
                req = vector["request"]
                opts = fzfp.Options(**req["options"])
                if req["operation"] == "match":
                    self.assertEqual(asdict(fzfp.match(req["text"], req["query"], opts)), vector["expected"]["match"])
                else:
                    results = fzfp.filter(req["candidates"], req["query"], opts)
                    self.assertEqual([{k:v for k,v in asdict(r).items() if k != "text"} for r in results], vector["expected"].get("results", []))
                texts = [req["text"]] if req["operation"] == "match" else req["candidates"]
                expected = ([dict(index=0, **vector["expected"]["match"])] if vector["expected"].get("match", {}).get("matched") else []) if req["operation"] == "match" else vector["expected"].get("results", [])
                for workers in (1, 4):
                    with fzfp.Index(texts, opts, workers=workers) as index:
                        for k in (None, 0, 1, 7, 50):
                            want = expected if k is None else expected[:k]
                            for positions in (False, True):
                                actual = index.find(req["query"], k, positions)
                                self.assertEqual([(h.index, h.score) for h in actual], [(h["index"], h["score"]) for h in want])
                                if positions and opts.positions:
                                    self.assertEqual([asdict(h.alignment) for h in actual], [{key:value for key,value in h.items() if key != "index"} for h in want])
        print(f"Verified {len(fixture['vectors'])} shared golden vectors")

    def test_concurrency(self):
        def run(i):
            scheme = ("default", "path", "history")[i % 3]
            return scheme, fzfp.match("/马超", "machao", fzfp.Options(scheme=scheme)).score
        with concurrent.futures.ThreadPoolExecutor(8) as pool:
            for scheme, score in pool.map(run, range(300)):
                self.assertEqual(score, 151 if scheme == "history" else 158)

    def test_position_toggle_preserves_alignment(self):
        for text, query in (("ab abb", "abb"), ("a-b aab", "ab"), ("中国中国", "zg"), ("xx/path/fooBar", "fb")):
            with_positions = fzfp.match(text, query)
            without = fzfp.match(text, query, fzfp.Options(positions=False))
            self.assertEqual((with_positions.score, with_positions.start, with_positions.end), (without.score, without.start, without.end))
            self.assertEqual(without.positions, [])

    def test_errors_and_ownership(self):
        for request in (b"{", b"null", b"{} {}", b"\xff", b'{"operation":"unknown"}', b'{"operation":"match","extra":1}'):
            size, status = ctypes.c_size_t(), ctypes.c_int32()
            pointer = fzfp._lib.fzfp_call(request, len(request), ctypes.byref(size), ctypes.byref(status))
            self.assertTrue(pointer)
            try:
                self.assertEqual(status.value, 1)
                self.assertIn("error", json.loads(ctypes.string_at(pointer, size.value)))
            finally:
                fzfp._lib.fzfp_free(pointer)
        with self.assertRaises(ValueError):
            fzfp.match("中国", "zg", fzfp.Options(scheme="bad"))
        for _ in range(1000):
            self.assertEqual(fzfp.match("a\0中国", "zg").positions, [2, 3])
        self.assertEqual(fzfp.filter([], "x"), [])
        index = fzfp.Index(["北京"])
        with self.assertRaises(ValueError):
            index.find("bj", -1)
        handle = index._handle.value
        index.close()
        index.close()
        self.assertEqual(fzfp._lib.fzfp_index_destroy(handle), 3)
        with self.assertRaises(RuntimeError):
            index.find("bj")
        with self.assertRaises(ValueError):
            fzfp.Index([], workers=65)

    def test_same_index_concurrency_and_close(self):
        index = fzfp.Index(["北京"] * 1100)
        def run(_):
            try:
                hits = index.find("bj", 7, True)
                self.assertEqual([h.index for h in hits], list(range(7)))
            except RuntimeError:
                pass  # close won the serialized wrapper lock
        with concurrent.futures.ThreadPoolExecutor(8) as pool:
            tasks = [pool.submit(run, i) for i in range(20)]
            index.close()
            for task in tasks:
                task.result()

    def test_cli_order(self):
        binary = os.environ.get("FZFP_CLI")
        if not binary:
            self.skipTest("set FZFP_CLI for CLI differential test")
        candidates = [f"{prefix}/{word}/{i}.md" for i in range(40) for prefix in ("a", "long/path", " 北京 ") for word in ("北京", "b-j", "bj", "beijing", "中国")]
        candidates += ["北京", "北京", " 北京 ", "😀北京", "\t北京\t", "fooBar", "café"]
        for scheme in ("default", "path", "history"):
            for query in ("bj", "beijing", "zg", "fb", "cafe", ""):
                with self.subTest(scheme=scheme, query=query):
                    env = dict(os.environ, FZF_DEFAULT_OPTS="", FZF_DEFAULT_OPTS_FILE="/dev/null")
                    result = subprocess.run([binary, "--no-extended", "-i", "--scheme="+scheme, "--filter="+query], input="\n".join(candidates)+"\n", text=True, capture_output=True, check=True, env=env)
                    self.assertEqual([r.text for r in fzfp.filter(candidates, query, fzfp.Options(scheme=scheme))], result.stdout.splitlines())
                    self.assertEqual([r.text for r in fzfp.filter(candidates, query, fzfp.Options(scheme=scheme, positions=False))], result.stdout.splitlines())


if __name__ == "__main__":
    unittest.main()
