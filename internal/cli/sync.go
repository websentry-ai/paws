package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

// syncRecord is everything `paws sync` uploads about one rule firing. No command or file
// text, tool input, session id or path ever goes in here.
type syncRecord struct {
	TS            string `json:"ts"`
	Event         string `json:"event"`
	Binding       string `json:"binding"`
	Pattern       string `json:"pattern,omitempty"` // catalog bindings only
	Skill         string `json:"skill"`
	Verdict       string `json:"verdict"`
	AlreadyLoaded bool   `json:"already_loaded"`
	SatisfiedBy   string `json:"satisfied_by,omitempty"`
}

type syncBody struct {
	Records []syncRecord `json:"records"`
}

// syncCursor marks the last uploaded record: its ts, and how many rule records carry that
// ts up to and including it (ts has one-second resolution). Survives log rotation.
type syncCursor struct {
	TS string `json:"last_ts"`
	N  int    `json:"count_at_ts"`
}

type pendingRecord struct {
	rec   syncRecord
	after syncCursor // the cursor once this record is uploaded
}

const (
	maxSyncRecords = 1000
	maxSyncBytes   = 1 << 20
	dryRunShown    = 20
)

// Sync uploads rule records not sent before: `paws sync [--dry-run]`.
func Sync(args []string) int {
	dry := false
	for _, a := range args {
		if a != "--dry-run" {
			fmt.Fprintf(os.Stderr, "paws: unknown flag %q\nusage: paws sync [--dry-run]\n", a)
			return 2
		}
		dry = true
	}
	creds, err := loadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	if creds == nil && !dry {
		fmt.Fprintln(os.Stderr, "Run `paws login` first.")
		return 1
	}
	log, statePath := gate.AuditPath(), syncStatePath()
	if log == "" {
		fmt.Fprintln(os.Stderr, "paws: no home directory for the audit log; set PAWS_HOME")
		return 1
	}
	cur, err := loadSyncCursor(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	recs := pendingSync([]string{log + ".1", log}, cur)
	if dry {
		return printDryRun(recs)
	}
	return upload(creds, statePath, recs)
}

func syncStatePath() string {
	dir := gate.UserDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "sync-state.json")
}

func loadSyncCursor(path string) (syncCursor, error) {
	var c syncCursor
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("%s is unreadable: %v", path, err)
	}
	return c, nil
}

// pendingSync reads the audit logs in order (rotated file first) and returns the rule records
// after cur. The whole log is read so already_loaded sees denies from earlier syncs.
func pendingSync(paths []string, cur syncCursor) []pendingRecord {
	var out []pendingRecord
	deniedIn := map[string]bool{} // rule|session pairs that saw a deny, as in summarize
	atTS := map[string]int{}
	rules := sourceRules{}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		rd := bufio.NewReader(f)
		for {
			line, err := rd.ReadBytes('\n')
			if len(line) == 0 && err != nil {
				break
			}
			var r auditRecord
			if json.Unmarshal(line, &r) != nil || r.Rule == "" || r.TS == "" || (r.Verdict != "allow" && r.Verdict != "deny") {
				continue
			}
			key := r.Rule + "|" + r.Session
			already := r.Verdict == "allow" && r.SatisfiedBy == "skill-load" && !deniedIn[key]
			if r.Verdict == "deny" {
				deniedIn[key] = true
			}
			atTS[r.TS]++
			after := syncCursor{TS: r.TS, N: atTS[r.TS]}
			if after.TS < cur.TS || after.TS == cur.TS && after.N <= cur.N {
				continue
			}
			if rec, ok := toSyncRecord(r, already, rules); ok {
				out = append(out, pendingRecord{rec: rec, after: after})
			}
		}
		f.Close()
	}
	return out
}

// toSyncRecord keeps only the fields the portal gets. A record whose skill can't be known
// (written before the audit log carried it, rule file since removed) is dropped.
func toSyncRecord(r auditRecord, already bool, rules sourceRules) (syncRecord, bool) {
	src := rules.get(r.Source, r.Rule)
	skill := r.Skill
	if skill == "" && src != nil {
		skill = src.Action.Skill
	}
	if skill == "" {
		return syncRecord{}, false
	}
	rec := syncRecord{TS: r.TS, Event: r.Event, Binding: r.Rule, Skill: skill, Verdict: r.Verdict, AlreadyLoaded: already, SatisfiedBy: r.SatisfiedBy}
	if src != nil && fromCatalog(r.Source) {
		rec.Pattern = matcherText(src.Matcher)
	}
	return rec, true
}

