package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

// ruleUpload is what `paws sync` uploads about one rule firing. command is the redacted
// Bash command only: a file path or MCP input never goes in it. No session id or rule path.
type ruleUpload struct {
	Kind          string `json:"kind"` // "rule"
	TS            string `json:"ts"`
	Event         string `json:"event"`
	Skill         string `json:"skill"`
	ActionLabel   string `json:"action_label,omitempty"` // portal-managed rules only
	Matcher       string `json:"matcher,omitempty"`      // catalog and portal-managed rules only
	Command       string `json:"command,omitempty"`
	Tool          string `json:"tool,omitempty"`
	Verdict       string `json:"verdict"`
	AlreadyLoaded bool   `json:"already_loaded"`
	SatisfiedBy   string `json:"satisfied_by,omitempty"`
}

// actionUpload is what `paws sync` uploads about one finished tool call (see gate.ToolAction).
type actionUpload struct {
	Kind    string        `json:"kind"` // "action"
	TS      string        `json:"ts"`
	Tool    string        `json:"tool"`
	Action  string        `json:"action"`
	Command string        `json:"command,omitempty"`
	File    string        `json:"file,omitempty"`
	MCP     *gate.MCPCall `json:"mcp,omitempty"`
}

type syncBody struct {
	Records           []any      `json:"records"` // ruleUpload and actionUpload
	AppliedConfigHash *string    `json:"applied_config_hash"`
	Inventory         *inventory `json:"inventory"`
}

type syncResponse struct {
	Accepted   int    `json:"accepted"`
	Rejected   int    `json:"rejected"`
	ConfigHash string `json:"config_hash"`
}

// syncCursor marks the last uploaded record: its ts, and how many uploadable (rule or action)
// records carry that ts up to and including it (ts has one-second resolution). Survives log rotation.
type syncCursor struct {
	TS string `json:"last_ts"`
	N  int    `json:"count_at_ts"`
}

type pendingRecord struct {
	rec   any
	after syncCursor // the cursor once this record is uploaded
}

const (
	maxSyncRecords = 1000
	maxSyncBytes   = 1 << 20
	dryRunShown    = 20
)

var errRevoked = errors.New("your login was revoked; run `paws login`")

// Sync uploads new records and pulls the portal config now: `paws sync [--dry-run]`.
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
		if loadPortalState().Revoked {
			fmt.Fprintln(os.Stderr, "Your login was revoked and its portal rules were removed. Run `paws login`.")
		} else {
			fmt.Fprintln(os.Stderr, "Run `paws login` first.")
		}
		return 1
	}
	if gate.AuditPath() == "" {
		fmt.Fprintln(os.Stderr, "paws: no home directory for the audit log; set PAWS_HOME")
		return 1
	}
	cwd, _ := os.Getwd()
	if dry {
		return printDryRun(cwd)
	}
	rep, err := syncNow(creds, cwd)
	if errors.Is(err, errRevoked) {
		fmt.Fprintln(os.Stderr, "Your login was revoked and its portal rules were removed. Run `paws login`.")
		return 1
	}
	printSyncReport(rep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	return 0
}

// syncReport is what one syncNow did, for `paws sync` and `paws login` to print.
type syncReport struct {
	Uploaded int
	Uploads  bool         // the upload step ran to the end
	Config   *applyResult // nil when the config was unchanged or never fetched
	Skills   []string     // skills the applied config turns on
}

// syncNow uploads every new record (posting once even with none, so the portal gets the
// inventory and applied hash) and then pulls the config if its hash changed. A 401 logs this
// machine out (forgetIfRevoked).
func syncNow(creds *credentials, cwd string) (syncReport, error) {
	var rep syncReport
	recs, err := pendingRecords()
	if err != nil {
		return rep, err
	}
	st := loadPortalState()
	res, err := uploadRecords(creds, st.AppliedHash, cwd, recs, true)
	rep.Uploaded = res.n
	if err != nil {
		forgetIfRevoked(err)
		return rep, err
	}
	rep.Uploads = true
	now := time.Now().UTC()
	st.LastUpload, st.LastConfigCheck = now, now
	rep.Config, err = pullConfig(creds, &st, res.configHash, cwd)
	if forgetIfRevoked(err) {
		return rep, err
	}
	rep.Skills = st.Skills
	if serr := savePortalState(st); err == nil {
		err = serr
	}
	if rep.Config != nil && rep.Config.Hash == st.AppliedHash {
		reportApplied(creds, st.AppliedHash, cwd)
	}
	return rep, err
}

