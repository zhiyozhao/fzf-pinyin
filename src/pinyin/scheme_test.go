package pinyin

import (
	"os"
	"os/exec"
	"testing"
)

// Upstream Init is a once-per-process API: path changes globals that default
// does not reset. Separate processes exercise the actual CLI initialization,
// rather than contaminating later tests by switching schemes in one process.
func testSchemes(t *testing.T) []string {
	t.Helper()
	if scheme := os.Getenv("FZFP_TEST_SCHEME"); scheme != "" {
		return []string{scheme}
	}
	for _, scheme := range []string{"default", "path", "history"} {
		cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
		cmd.Env = append(os.Environ(), "FZFP_TEST_SCHEME="+scheme)
		out, err := cmd.CombinedOutput()
		t.Logf("%s: %s", scheme, out)
		if err != nil {
			t.Fatal(err)
		}
	}
	return nil
}