// sourceRules parses each rule file named by the audit log once.
type sourceRules map[string]*rule.Rule

// get returns the rule at path if it still parses and still has this name.
func (s sourceRules) get(path, name string) *rule.Rule {
	r, seen := s[path]
	if !seen {
		if data, err := os.ReadFile(path); err == nil {
			r, _ = rule.Parse(data, path)
		}
		s[path] = r
	}
	if r == nil || r.Name != name {
		return nil
	}
	return r
}

// fromCatalog reports whether a rule file was installed by `paws add`: user packs live in
// $PAWS_HOME/packs, project packs in <repo>/.paws/packs.
func fromCatalog(source string) bool {
	sep := string(filepath.Separator)
	if user := gate.UserDir(); user != "" && strings.HasPrefix(source, filepath.Join(user, "packs")+sep) {
		return true
	}
	return strings.Contains(source, sep+filepath.Join(".paws", "packs")+sep)
}

// matcherText renders a matcher's set fields, e.g. "tool=Bash command_regex=\bgit\s+commit\b".
func matcherText(m rule.Matcher) string {
	var parts []string
	for _, kv := range [][2]string{{"tool", m.Tool}, {"command_regex", m.CommandRegex}, {"file_glob", m.FileGlob}, {"mcp_action", m.MCPAction}} {
		if kv[1] != "" {
			parts = append(parts, kv[0]+"="+kv[1])
		}
	}
	return strings.Join(parts, " ")
}

// batches splits records into requests of at most maxSyncRecords records and maxSyncBytes.
func batches(recs []pendingRecord) [][]pendingRecord {
	var out [][]pendingRecord
	var cur []pendingRecord
	size := len(`{"records":[]}`)
	for _, p := range recs {
		b, _ := json.Marshal(p.rec)
		n := len(b) + 1 // the comma
		if len(cur) == maxSyncRecords || len(cur) > 0 && size+n > maxSyncBytes {
			out = append(out, cur)
			cur, size = nil, len(`{"records":[]}`)
		}
		cur = append(cur, p)
		size += n
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func recordsOf(batch []pendingRecord) []syncRecord {
	out := make([]syncRecord, len(batch))
	for i, p := range batch {
		out[i] = p.rec
	}
	return out
}

// printDryRun shows the first request's body, cut to the first records, on stdout.
func printDryRun(recs []pendingRecord) int {
	shown := recs
	if len(shown) > dryRunShown {
		shown = shown[:dryRunShown]
	}
	data, _ := json.MarshalIndent(syncBody{Records: recordsOf(shown)}, "", "  ")
	fmt.Println(string(data))
	fmt.Fprintf(os.Stderr, "%d record(s) would be uploaded in %d request(s); showing %d. Nothing was sent.\n", len(recs), len(batches(recs)), len(shown))
	return 0
}

// upload sends each batch and advances the cursor after each one the portal took.
func upload(creds *credentials, statePath string, recs []pendingRecord) int {
	if len(recs) == 0 {
		fmt.Println("Nothing new to upload.")
		return 0
	}
	accepted, rejected := 0, 0
	for _, b := range batches(recs) {
		status, resp, err := postJSON(creds.API+"/paws/v1/sync", creds.Token, syncBody{Records: recordsOf(b)})
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			return 1
		}
		switch status {
		case http.StatusOK:
		case http.StatusUnauthorized:
			fmt.Fprintln(os.Stderr, "Your login was revoked. Run `paws login`.")
			return 1
		default:
			fmt.Fprintf(os.Stderr, "paws: %v\n", portalError(status, resp))
			return 1
		}
		var res struct{ Accepted, Rejected int }
		json.Unmarshal(resp, &res)
		accepted, rejected = accepted+res.Accepted, rejected+res.Rejected
		if err := writePrivate(statePath, b[len(b)-1].after); err != nil {
			fmt.Fprintf(os.Stderr, "paws: saving sync state: %v\n", err)
			return 1
		}
	}
	fmt.Printf("Uploaded %d record(s): %d accepted, %d rejected.\n", len(recs), accepted, rejected)
	return 0
}