func printSyncReport(rep syncReport) {
	if rep.Uploads {
		if rep.Uploaded == 0 {
			fmt.Println("Nothing new to upload.")
		} else {
			fmt.Printf("Uploaded %d records.\n", rep.Uploaded)
		}
	}
	if rep.Config == nil {
		if rep.Uploads {
			fmt.Printf("Config: up to date, %s.\n", skillList(rep.Skills))
		}
		return
	}
	fmt.Printf("Config: %s applied.\n", skillList(rep.Config.Applied))
	for _, s := range rep.Config.Skipped {
		fmt.Printf("  skipped %s\n", printable(s))
	}
}

// skillList renders "3 skills (humanizer, tdd, git)".
func skillList(names []string) string {
	switch len(names) {
	case 0:
		return "no skills"
	case 1:
		return "1 skill (" + names[0] + ")"
	}
	return fmt.Sprintf("%d skills (%s)", len(names), strings.Join(names, ", "))
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

// pendingRecords is every rule and action record in the audit logs not uploaded yet.
func pendingRecords() ([]pendingRecord, error) {
	log := gate.AuditPath()
	if log == "" {
		return nil, errors.New("no home directory for the audit log; set PAWS_HOME")
	}
	cur, err := loadSyncCursor(syncStatePath())
	if err != nil {
		return nil, err
	}
	return pendingSync([]string{log + ".1", log}, cur), nil
}

// pendingSync reads the audit logs in order (rotated file first) and returns the rule and
// action records after cur. The whole log is read so already_loaded sees earlier denies.
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
			if json.Unmarshal(line, &r) != nil || r.TS == "" {
				continue
			}
			isRule := r.Rule != "" && (r.Verdict == "allow" || r.Verdict == "deny")
			if !isRule && (r.Event != "action" || r.Action == "") {
				continue
			}
			key := r.Rule + "|" + r.Session
			already := isRule && r.Verdict == "allow" && r.SatisfiedBy == "skill-load" && !deniedIn[key]
			if isRule && r.Verdict == "deny" {
				deniedIn[key] = true
			}
			atTS[r.TS]++
			after := syncCursor{TS: r.TS, N: atTS[r.TS]}
			if after.TS < cur.TS || after.TS == cur.TS && after.N <= cur.N {
				continue
			}
			if !isRule {
				out = append(out, pendingRecord{rec: toActionUpload(r), after: after})
			} else if rec, ok := toRuleUpload(r, already, rules); ok {
				out = append(out, pendingRecord{rec: rec, after: after})
			}
		}
		f.Close()
	}
	return out
}

// toRuleUpload keeps only the fields the portal gets. A record whose skill can't be known
// (written before the audit log carried it, rule file since removed) is dropped.
func toRuleUpload(r auditRecord, already bool, rules sourceRules) (ruleUpload, bool) {
	src := rules.get(r.Source, r.Rule)
	skill := r.Skill
	if skill == "" && src != nil {
		skill = src.Action.Skill
	}
	if skill == "" {
		return ruleUpload{}, false
	}
	rec := ruleUpload{Kind: "rule", TS: r.TS, Event: r.Event, Skill: skill, Tool: r.Tool, Verdict: r.Verdict,
		AlreadyLoaded: already, SatisfiedBy: r.SatisfiedBy}
	if r.Tool == "Bash" {
		rec.Command = gate.RedactCommand(r.Target)
	}
	if src != nil && (fromCatalog(r.Source) || managed(r.Source)) {
		rec.Matcher = matcherText(src.Matcher)
	}
	if src != nil && managed(r.Source) {
		rec.ActionLabel = firstLine(src.Body)
	}
	return rec, true
}

