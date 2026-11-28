"""Shared-core pinyin matching. Positions are Python str (Unicode scalar) indexes."""
from __future__ import annotations

import ctypes
from dataclasses import dataclass
import json
from pathlib import Path
import sys
import threading
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
    typed_abi_version: int


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
    return dict(_call({"operation": "version"}), typed_abi_version=2)


def match(text: str, query: str, options: Options = Options(), *, exact: bool = False) -> Match:
    """Match one literal fuzzy term (no extended-query operators or smart case)."""
    with Index([text], options) as index:
        hits = index._find(query, None, options.positions, True, exact)
        return _alignment(text, hits[0]) if hits else Match(False, 0, -1, -1, [], [], [])


def filter(candidates: Sequence[str], query: str, options: Options = Options(), *, exact: bool = False) -> list[Result]:
    """Batch match and sort in Go, retaining original indexes and duplicates."""
    with Index(candidates, options) as index:
        return [Result(**vars(_alignment(index.candidates[h[0]], h)), index=h[0], text=index.candidates[h[0]])
                for h in index._find(query, None, options.positions, True, exact)]


class _Options(ctypes.Structure):
    _fields_ = [(n, ctypes.c_uint32) for n in ("abi", "size", "scheme", "case", "normalize", "workers", "reserved0", "reserved1")]


class _FindOptions(ctypes.Structure):
    _fields_ = [("abi", ctypes.c_uint32), ("size", ctypes.c_uint32), ("limit", ctypes.c_int64),
                ("flags", ctypes.c_uint32), ("reserved", ctypes.c_uint32)]


class _Hit(ctypes.Structure):
    _fields_ = [("index", ctypes.c_uint64), ("score", ctypes.c_int64), ("start", ctypes.c_int64),
                ("end", ctypes.c_int64), ("offset", ctypes.c_uint64), ("count", ctypes.c_uint64)]


class _Results(ctypes.Structure):
    _fields_ = [("abi", ctypes.c_uint32), ("size", ctypes.c_uint32), ("count", ctypes.c_uint64),
                ("position_count", ctypes.c_uint64), ("hits", ctypes.POINTER(_Hit)),
                ("positions", ctypes.POINTER(ctypes.c_uint64))] + [
                    (n + "_ms", ctypes.c_double) for n in ("prepare", "scan", "select", "trace", "copy", "native")]


_lib.fzfp_typed_abi_version.restype = ctypes.c_uint32
if _lib.fzfp_typed_abi_version() != 2:
    raise ImportError("fzfp typed ABI mismatch (expected 2)")
_lib.fzfp_index_create.argtypes = [ctypes.c_char_p, ctypes.c_uint64, ctypes.POINTER(ctypes.c_uint64),
                                   ctypes.c_uint64, ctypes.POINTER(_Options), ctypes.POINTER(ctypes.c_uint64)]
_lib.fzfp_index_create.restype = ctypes.c_int32
_lib.fzfp_index_find.argtypes = [ctypes.c_uint64, ctypes.c_char_p, ctypes.c_uint64,
                                 ctypes.POINTER(_FindOptions), ctypes.POINTER(ctypes.POINTER(_Results))]
_lib.fzfp_index_find.restype = ctypes.c_int32
_lib.fzfp_index_destroy.argtypes = [ctypes.c_uint64]
_lib.fzfp_index_destroy.restype = ctypes.c_int32
_lib.fzfp_results_free.argtypes = [ctypes.POINTER(_Results)]
_lib.fzfp_results_free.restype = None


def _check(status: int) -> None:
    if status:
        if status == 4:
            raise MemoryError("fzfp native allocation failed")
        error = ValueError if status == 1 else RuntimeError
        raise error({1: "invalid fzfp input or options", 2: "fzfp internal panic", 3: "closed or invalid fzfp index"}.get(status, "fzfp failed"))


