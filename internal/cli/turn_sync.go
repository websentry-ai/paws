package cli

import (
	"net/http"
	"os"
	"time"
)

// Stop-hook defaults: upload at most every 10 minutes, check the config hash every 15.
const (
	defaultSyncInterval   = 10 * time.Minute
	defaultConfigInterval = 15 * time.Minute
	stopHookTimeout       = 5 * time.Second // per portal request
)

// syncAfterTurn is the portal step of the async Stop hook. When logged in it uploads new
// records (at most every PAWS_SYNC_INTERVAL) and checks the config hash (at most every
// PAWS_CONFIG_INTERVAL, or at once when an upload reports a new hash). Best effort: errors
// are dropped and nothing is printed, since the hook's stdout carries its decision.
func syncAfterTurn(cwd string) {
	creds, err := loadCredentials()
	if err != nil || creds == nil {
		return
	}
	old := portalClient
	portalClient = &http.Client{Timeout: stopHookTimeout}
	defer func() { portalClient = old }()

	st := loadPortalState()
	now := time.Now().UTC()
	serverHash := ""
	if due(st.LastUpload, "PAWS_SYNC_INTERVAL", defaultSyncInterval, now) {
		if recs, err := pendingRecords(); err == nil && len(recs) > 0 {
			st.LastUpload = now
			savePortalState(st) // claim the slot before the network: a parallel session's Stop skips it
			if res, err := uploadRecords(creds, st.AppliedHash, cwd, recs, false); err == nil {
				serverHash = res.configHash
			}
		}
	}
	if serverHash == "" && !due(st.LastConfigCheck, "PAWS_CONFIG_INTERVAL", defaultConfigInterval, now) {
		return
	}
	st.LastConfigCheck = now
	savePortalState(st)
	res, _ := pullConfig(creds, &st, serverHash)
	if res == nil {
		return
	}
	savePortalState(st)
	if res.Hash == st.AppliedHash {
		reportApplied(creds, st.AppliedHash, cwd)
	}
}

// due reports whether the interval in env (a Go duration, else def) has passed since last.
func due(last time.Time, env string, def time.Duration, now time.Time) bool {
	d, err := time.ParseDuration(os.Getenv(env))
	if err != nil || d < 0 {
		d = def
	}
	return now.Sub(last) >= d
}
