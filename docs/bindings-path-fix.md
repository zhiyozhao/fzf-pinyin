# Path direction correction and independent CLI regression

The independent gate on `662cfd7b` found a correctness bug: the bindings always
passed `forward=true`, but CLI path defaults use backward alignment. This can
change exact bounds, positions and pathname tie ranking without changing score.
For `case-test_qux` / `st`, default alignment is [2,5]; path must use [7,8]
in the binding's ascending position convention (native traceback is [8,7]).
The six-candidate gate example now puts this candidate fourth, like the CLI.

## Complete direction rule

`src/core.go` starts with forward=true, withPos=false, then traverses effective
criteria from last to first, excluding score. `byEnd` selects backward,
`byBegin` forward, `byPathname` backward plus positions, `byChunk` positions
only. Thus the highest-priority direction criterion wins; chunk doesn't
override direction selected by a lower-priority key. Length/index do not select
direction. The binding helper models the whole loop, not just a path special
case. Both Execute and Index.Find (scan and returned alignment) use it.

`parseScheme`: default = score,length; history = score; path =
score,pathname,length. Therefore default/history are forward and path backward.
Both --scheme and --tiebreak replace effective criteria, so CLI option order
matters. --tac changes index/output order; --no-sort changes sorting/streaming,
neither changes forward. --nth/delimiter and fuzzy/exact choose input/algorithm
without altering this loop. The CLI's implicit TTY/walker scheme selection and
FZF_DEFAULT_OPTS affect effective options; the library has an explicit scheme
and doesn't inherit CLI environment. Arbitrary tiebreak, tac, no-sort, fields,
smart case and extended syntax remain unsupported binding options.

The shared `bindings/testdata/direction.json` tests eleven tiebreak combinations,
including begin/end/pathname priority and chunk independence. A spy in the real
Run patternBuilder observes the actual forward/withPos arguments for all three
schemes; extra probes verify --tac, --no-sort and --scheme/--tiebreak ordering.
These probes do not use the binding helper to determine expected CLI policy.

## Why prior testing missed this

The golden generator used Execute itself; the Index differential used Execute
as oracle. Even the direct primitive test hardcoded the same forward=true.
These agreed with each other while sharing the same wrong direction assumption.
The earlier CLI comparisons used mixed paths/queries (bj/beijing/fb) with few
equal-score alternative alignments. Their agreement was true for those inputs,
but did not establish path correctness for tie-heavy data. The performance
corpus likewise did not expose this alignment-distribution gap. Prior benchmark
files remain historical measurements of the old implementation, not proof that
its path alignment was correct. Current golden data was regenerated, with the
gate's explicit reproduction added for every scheme (189 vectors total).

## Permanent live oracle

`src/bindings_oracle_test.go` generates 420 deterministic datasets per scheme:
pure Latin, Han/pinyin and mixed text; repeated letters/characters, short
queries, delimiters, whitespace, duplicate records, emoji/combining characters,
and selected >1024-row batches to exercise multiple workers. Seed is fixed.

For each of 1260 datasets:

1. Real built `fzfp --filter` is invoked with --read0/--print0 and clean default
   options. Complete NUL-delimited output is compared byte-for-byte.
2. Expected rank/bounds/positions come from the CLI's real
   BuildPattern/MatchItem/buildResult/ByRelevance path using direction observed
   in Run, independently of binding Execute/Find. Short test records keep scores
   below rank saturation, allowing exact scores from the CLI Result score key.
3. Execute JSON values are compared byte-for-byte against that oracle. Index
   verifies 1/4 workers, unlimited/1/7/50 limits, position toggles, scores and
   exact requested alignment against the same oracle.
4. The live fixtures, generated into ignored `.cache/bindings/cli-oracle`, are
   consumed by Swift/Python/JS. Each checks full convenience output, complete
   text byte order, 1/4-worker Index full/top-50, scores, bounds, canonical
   positions and UTF-8/UTF-16 offsets.

CLI --filter does not output scores or positions; the test does not claim
otherwise. Its textual output independently anchors ordering. The test-only
CLI pipeline instrumentation anchors score/alignment. Traceback positions are
canonicalized ascending once to honor the existing binding coordinate contract;
backward traceback order is not confused with a different alignment.

Run `bash scripts/test-bindings.sh` after building artifacts. It builds the CLI,
generates/validates all live oracle datasets, and requires all three languages
to consume them. Standalone language tests explicitly skip this one test if the
live-fixture environment is absent; the unified suite and bindings CI set it
and require exactly 1260 datasets. `go test ./...` also runs the live CLI
differential, building a temporary CLI if no FZFP_CLI is supplied. No fixture
generated from stale binding output is checked in as an independent oracle.

## Gate suggestions

- S1: `_json` is declared with @internal in TypeScript.
- S2: Python rejects bool for limit and workers; regressions included.
- S3: addressed by the 1260-case live CLI/pipeline differential above.
- S4: a strict TypeScript consumer test and pinned tsc CI step are supplied;
  not executed locally because no existing tsc is installed. No new local tool
  was installed. CI execution is still a release prerequisite.
- S5: Linux manylinux CI adds C ASan with detect_leaks=1. Not run locally;
  macOS cannot substitute for the requested Linux leak check.
- S6: Swift README explains local path identity `swift` and explicit
  `.product(name: "Fzfp", package: "swift")`.

## Corrected performance

Fresh measurements for default/bj and path/bj, top-50 with alignment and four
workers, are recorded in `bindings-benchmark-path-fix.json` after all tests.
Corpus construction remains separate. One cold query precedes 21 warm samples.
Every returned index/score is checked against full results validated by CLI
text order. The raw results include native phase metrics and host overhead.
