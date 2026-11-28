// Verify release headers without depending on GNU/BSD tar listing formats.
package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"time"
)

func verifyArchiveMetadata(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	entries := map[string]bool{
		"fzfp": false, "fzfp.1": false, "LICENSE": false,
		"NOTICE": false, "THIRD_PARTY_NOTICES.md": false,
	}
	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		seen, allowed := entries[header.Name]
		if !allowed || seen || header.Typeflag != tar.TypeReg {
			return fmt.Errorf("unexpected, duplicate or nonregular entry: %q", header.Name)
		}
		if header.Uname != "root" || header.Gname != "root" || header.Uid != 0 || header.Gid != 0 {
			return fmt.Errorf("%s: owner/group must be root/root with UID/GID 0", header.Name)
		}
		if !header.ModTime.Equal(time.Unix(0, 0)) || !header.AccessTime.IsZero() || !header.ChangeTime.IsZero() {
			return fmt.Errorf("%s: mtime must be Unix epoch, with no access/change times", header.Name)
		}
		entries[header.Name] = true
	}
	for name, seen := range entries {
		if !seen {
			return fmt.Errorf("missing entry: %s", name)
		}
	}
	// Read through the gzip trailer so a corrupt checksum cannot go unnoticed.
	_, err = io.Copy(io.Discard, compressed)
	return err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: verify-archive-metadata ARCHIVE...")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		if err := verifyArchiveMetadata(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("Archive metadata passed: %s\n", path)
	}
}
