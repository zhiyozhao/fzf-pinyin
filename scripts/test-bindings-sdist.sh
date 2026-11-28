#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root="$PWD"
task_dir="$(mktemp -d /tmp/fzfp-sdist-v2.XXXXXX)"
trap 'rm -r "$task_dir"' EXIT
FZFP_SDIST_DEST="$task_dir" python3 -c 'import os,sys; sys.path.insert(0,"bindings/python"); import backend; print(backend.build_sdist(os.environ["FZFP_SDIST_DEST"]))'
tar -xzf "$task_dir/fzfp-0.2.1.tar.gz" -C "$task_dir"
python3 -m venv "$task_dir/venv"
python3 -m pip wheel --no-deps --wheel-dir "$task_dir/wheels" "$task_dir/fzfp-0.2.1"
"$task_dir/venv/bin/python" -m pip install --no-deps "$task_dir"/wheels/*.whl
FZFP_CLI="$root/.cache/bindings/fzfp" "$task_dir/venv/bin/python" -m unittest discover -s bindings/python/tests -v
