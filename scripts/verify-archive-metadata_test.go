package main

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseArchiveMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]*tar.Header) []*tar.Header
	}{
		{"valid", nil},
		{"host user", func(h []*tar.Header) []*tar.Header { h[0].Uname = "builder"; return h }},
		{"host group", func(h []*tar.Header) []*tar.Header { h[1].Gname = "staff"; return h }},
		{"numeric owner", func(h []*tar.Header) []*tar.Header { h[2].Uid = 501; return h }},
		{"numeric group", func(h []*tar.Header) []*tar.Header { h[3].Gid = 20; return h }},
		{"build mtime", func(h []*tar.Header) []*tar.Header { h[4].ModTime = time.Unix(1, 0); return h }},
		{"access time", func(h []*tar.Header) []*tar.Header { h[0].AccessTime = time.Unix(1, 0); return h }},
		{"change time", func(h []*tar.Header) []*tar.Header { h[0].ChangeTime = time.Unix(1, 0); return h }},
		{"extra file", func(h []*tar.Header) []*tar.Header {
			extra := *h[0]
			extra.Name = "private.txt"
			return append(h, &extra)
		}},
		{"duplicate", func(h []*tar.Header) []*tar.Header { return append(h, h[0]) }},
		{"missing file", func(h []*tar.Header) []*tar.Header { return h[:4] }},
		{"symlink", func(h []*tar.Header) []*tar.Header {
			h[0].Typeflag = tar.TypeSymlink
			h[0].Linkname = "elsewhere"
			return h
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var headers []*tar.Header
			for _, name := range []string{"fzfp", "fzfp.1", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md"} {
				headers = append(headers, &tar.Header{Name: name, Mode: 0o644, Typeflag: tar.TypeReg,
					Uname: "root", Gname: "root", ModTime: time.Unix(0, 0), Format: tar.FormatPAX})
			}
			if test.mutate != nil {
				headers = test.mutate(headers)
			}
			path := filepath.Join(t.TempDir(), "release.tar.gz")
			file, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			compressed := gzip.NewWriter(file)
			writer := tar.NewWriter(compressed)
			for _, header := range headers {
				if err := writer.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
			}
			for _, closer := range []interface{ Close() error }{writer, compressed, file} {
				if err := closer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := verifyArchiveMetadata(path); (err != nil) != (test.mutate != nil) {
				t.Fatalf("verifyArchiveMetadata: %v", err)
			}
		})
	}
}