def _alignment(text: str, row: tuple) -> Match:
    _, score, start, end, positions = row
    u8, u16 = [], []
    if not positions:
        return Match(True, score, start, end, positions, u8, u16)
    wanted = set(positions)
    b = u = 0
    for i, c in enumerate(text):
        if i in wanted:
            u8.append(b)
            u16.append(u)
        b += len(c.encode("utf-8"))
        u += 2 if ord(c) > 0xffff else 1
    return Match(True, score, start, end, positions, u8, u16)


@dataclass(frozen=True)
class Hit:
    index: int
    score: int
    alignment: Match | None

    @property
    def positions(self) -> list[int]:
        return self.alignment.positions if self.alignment else []


class Index:
    """Owned immutable corpus. Queries/close serialize; native calls release GIL.

    limit=None returns all; limit=0 returns none. Positions are opt-in. Use a
    context manager or close(). Async callers may compose asyncio.to_thread;
    cancelling the await does not interrupt a native query.
    """
    def __init__(self, candidates: Sequence[str], options: Options = Options(), *, workers: int = 0):
        self._lock = threading.RLock()
        self._handle = ctypes.c_uint64()
        self._candidates = tuple(candidates)
        self.timings: dict[str, float] = {}
        if options.scheme not in ("default", "path", "history") or type(workers) is not int or not 0 <= workers <= 64:
            raise ValueError("invalid scheme or workers (expected 0..64)")
        chunks = [s.encode("utf-8") for s in self.candidates]
        offsets = [0]
        for chunk in chunks:
            offsets.append(offsets[-1] + len(chunk))
        data = b"".join(chunks)
        native = _Options(2, ctypes.sizeof(_Options), ("default", "path", "history").index(options.scheme),
                          options.case_sensitive, options.normalize, workers, 0, 0)
        _check(_lib.fzfp_index_create(data, len(data), (ctypes.c_uint64 * len(offsets))(*offsets),
                                     len(chunks), ctypes.byref(native), ctypes.byref(self._handle)))

    def _find(self, query: str, limit: int | None, positions: bool, exact_bounds: bool = False, exact: bool = False) -> list[tuple]:
        if limit is not None and (type(limit) is not int or not 0 <= limit <= 2**63 - 1):
            raise ValueError("limit must be None or a nonnegative int64")
        if type(positions) is not bool or type(exact) is not bool:
            raise ValueError("positions and exact must be bool")
        data = query.encode("utf-8")
        options = _FindOptions(2, ctypes.sizeof(_FindOptions), -1 if limit is None else limit,
                               int(positions) | (2 if exact_bounds else 0) | (4 if exact else 0), 0)
        with self._lock:
            output = ctypes.POINTER(_Results)()
            _check(_lib.fzfp_index_find(self._handle, data, len(data), ctypes.byref(options), ctypes.byref(output)))
            try:
                r = output.contents
                self.timings = {n: getattr(r, n + "_ms") for n in ("prepare", "scan", "select", "trace", "copy", "native")}
                return [(h.index, h.score, h.start, h.end,
                         list(r.positions[h.offset:h.offset+h.count])) for h in r.hits[:r.count]]
            finally:
                _lib.fzfp_results_free(output)

    def find(self, query: str, limit: int | None = None, positions: bool = False, exact: bool = False) -> list[Hit]:
        return [Hit(h[0], h[1], _alignment(self.candidates[h[0]], h) if positions else None)
                for h in self._find(query, limit, positions, exact=exact)]

    def close(self) -> None:
        with self._lock:
            if self._handle.value:
                _check(_lib.fzfp_index_destroy(self._handle))
                self._handle.value = 0

    @property
    def candidates(self) -> tuple[str, ...]:
        return self._candidates

    def __enter__(self) -> Index:
        return self

    def __exit__(self, *exc: object) -> None:
        self.close()

    def __del__(self):
        if getattr(self, "_handle", None) and self._handle.value:
            self.close()


__all__ = ["Index", "Hit", "Options", "Match", "Result", "Scheme", "Version", "match", "filter", "version"]
