package main

/*
#include <stdlib.h>
#include <stdint.h>
*/
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
	"unsafe"

	"github.com/junegunn/fzf/bindings/core"
)

//export fzfp_abi_version
func fzfp_abi_version() C.uint32_t { return core.ABIVersion }

//export fzfp_call
func fzfp_call(input *C.char, length C.size_t, outputLength *C.size_t, status *C.int32_t) (result *C.char) {
	if outputLength == nil || status == nil {
		return nil
	}
	*outputLength, *status = 0, 0
	emit := func(v any) *C.char {
		b, err := json.Marshal(v)
		if err != nil {
			b = []byte(`{"error":"response encoding failed"}`)
			*status = 2
		}
		*outputLength = C.size_t(len(b))
		return (*C.char)(C.CBytes(b))
	}
	defer func() {
		if p := recover(); p != nil {
			*status = 2
			result = emit(core.Response{Error: fmt.Sprintf("internal panic: %v", p)})
		}
	}()
	if length > 256*1024*1024 || (input == nil && length != 0) {
		*status = 1
		return emit(core.Response{Error: "invalid input pointer or request exceeds 256 MiB"})
	}
	b := C.GoBytes(unsafe.Pointer(input), C.int(length))
	if !utf8.Valid(b) {
		*status = 1
		return emit(core.Response{Error: "request must be UTF-8"})
	}
	var req core.Request
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	err := d.Decode(&req)
	if err == nil {
		var extra any
		if d.Decode(&extra) != io.EOF {
			err = fmt.Errorf("trailing JSON")
		}
	}
	if err != nil {
		*status = 1
		return emit(core.Response{Error: err.Error()})
	}
	if req.Operation == "version" {
		return emit(map[string]any{"version": core.Version, "core_revision": core.CoreRevision, "abi_version": core.ABIVersion, "dictionary_version": "go-pinyin v0.21.0"})
	}
	out, err := core.Execute(req)
	if err != nil {
		*status = 1
		return emit(core.Response{Error: err.Error()})
	}
	return emit(out)
}

//export fzfp_free
func fzfp_free(buffer unsafe.Pointer) { C.free(buffer) }

func main() {}
