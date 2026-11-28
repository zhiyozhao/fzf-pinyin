"""Dependency-free PEP 517 backend for a ctypes platform wheel.

The wheel has no CPython ABI dependency. Source builds require Go and a C
compiler; users installing wheels require neither. No global installs occur.
"""
import base64
import csv
import hashlib
import io
import os
from pathlib import Path
import shutil
import subprocess
import sys
import sysconfig
import tarfile
import tempfile
import zipfile

HERE = Path(__file__).resolve().parent
VERSION = "0.2.1"


def _root():
    if (HERE / "native/go.mod").exists():
        return HERE / "native"
    root = HERE.parent.parent
    if not (root / "go.mod").exists():
        raise RuntimeError("build from the fzfp checkout or its source distribution")
    return root


def build_wheel(wheel_directory, config_settings=None, metadata_directory=None):
    if sys.platform not in ("darwin", "linux"):
        raise RuntimeError("supported wheel platforms: macOS and Linux")
    platform = sysconfig.get_platform().replace("-", "_").replace(".", "_")
    machine = os.uname().machine
    if sys.platform == "darwin":
        machine = (config_settings or {}).get("arch", machine)
        if machine not in ("arm64", "x86_64"):
            raise RuntimeError("macOS arch must be arm64 or x86_64")
        platform = "macosx_13_0_" + machine
    tag = "py3-none-" + platform
    name = f"fzfp-{VERSION}-{tag}.whl"
    info = f"fzfp-{VERSION}.dist-info"
    with tempfile.TemporaryDirectory(prefix="fzfp-wheel-") as tmp:
        suffix = ".dylib" if sys.platform == "darwin" else ".so"
        library = Path(tmp) / ("libfzfp" + suffix)
        env = dict(os.environ, CGO_ENABLED="1")
        arch = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64"}.get(machine)
        if arch is None:
            raise RuntimeError("supported wheel CPUs: arm64 and x86_64")
        env.update(GOOS="darwin" if sys.platform == "darwin" else "linux", GOARCH=arch)
        if sys.platform == "darwin":
            env["MACOSX_DEPLOYMENT_TARGET"] = "13.0"
        subprocess.run(["go", "build", "-trimpath", "-buildvcs=false", "-buildmode=c-shared", "-o", str(library), "./capi"], cwd=_root(), env=env, check=True)
        files = {"fzfp/" + p.name: p.read_bytes() for p in (HERE / "fzfp").iterdir() if p.is_file()}
        files["fzfp/" + library.name] = library.read_bytes()
        files[info + "/METADATA"] = (f"Metadata-Version: 2.4\nName: fzfp\nVersion: {VERSION}\nRequires-Python: >=3.10\nLicense-Expression: MIT\nSummary: fzf pinyin shared-core matcher\n\n").encode()
        files[info + "/WHEEL"] = (f"Wheel-Version: 1.0\nGenerator: fzfp.backend\nRoot-Is-Purelib: false\nTag: {tag}\n").encode()
        for filename in ("LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md"):
            files[info + "/licenses/" + filename] = (_root() / filename).read_bytes()
        goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], text=True).strip())
        go_license = next((p for p in (goroot / "LICENSE", goroot.parent / "LICENSE") if p.is_file()), None)
        if go_license is None:
            raise RuntimeError("Go toolchain LICENSE is missing")
        files[info + "/licenses/Go-LICENSE"] = go_license.read_bytes()
        record = io.StringIO()
        writer = csv.writer(record, lineterminator="\n")
        for path, data in files.items():
            digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()
            writer.writerow([path, "sha256=" + digest, len(data)])
        writer.writerow([info + "/RECORD", "", ""])
        files[info + "/RECORD"] = record.getvalue().encode()
        Path(wheel_directory).mkdir(parents=True, exist_ok=True)
        with zipfile.ZipFile(Path(wheel_directory) / name, "w", zipfile.ZIP_DEFLATED) as wheel:
            for path, data in files.items():
                wheel.writestr(path, data)
    return name


def build_sdist(sdist_directory, config_settings=None):
    name = f"fzfp-{VERSION}.tar.gz"
    with tempfile.TemporaryDirectory(prefix="fzfp-sdist-") as tmp:
        stage = Path(tmp) / f"fzfp-{VERSION}"
        stage.mkdir()
        (stage / "PKG-INFO").write_text(f"Metadata-Version: 2.4\nName: fzfp\nVersion: {VERSION}\nRequires-Python: >=3.10\nLicense-Expression: MIT\n\n")
        for filename in ("backend.py", "pyproject.toml", "README.md"):
            shutil.copy2(HERE / filename, stage / filename)
        shutil.copytree(HERE / "fzfp", stage / "fzfp", ignore=shutil.ignore_patterns("__pycache__", "*.so", "*.dylib"))
        root = _root()
        for rel in ("go.mod", "go.sum", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md", "capi", "bindings/core", "src/algo", "src/pinyin", "src/util"):
            source, target = root / rel, stage / "native" / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            if source.is_dir():
                shutil.copytree(source, target, ignore=shutil.ignore_patterns("*_test.go"))
            else:
                shutil.copy2(source, target)
        Path(sdist_directory).mkdir(parents=True, exist_ok=True)
        with tarfile.open(Path(sdist_directory) / name, "w:gz") as archive:
            archive.add(stage, arcname=stage.name)
    return name
