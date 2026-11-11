package cli

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const (
	// defaultSyncInterval is how often the Stop hook talks to the portal (PAWS_SYNC_INTERVAL).
	defaultSyncInterval = 5 * time.Minute
	// sessionStartFloor spaces out the syncs new sessions start, so several sessions opened
	// together (or a run of /clear) make one round, not one each.
	sessionStartFloor = time.Minute
	stopHookTimeout   = 5 * time.Second // per portal request
)

// stopHookDeadline bounds a background portal round, skill downloads included. Tests shorten it.
var stopHookDeadline = 12 * time.Second

// syncAfterTurn is the portal step of the async Stop hook: a round at most every PAWS_SYNC_INTERVAL.
func syncAfterTurn(cwd string) {
	syncRound(cwd, syncInterval())
}

// syncRound is one best-effort portal round when logged in and `every` has passed since the last:
// one POST /sync uploads new records (none is fine: it still reports the inventory) and returns the
// portal's config hash, and the config itself is fetched only when that hash changed. Errors are
// dropped and nothing is printed, since a hook's stdout carries its decision. A 401 logs this machine
// out (forgetIfRevoked).
func syncRound(cwd string, every time.Duration) {
	creds, err := loadCredentials()
	if err != nil || creds == nil {
		return
	}
	st := loadPortalState()
	now := time.Now().UTC()
	if now.Sub(st.LastUpload) < every {
		return
	}
	st.LastUpload = now
	savePortalState(st) // claim the slot before the network: a parallel session's round skips it

	ctx, cancel := context.WithTimeout(context.Background(), stopHookDeadline)
	defer cancel()
	oldClient, oldCtx := portalClient, portalCtx
	portalClient, portalCtx = &http.Client{Timeout: stopHookTimeout}, ctx
	defer func() { portalClient, portalCtx = oldClient, oldCtx }()

	recs, _ := pendingRecords()
	up, err := uploadRecords(creds, st.AppliedHash, cwd, recs, true)
	if forgetIfRevoked(err) || err != nil {
		return
	}
	res, err := pullConfig(creds, &st, up.configHash, cwd)
	if forgetIfRevoked(err) || res == nil {
		return
	}
	savePortalState(st)
	if res.Hash == st.AppliedHash {
		reportApplied(creds, st.AppliedHash, cwd)
	}
}

// syncInterval is PAWS_SYNC_INTERVAL (a Go duration), else the default.
func syncInterval() time.Duration {
	d, err := time.ParseDuration(os.Getenv("PAWS_SYNC_INTERVAL"))
	if err != nil || d < 0 {
		return defaultSyncInterval
	}
	return d
}

// spawnSessionSync starts `paws sync --background` and does not wait for it. The SessionStart hook
// is synchronous and the gate stays offline, so the network work runs in that separate process.
// Tests replace it.
var spawnSessionSync = func(cwd string) {
	self, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(self, "sync", "--background") // nil stdio is the null device: the hook's pipes close when the gate exits
	cmd.Dir = cwd
	if cmd.Start() == nil {
		cmd.Process.Release()
	}
}

// syncAtSessionStart pulls portal changes when a session starts, so a skill turned on in the
// browser reaches the next session without waiting for a turn to end. It spawns only when logged
// in and the last round is older than sessionStartFloor, so the gate pays two small file reads.
func syncAtSessionStart(cwd, source string) {
	if source == "compact" { // the same session carrying on, not a new one
		return
	}
	if creds, err := loadCredentials(); err != nil || creds == nil {
		return
	}
	if time.Since(loadPortalState().LastUpload) < sessionStartFloor {
		return
	}
	spawnSessionSync(cwd)
}
