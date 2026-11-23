//go:build unix

package integration

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestInteractivePrefixCache(t *testing.T) {
	bin := binary(t, "FZFP_BIN", "fzfp")
	dir := t.TempDir()
	input, output := filepath.Join(dir, "input"), filepath.Join(dir, "selected")
	if err := os.WriteFile(input, []byte("中国\n中x国\n北京\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `"$1" --sync --no-mouse --no-clear --no-color --no-unicode --print-query --prompt='query> ' < "$2" > "$3"`, "sh", bin, input, output)
	cmd.Env = append(cleanEnv(), "TERM=xterm-256color")
	terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	defer func() {
		cancel()
		if cmd.ProcessState == nil {
			_ = cmd.Wait()
		}
	}()
	chunks := make(chan []byte, 64)
	go func() {
		defer close(chunks)
		buf := make([]byte, 8192)
		for {
			n, err := terminal.Read(buf)
			if n > 0 {
				select {
				case chunks <- bytes.Clone(buf[:n]):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	var screen strings.Builder
	waitFor := func(fragment string) {
		t.Helper()
		for !strings.Contains(screen.String(), fragment) {
			select {
			case chunk, ok := <-chunks:
				if !ok {
					t.Fatalf("terminal closed waiting for %q: %q", fragment, screen.String())
				}
				screen.Write(chunk)
			case <-ctx.Done():
				t.Fatalf("terminal timed out waiting for %q: %q", fragment, screen.String())
			}
		}
	}
	waitFor("query>")
	query := ""
	for _, letter := range "zhonggu" {
		query += string(letter)
		if _, err := io.WriteString(terminal, string(letter)); err != nil {
			t.Fatal(err)
		}
		waitFor(query)
	}
	if _, err := io.WriteString(terminal, "\r"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	selected, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(selected) != "zhonggu\n中国\n" {
		t.Fatalf("interactive selection: %q", selected)
	}
}
