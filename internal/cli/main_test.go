package cli

import (
	"os"
	"testing"
)

// TestMain keeps SessionStart from spawning the test binary as `sync --background`; tests that
// check the spawn replace spawnSessionSync themselves.
func TestMain(m *testing.M) {
	spawnSessionSync = func(string) {}
	interactive = func() bool { return false } // a test run from a terminal must not stop to ask
	os.Exit(m.Run())
}
