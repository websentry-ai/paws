package cli

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// defaultSyncInterval is how often the Stop hook uploads records (PAWS_SYNC_INTERVAL).
	defaultSyncInterval = 5 * time.Minute
	// defaultCheckInterval is how often, between uploads, the Stop hook asks the portal whether the
	// config changed (PAWS_CHECK_INTERVAL): one small GET, so a skill turned on in the browser lands
	// within a minute of a turn ending.
	defaultCheckInterval = time.Minute
	// sessionStartFloor spaces out the syncs new sessions start, so several sessions opened
	// together (or a run of /clear) make one round, not one each.
	sessionStartFloor = time.Minute
	stopHookTimeout   = 5 * time.Second // per portal request
)

// stopHookDeadline bounds a background portal round, skill downloads included. Tests shorten it.
var stopHookDeadline = 12 * time.Second

// syncAfterTurn is the portal step of the async Stop hook: an upload at most every PAWS_SYNC_INTERVAL,
// and a config check at most every PAWS_CHECK_INTERVAL in between.
func syncAfterTurn(cwd string) {
	syncRound(cwd, syncInterval())
}

// syncRound is one best-effort portal round when logged in. Once `every` has passed since the last
// upload, one POST /sync uploads new records (none is fine: it still reports the inventory) and returns
// the portal's config hash. Otherwise, once the check interval has passed since the last check, GET
// /config/hash asks for the hash alone. Either way the config itself is fetched only when that hash
// changed. Errors are dropped and nothing is printed, since a hook's stdout carries its decision. A 401
// logs this machine out (forgetIfRevoked).
func syncRound(cwd string, every time.Duration) {
	creds, err := loadCredentials()
	if err != nil || creds == nil {
		return
	}
	unlock, ok := lockSyncRound()
	if !ok {
		return // another process is in a round right now
	}
	defer unlock()
	st := loadPortalState()
	now := time.Now().UTC()
	upload := now.Sub(st.LastUpload) >= every
	if !upload && now.Sub(st.LastCheck) < checkInterval() {
		return
	}
	st.LastCheck = now
	if upload {
		st.LastUpload = now
	}
	savePortalState(st) // claim the slot before the network: a parallel session's round skips it

	ctx, cancel := context.WithTimeout(context.Background(), stopHookDeadline)
	defer cancel()
	oldClient, oldCtx := portalClient, portalCtx
	portalClient, portalCtx = &http.Client{Timeout: stopHookTimeout}, ctx
	defer func() { portalClient, portalCtx = oldClient, oldCtx }()

	known := "" // pullConfig asks GET /config/hash
	if upload {
		recs, _ := pendingRecords()
		up, err := uploadRecords(creds, st.AppliedHash, cwd, recs, true)
		if forgetIfRevoked(err) || err != nil {
			return
		}
		known = up.configHash
	}
	res, err := pullConfig(creds, &st, known, cwd)
	if forgetIfRevoked(err) || res == nil {
		return
	}
	savePortalState(st)
	if res.Hash == st.AppliedHash {
		reportApplied(creds, st.AppliedHash, cwd)
	}
}

// syncLockStale is the age past which a round's lock is taken to be left by a round that died: a live
// round ends within stopHookDeadline.
const syncLockStale = time.Minute

// lockSyncRound takes the background-round lock, $PAWS_HOME/portal/sync.lock, or reports false while
// another process holds it. Without it, rounds in parallel processes (two sessions starting together, a
// Stop hook beside a session start) could both pass the interval check before either saved its claim,
// and upload the same records twice.
func lockSyncRound() (unlock func(), ok bool) {
	dir := portalDir()
	if dir == "" || os.MkdirAll(dir, 0o700) != nil {
		return nil, false
	}
	path := filepath.Join(dir, "sync.lock")
	for try := 0; try < 2; try++ {
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err == nil {
			f.Close()
			return func() { os.Remove(path) }, true
		}
		if fi, err := os.Stat(path); err != nil || time.Since(fi.ModTime()) < syncLockStale {
			return nil, false
		}
		os.Remove(path) // stale: its round died without unlocking
	}
	return nil, false
}

// syncInterval is PAWS_SYNC_INTERVAL (a Go duration), else the default.
func syncInterval() time.Duration {
	return envDuration("PAWS_SYNC_INTERVAL", defaultSyncInterval)
}

// checkInterval is PAWS_CHECK_INTERVAL (a Go duration), else the default.
func checkInterval() time.Duration {
	return envDuration("PAWS_CHECK_INTERVAL", defaultCheckInterval)
}

func envDuration(name string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(name))
	if err != nil || d < 0 {
		return fallback
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

// syncAtSessionStart pulls portal changes when a session starts, without waiting for a turn to end.
// Tool rules are read on every tool call, so they apply in this session as soon as the round lands.
// A session-start rule's context went out before the round, so a new one applies from the next
// session; waiting here would put the network in front of every session start. It spawns only when
// logged in and the last round is older than sessionStartFloor, so the gate pays two small file reads.
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
