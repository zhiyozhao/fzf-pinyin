"""Shared-core pinyin matching. Positions are Python str (Unicode scalar) indexes."""
from __future__ import annotations

import ctypes
from dataclasses import dataclass
import json
from pathlib import Path
import sys
from typing import Literal, Sequence, TypedDict

Scheme = Literal["default", "path", "history"]


@dataclass(frozen=True)
class Options:
    scheme: Scheme = "default"
    case_sensitive: bool = False
    normalize: bool = True
    positions: bool = True


@dataclass(frozen=True)
class Match:
    matched: bool
    score: int
    start: int
    end: int
    positions: list[int]
    utf8_positions: list[int]
    utf16_positions: list[int]


@dataclass(frozen=True)
class Result(Match):
    index: int
    text: str


class Version(TypedDict):
    version: str
    core_revision: str
    abi_version: int
    dictionary_version: str


_lib = ctypes.CDLL(str(Path(__file__).with_name("libfzfp.dylib" if sys.platform == "darwin" else "libfzfp.so")))
_lib.fzfp_abi_version.argtypes = []
_lib.fzfp_abi_version.restype = ctypes.c_uint32
if _lib.fzfp_abi_version() != 1:
    raise ImportError("fzfp ABI mismatch (expected 1)")
_lib.fzfp_call.argtypes = [ctypes.c_char_p, ctypes.c_size_t, ctypes.POINTER(ctypes.c_size_t), ctypes.POINTER(ctypes.c_int32)]
_lib.fzfp_call.restype = ctypes.c_void_p
_lib.fzfp_free.argtypes = [ctypes.c_void_p]
_lib.fzfp_free.restype = None


def _call(request: dict) -> dict:
    data = json.dumps(request, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    size, status = ctypes.c_size_t(), ctypes.c_int32()
    pointer = _lib.fzfp_call(data, len(data), ctypes.byref(size), ctypes.byref(status))
    if not pointer:
        raise MemoryError("fzfp returned a null response")
    try:
        response = json.loads(ctypes.string_at(pointer, size.value))
    finally:
        _lib.fzfp_free(pointer)
    if status.value:
        error = ValueError if status.value == 1 else RuntimeError
        raise error(response.get("error", "fzfp call failed"))
    return response


def version() -> Version:
    return _call({"operation": "version"})


def match(text: str, query: str, options: Options = Options()) -> Match:
    """Match one literal fuzzy term (no extended-query operators or smart case)."""
    return Match(**_call({"operation": "match", "text": text, "query": query, "options": vars(options)})["match"])


def filter(candidates: Sequence[str], query: str, options: Options = Options()) -> list[Result]:
    """Batch match and sort in Go, retaining original indexes and duplicates."""
    texts = list(candidates)
    hits = _call({"operation": "filter", "candidates": texts, "query": query, "options": vars(options)}).get("results", [])
    return [Result(**hit, text=texts[hit["index"]]) for hit in hits]


__all__ = ["Options", "Match", "Result", "Scheme", "Version", "match", "filter", "version"]
