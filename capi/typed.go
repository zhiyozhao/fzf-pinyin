package main

/*
#define FZFP_CGO
#include "fzfp.h"
#include <stdlib.h>
*/
import "C"

import (
	"sync"
	"time"
	"unsafe"

	"github.com/junegunn/fzf/bindings/core"
)

var indexes = struct {
	sync.Mutex
	next   uint64
	values map[uint64]*core.Index
}{values: make(map[uint64]*core.Index)}

func recoverStatus(status *C.int32_t) {
	if recover() != nil {
		*status = 2
	}
}

//export fzfp_typed_abi_version
func fzfp_typed_abi_version() C.uint32_t { return 2 }

//export fzfp_index_create
func fzfp_index_create(bytes *C.char, byteLength C.uint64_t, offsets *C.uint64_t, count C.uint64_t, options *C.fzfp_options, handle *C.uint64_t) (status C.int32_t) {
	defer recoverStatus(&status)
	if handle == nil {
		return 1
	}
	*handle = 0
	if byteLength > 256*1024*1024 || count > 10_000_000 || (bytes == nil && byteLength != 0) || offsets == nil || options == nil {
		return 1
	}
	o := *options
	if o.abi_version != 2 || o.struct_size != C.sizeof_fzfp_options || o.scheme > 2 || o.case_sensitive > 1 || o.normalize > 1 || o.workers > 64 || o.reserved0 != 0 || o.reserved1 != 0 {
		return 1
	}
	os := unsafe.Slice(offsets, int(count)+1)
	if os[0] != 0 || os[len(os)-1] != byteLength {
		return 1
	}
	for i := 1; i < len(os); i++ {
		if os[i] < os[i-1] || os[i] > byteLength {
			return 1
		}
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(bytes)), int(byteLength))
	texts := make([]string, int(count))
	for i := range texts {
		texts[i] = string(b[int(os[i]):int(os[i+1])])
	}
	norm := o.normalize != 0
	x, err := core.NewIndex(texts, core.Options{Scheme: []string{"default", "path", "history"}[o.scheme], CaseSensitive: o.case_sensitive != 0, Normalize: &norm}, int(o.workers))
	if err != nil {
		return 1
	}
	indexes.Lock()
	defer indexes.Unlock()
	if indexes.next == ^uint64(0) {
		return 2
	} // Never reuse a stale integer handle.
	indexes.next++
	indexes.values[indexes.next] = x
	*handle = C.uint64_t(indexes.next)
	return 0
}

//export fzfp_index_destroy
func fzfp_index_destroy(handle C.uint64_t) (status C.int32_t) {
	defer recoverStatus(&status)
	indexes.Lock()
	defer indexes.Unlock()
	if _, ok := indexes.values[uint64(handle)]; !ok {
		return 3
	}
	delete(indexes.values, uint64(handle))
	return 0
}

//export fzfp_index_find
func fzfp_index_find(handle C.uint64_t, query *C.char, length C.uint64_t, options *C.fzfp_find_options, output **C.fzfp_results) (status C.int32_t) {
	callStart := time.Now()
	defer recoverStatus(&status)
	if output == nil {
		return 1
	}
	*output = nil
	if length > 256*1024*1024 || (query == nil && length != 0) || options == nil {
		return 1
	}
	o := *options
	if o.abi_version != 2 || o.struct_size != C.sizeof_fzfp_find_options || o.limit < -1 || o.flags & ^C.uint32_t(7) != 0 || o.reserved != 0 {
		return 1
	}
	indexes.Lock()
	x := indexes.values[uint64(handle)]
	indexes.Unlock()
	if x == nil {
		return 3
	}
	q := string(unsafe.Slice((*byte)(unsafe.Pointer(query)), int(length)))
	hits, times, err := x.FindMode(q, int(o.limit), o.flags&1 != 0, o.flags&2 != 0, o.flags&4 != 0)
	if err != nil {
		return 1
	}
	start := time.Now()
	r := (*C.fzfp_results)(C.calloc(1, C.sizeof_fzfp_results))
	if r == nil {
		return 4
	}
	complete := false
	defer func() {
		if !complete {
			fzfp_results_free(r)
		}
	}()
	r.abi_version = 2
	r.struct_size = C.sizeof_fzfp_results
	r.count = C.uint64_t(len(hits))
	pc := 0
	for _, h := range hits {
		pc += len(h.Positions)
	}
	r.position_count = C.uint64_t(pc)
	if len(hits) > 0 {
		r.hits = (*C.fzfp_hit)(C.malloc(C.size_t(len(hits)) * C.sizeof_fzfp_hit))
		if r.hits == nil {
			return 4
		}
	}
	if pc > 0 {
		r.positions = (*C.uint64_t)(C.malloc(C.size_t(pc) * 8))
		if r.positions == nil {
			return 4
		}
	}
	rows := unsafe.Slice(r.hits, len(hits))
	pp := unsafe.Slice(r.positions, pc)
	offset := 0
	for i, h := range hits {
		rows[i] = C.fzfp_hit{index: C.uint64_t(h.Index), score: C.int64_t(h.Score), start: C.int64_t(h.Start), end: C.int64_t(h.End), position_offset: C.uint64_t(offset), position_count: C.uint64_t(len(h.Positions))}
		for _, p := range h.Positions {
			pp[offset] = C.uint64_t(p)
			offset++
		}
	}
	r.prepare_ms = C.double(times.Prepare)
	r.scan_ms = C.double(times.Scan)
	r.select_ms = C.double(times.Select)
	r.trace_ms = C.double(times.Trace)
	r.copy_ms = C.double(float64(time.Since(start)) / float64(time.Millisecond))
	r.native_ms = C.double(float64(time.Since(callStart)) / float64(time.Millisecond))
	*output = r
	complete = true
	return 0
}

//export fzfp_results_free
func fzfp_results_free(r *C.fzfp_results) {
	if r != nil {
		C.free(unsafe.Pointer(r.hits))
		C.free(unsafe.Pointer(r.positions))
		C.free(unsafe.Pointer(r))
	}
}