// toActionUpload passes an action record through, re-checking what may leave the machine.
func toActionUpload(r auditRecord) actionUpload {
	rec := actionUpload{Kind: "action", TS: r.TS, Tool: r.Tool, Action: r.Action, File: r.File, MCP: r.MCP}
	if r.Command != "" {
		rec.Command = gate.RedactCommand(r.Command)
	}
	if filepath.IsAbs(rec.File) || rec.File == ".." || strings.HasPrefix(rec.File, "../") {
		rec.File = ""
	}
	return rec
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

// batches splits records into requests of at most maxSyncRecords records and maxSyncBytes,
// base being the size of the request with no records.
func batches(recs []pendingRecord, base int) [][]pendingRecord {
	var out [][]pendingRecord
	var cur []pendingRecord
	size := base
	for _, p := range recs {
		b, _ := json.Marshal(p.rec)
		n := len(b) + 1 // the comma
		if len(cur) == maxSyncRecords || len(cur) > 0 && size+n > maxSyncBytes {
			out = append(out, cur)
			cur, size = nil, base
		}
		cur = append(cur, p)
		size += n
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func recordsOf(batch []pendingRecord) []any {
	out := make([]any, len(batch))
	for i, p := range batch {
		out[i] = p.rec
	}
	return out
}

// newSyncBody is a request with no records yet: the applied hash (null before the first
// apply) and what is installed on this machine.
func newSyncBody(applied, cwd string) syncBody {
	b := syncBody{Records: []any{}, Inventory: inventoryFor(cwd)}
	if applied != "" {
		b.AppliedConfigHash = &applied
	}
	return b
}

// printDryRun shows the first request's body, cut to the first records, on stdout.
func printDryRun(cwd string) int {
	recs, err := pendingRecords()
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	body := newSyncBody(loadPortalState().AppliedHash, cwd)
	base, _ := json.Marshal(body)
	shown := recs
	if len(shown) > dryRunShown {
		shown = shown[:dryRunShown]
	}
	body.Records = recordsOf(shown)
	data, _ := json.MarshalIndent(body, "", "  ")
	fmt.Println(string(data))
	fmt.Fprintf(os.Stderr, "%d record(s) would be uploaded in %d request(s); showing %d. Nothing was sent.\n", len(recs), len(batches(recs, len(base))), len(shown))
	return 0
}

type uploadResult struct {
	n          int    // records the portal took
	configHash string // the portal's current config hash, from the last response
}

// uploadRecords sends recs in batches and advances the cursor after each one the portal took.
// With always set it posts once even when there is nothing new.
func uploadRecords(creds *credentials, applied, cwd string, recs []pendingRecord, always bool) (uploadResult, error) {
	var res uploadResult
	body := newSyncBody(applied, cwd)
	base, _ := json.Marshal(body)
	bs := batches(recs, len(base))
	if len(bs) == 0 && always {
		bs = [][]pendingRecord{nil}
	}
	for _, b := range bs {
		body.Records = recordsOf(b)
		resp, err := postSync(creds, body)
		if err != nil {
			return res, err
		}
		res.configHash = resp.ConfigHash
		if len(b) == 0 {
			continue
		}
		res.n += len(b)
		if err := writePrivate(syncStatePath(), b[len(b)-1].after); err != nil {
			return res, fmt.Errorf("saving sync state: %v", err)
		}
	}
	return res, nil
}

func postSync(creds *credentials, body syncBody) (syncResponse, error) {
	var res syncResponse
	status, resp, err := postJSON(creds.API+"/paws/v1/sync", creds.Token, body)
	if err != nil {
		return res, err
	}
	switch status {
	case http.StatusOK:
		json.Unmarshal(resp, &res)
		return res, nil
	case http.StatusUnauthorized:
		return res, errRevoked
	default:
		return res, portalError(status, resp)
	}
}

// reportApplied tells the portal which config this machine now runs, so the website can clear
// "waiting for sync" without waiting for the next upload. Best effort.
func reportApplied(creds *credentials, applied, cwd string) {
	postSync(creds, newSyncBody(applied, cwd))
}
