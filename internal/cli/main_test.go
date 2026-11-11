package cli

import (
	"os"
	"testing"
)

// TestMain keeps SessionStart from spawning the test binary as `sync --background`; tests that
// check the spawn replace spawnSessionSync themselves.
func TestMain(m *testing.M) {
	spawnSessionSync = func(string) {}
	os.Exit(m.Run())
}
