"""Create deterministic Swift archives, or verify a reviewed release asset.

Only preparation writes files, under --directory (ignored by default). The
provenance is outside the zip so packaging-only commits do not change its hash.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import stat
import subprocess
import zipfile

ROOT = Path(__file__).resolve().parents[1]
# These complete subtrees deliberately include tests: a release may change the
# root manifest or release pin, but not silently replace the reviewed core,
# wrapper, public headers, dependencies, licenses, or asset-building scripts.
INPUTS = ["src", "bindings/core", "bindings/testdata", "bindings/swift/Sources", "bindings/swift/Tests", "bindings/swift/Consumer", "capi",
          "go.mod", "go.sum", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md",
          "scripts/build-bindings.sh", "scripts/swift-release.py", "scripts/test-swift-consumer.sh"]


def command(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True).strip()


def inputs_digest():
    names = subprocess.check_output(["git", "ls-files", "-z", "--"] + INPUTS, cwd=ROOT).split(b"\0")
    result = hashlib.sha256()
    for name in sorted(n for n in names if n):
        result.update(name + b"\0" + hashlib.sha256((ROOT / name.decode()).read_bytes()).digest())
    return result.hexdigest()


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def prepare(directory):
    assert not command("git", "status", "--porcelain", "--", *INPUTS), "commit native inputs before preparing release assets"
    framework = ROOT / "bindings/swift/.cache/CFzfp.xcframework"
    directory.mkdir(parents=True, exist_ok=True)
    archive = directory / "CFzfp.xcframework.zip"
    with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as out:
        for path in sorted(framework.rglob("*")):
            assert not path.is_symlink(), f"unexpected framework symlink: {path}"
            if path.is_file():
                info = zipfile.ZipInfo(str(Path(framework.name) / path.relative_to(framework)), (1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (stat.S_IFREG | 0o644) << 16
                out.writestr(info, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)
    checksum = sha256(archive)
    (directory / "CFzfp.xcframework.zip.sha256").write_text(f"{checksum}  {archive.name}\n")
    provenance = dict(format_version=1, version="0.2.1", source_commit=command("git", "rev-parse", "HEAD"),
                      inputs_sha256=inputs_digest(), archive_sha256=checksum,
                      go=command("go", "version"), xcode=command("xcodebuild", "-version"),
                      sdk=command("xcrun", "--sdk", "macosx", "--show-sdk-version"),
                      clang=command("xcrun", "clang", "--version"),
                      python=command("python3", "--version"), zlib=zipfile.zlib.ZLIB_VERSION,
                      deployment_target="13.0", architectures=["arm64", "x86_64"])
    (directory / "CFzfp.provenance.json").write_text(json.dumps(provenance, indent=2, sort_keys=True) + "\n")
    print(checksum)


def verify(directory, metadata):
    pin = json.loads(metadata.read_text())
    provenance = json.loads((directory / "CFzfp.provenance.json").read_text())
    archive = directory / "CFzfp.xcframework.zip"
    assert pin["version"] == provenance["version"] == "0.2.1"
    assert pin["source_commit"] == provenance["source_commit"]
    subprocess.run(["git", "merge-base", "--is-ancestor", pin["source_commit"], "HEAD"], cwd=ROOT, check=True)
    assert inputs_digest() == provenance["inputs_sha256"], "tag inputs differ from reviewed native/wrapper source"
    assert sha256(archive) == provenance["archive_sha256"] == pin["sha256"], "reviewed asset checksum differs"
    assert (directory / "CFzfp.xcframework.zip.sha256").read_text() == f"{pin['sha256']}  {archive.name}\n"
    manifest = (ROOT / "Package.swift").read_text()
    assert f'checksum: "{pin["sha256"]}"' in manifest
    assert 'url: "https://github.com/zhiyozhao/fzf-pinyin/releases/download/fzfp-v0.2.1/CFzfp.xcframework.zip"' in manifest
    core = (ROOT / "bindings/core/core.go").read_text()
    baseline = re.search(r'const CoreRevision = "([0-9a-f]{40})"', core)[1]
    subprocess.run(["git", "merge-base", "--is-ancestor", baseline, pin["source_commit"]], cwd=ROOT, check=True)
    print(f"Verified 0.2.1 asset, source {pin['source_commit']}, SHA256 {pin['sha256']}")


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("action", choices=["prepare", "verify"])
parser.add_argument("--directory", type=Path, default=ROOT / ".cache/swift-release")
parser.add_argument("--metadata", type=Path, default=ROOT / ".github/swift-release.json")
args = parser.parse_args()
if args.action == "prepare":
    prepare(args.directory)
else:
    verify(args.directory, args.metadata)
