package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/harness"
	"github.com/websentry-ai/paws/internal/history"
	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/rule"
)

// `paws scan` reads the tool calls of past Claude Code sessions on this machine, matches them against the catalog
// here, and sends the portal counts only, so it can pick skills on day one. See internal/history for what is kept.

const scanUsage = "usage: unbound-paws scan [--days 30|90] [--dry-run]"

// scanDays are the windows a scan may cover. Claude Code deletes sessions older than 30 days by default
// (cleanupPeriodDays), so 90 finds more only where that setting was raised.
var scanDays = map[int]bool{30: true, 90: true}

const defaultScanDays = 30

// historyUpload is the body of POST /history, and what --dry-run prints.
type historyUpload struct {
	history.Summary
	CatalogVersion string `json:"catalog_version"`
}

// catalogTriggers is GET /catalog/triggers: every catalog skill's actions.
type catalogTriggers struct {
	Version string `json:"version"`
	Skills  []struct {
		Name    string         `json:"name"`
		Actions []portalAction `json:"actions"`
	} `json:"skills"`
}

// skipSynced leaves out the calls sync has uploaded already: each call the hook logged, matched one to one against
// the audit log's action records of the same session. A session the hook joined partway keeps its earlier calls; one
// it never saw keeps all of them.
type skipSynced struct {
	history.Source
	from   syncCursor               // sync's cursor when this account first logged in here
	logged map[string][]*loggedCall // by session, in log order; read once the transcripts have been
	before time.Time                // tool calls before this may have lost their records to rotation; zero if none
}

// loggedCall is an action record: what the hook wrote when a call finished.
type loggedCall struct {
	tool, command, file, action, hash string
	at                                time.Time
	used                              bool
}

// maxCallTime bounds how long after a call starts the hook may log it. A record must also carry the call's own
// command, file or action, so a long window can't pair two different calls.
const maxCallTime = hook.MaxCallTime

// Calls reads every call first: a resumed or forked session copies calls into another log under the same id, and
// a call uploaded through one copy must not count through another. A call whose any copy the hook logged is left out
// with all its copies.
func (s skipSynced) Calls(since, until time.Time, fn func(history.Call)) (int, error) {
	var calls []history.Call
	if _, err := s.Source.Calls(since, until, func(c history.Call) { calls = append(calls, c) }); err != nil {
		return 0, err
	}
	// The audit log is read after the transcripts: a call the transcripts show as finished had its hook record
	// written before its result, so the record is already there, even for a call that finished during the scan.
	logged, kept, err := loggedCalls(s.from)
	if err != nil {
		// Without the log the scan can't tell which calls sync uploaded; guessing would count them twice.
		return 0, fmt.Errorf("reading the audit log: %w", err)
	}
	// The audit log keeps two files. Once it has rotated, records older than the kept ones may be gone though sync
	// uploaded their calls, so tool calls from before the oldest kept record are left out. Skill loads still count:
	// sync never sends them.
	if from, _ := time.Parse(time.RFC3339, s.from.TS); !kept.IsZero() && kept.Before(from) {
		kept = time.Time{} // older records went to an earlier login, so their loss can't cause a double count
	}
	s.logged, s.before = logged, kept
	synced, counted := map[string]bool{}, map[string]bool{}
	for _, c := range calls {
		if c.ID != "" && !synced[c.ID] && c.Tool != history.SkillTool && s.take(c) { // one record per call, not per copy
			synced[c.ID] = true
		}
	}
	for _, c := range calls {
		// A skill load counts toward `loaded`, which sync doesn't send, so it always goes through.
		if c.Tool != history.SkillTool && (c.At.Before(s.before) || c.ID != "" && synced[c.ID] || c.ID == "" && s.take(c)) {
			continue
		}
		counted[c.Session] = true
		fn(c)
	}
	return len(counted), nil // sessions with a call left in
}

// skillReads is a history.Source whose calls that read a whole SKILL.md are the skill loads they are: they count
// toward `loaded`, like the Skill tool. A read of part of the file stays an ordinary call: alone it loads nothing.
type skillReads struct{ history.Source }

func (s skillReads) Calls(since, until time.Time, fn func(history.Call)) (int, error) {
	return s.Source.Calls(since, until, func(c history.Call) {
		// From the session's folder. A Read takes no offset or limit here (history keeps only the path): its range is
		// the default, the file's first 2000 lines.
		if r, ok := gate.ReadOf(c.Tool, c.Target, nil, c.Cwd, false); ok && r.Lines.From == 1 && (r.Lines.To == 0 || c.Tool == "Read") {
			c.Tool, c.Target = history.SkillTool, r.Skill
		}
		fn(c)
	})
}

// take marks and reports the earliest unused record of c's session that is c: same tool, the same command (Bash,
// as the hook redacts it), file (as the hook keeps it, relative to the repo) or action key, written while c could
// have been running. The log has whole seconds, so a record may read up to a second before the call.
func (s skipSynced) take(c history.Call) bool {
	want := gate.DescribeAction(gate.Event{Tool: c.Tool, Target: c.Target})
	hash := gate.TargetHash(c.Target)
	for _, r := range s.logged[c.Session] {
		// The hook logs a read of a SKILL.md as a Skill action: its input hash names the call whatever the tool.
		sameTool := r.tool == c.Tool || r.tool == history.SkillTool && r.hash != "" && r.hash == hash
		if r.used || !sameTool || r.at.Before(c.At.Add(-time.Second)) || !r.at.Before(c.At.Add(maxCallTime)) {
			continue
		}
		switch {
		case r.hash != "" && r.hash != hash: // the record names another input
			continue
		case r.hash != "": // the same input: this call
		// A record from before target_hash: compare the closest fields it has.
		case c.Tool == "Bash" && r.command != want.Command:
			continue
		case fileTool(c.Tool) && (r.file == "" || c.Target != r.file && !strings.HasSuffix(c.Target, "/"+r.file)):
			continue // a record without a path (a file outside the repo) can't be told apart from another; it matches none
		case !fileTool(c.Tool) && c.Tool != "Bash" && r.action != want.Action:
			continue
		}
		r.used = true
		return true
	}
	return false
}

func fileTool(tool string) bool {
	return tool == "Read" || tool == "Write" || tool == "Edit" || tool == "NotebookEdit"
}

// loggedCalls are the audit log's action records after the cursor `from` (the ones this login's syncs uploaded, counted
// exactly as pendingSync counts them), by session, and,
// when the log has rotated (audit.log.1 exists), the time of the oldest record it kept: anything the hook logged
// before that may be gone. A log file that doesn't exist is empty; one that exists but can't be read is an error.
func loggedCalls(from syncCursor) (map[string][]*loggedCall, time.Time, error) {
	out := map[string][]*loggedCall{}
	var kept time.Time
	log := gate.AuditPath()
	if log == "" {
		return out, kept, nil
	}
	atTS := map[string]int{} // uploadable records per second, as pendingSync counts them
	for i, p := range []string{log + ".1", log} {
		f, err := os.Open(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, kept, err
		}
		rotated := i == 0
		rd := bufio.NewReader(f) // no line limit: a long record must not end the read early
		for {
			line, err := rd.ReadBytes('\n')
			if err != nil && !errors.Is(err, io.EOF) {
				f.Close()
				return nil, kept, fmt.Errorf("%s: %w", p, err)
			}
			if len(line) == 0 && err != nil {
				break
			}
			if rotated && kept.IsZero() {
				var first struct {
					TS string `json:"ts"`
				}
				if json.Unmarshal(line, &first) == nil {
					kept, _ = time.Parse(time.RFC3339, first.TS)
				}
			}
			var r struct {
				TS      string `json:"ts"`
				Event   string `json:"event"`
				Rule    string `json:"rule"`
				Outcome string `json:"outcome"`
				Skill   string `json:"skill"`
				Verdict string `json:"verdict"`
				Session string `json:"session"`
				Tool    string `json:"tool"`
				Command string `json:"command"`
				File    string `json:"file"`
				Action  string `json:"action"`
				Hash    string `json:"target_hash"`
			}
			if json.Unmarshal(line, &r) != nil || r.TS == "" {
				continue
			}
			isRule := r.Rule != "" && (r.Verdict == "allow" || r.Verdict == "deny")
			isOutcome := r.Event == "outcome" && r.Outcome != "" && r.Skill != ""
			if !isRule && !isOutcome && (r.Event != "action" || r.Action == "") {
				continue // pendingSync neither counts nor sends it
			}
			atTS[r.TS]++
			if r.TS < from.TS || r.TS == from.TS && atTS[r.TS] <= from.N || isRule || isOutcome || r.Session == "" {
				continue // sent before this login, or not a call
			}
			if ts, err := time.Parse(time.RFC3339, r.TS); err == nil {
				out[r.Session] = append(out[r.Session], &loggedCall{tool: r.Tool, command: r.Command, file: r.File, action: r.Action, hash: r.Hash, at: ts})
			}
		}
		f.Close()
	}
	return out, kept, nil
}

// scanStateFile is in the portal folder beside state.json, not in it: the hooks' syncs rewrite state.json in the
// background, and a scan saving its own fields there could overwrite theirs.
const scanStateFile = "scan.json"

// scanState is the last scan the portal got and the window it covered. Running a scan is the consent; it replaces
// the one before.
type scanState struct {
	ScannedAt time.Time `json:"scanned_at"`
	Days      int       `json:"window_days"`
}

func saveScanState(st scanState) error {
	dir := portalDir()
	if dir == "" {
		return errors.New("no home directory for the portal state; set PAWS_HOME")
	}
	data, _ := json.MarshalIndent(st, "", "  ")
	return writeAtomic(filepath.Join(dir, scanStateFile), append(data, '\n'), 0o600)
}

func loadScanState() scanState {
	var st scanState
	if dir := portalDir(); dir != "" {
		data, _ := os.ReadFile(filepath.Join(dir, scanStateFile))
		json.Unmarshal(data, &st)
	}
	return st
}

// scanNow is replaceable in tests: the clock the window ends at when no login bounds it.
var scanNow = time.Now

// Scan is `paws scan [--days 30|90] [--dry-run]`.
func Scan(args []string) int {
	if name := agentArg(args); name != "" {
		// Only Claude Code's session logs are read; Codex's rollout files aren't a stable format.
		if a := harness.Find(adapters, name); a == nil || a != defaultAdapter() {
			fmt.Fprintf(os.Stderr, "paws: scan reads %s sessions only; it can't read %s sessions yet\n", defaultAdapter().Title(), name)
			return 2
		}
		args = withoutAgent(args)
	}
	days, dry, ok := scanFlags(args)
	if !ok {
		fmt.Fprintln(os.Stderr, scanUsage)
		return 2
	}
	creds, err := loadCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	if creds == nil && !dry {
		fmt.Fprintf(os.Stderr, "Run `%s login --scan` first, or `%s scan --dry-run` to see what a scan would send.\n", cliName, cliName)
		return 1
	}
	return runScan(creds, days, dry)
}

// withoutAgent is args without --agent and its value.
func withoutAgent(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--agent" && i+1 < len(args) {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

// scanFlags reads --days N and --dry-run.
func scanFlags(args []string) (days int, dry, ok bool) {
	days = defaultScanDays
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--dry-run":
			dry = true
		case a == "--days" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || !scanDays[n] {
				return 0, false, false
			}
			days = n
		default:
			return 0, false, false
		}
	}
	return days, dry, true
}

// runScan scans, then prints (dry) or uploads the summary. creds is nil only for a dry run.
func runScan(creds *credentials, days int, dry bool) int {
	body, sum, err := buildScan(creds, days)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	if dry {
		os.Stdout.Write(body) // exactly the bytes an upload sends
		return 0
	}
	if err := postHistory(creds, body); err != nil {
		if forgetIfRevoked(err) {
			fmt.Fprint(os.Stderr, loggedOutNotice(errors.Is(err, errAccountDeleted)))
		} else {
			fmt.Fprintf(os.Stderr, "paws: sending the scan: %v\n", err)
		}
		return 1
	}
	if err := saveScanState(scanState{ScannedAt: scanNow().UTC(), Days: days}); err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
	}
	printScan(sum)
	return 0
}

// buildScan reads this machine's sessions against the catalog's triggers; body is the upload, byte for byte.
func buildScan(creds *credentials, days int) ([]byte, history.Summary, error) {
	api := ""
	if creds != nil {
		api = creds.API
	} else {
		var err error
		if api, err = catalogAPI(); err != nil {
			return nil, history.Summary{}, err
		}
	}
	var cat catalogTriggers
	if err := fetchCatalog(portalCtx, api+"/paws/v1/catalog/triggers", maxPortalResponse, &cat); err != nil {
		return nil, history.Summary{}, err
	}
	catalog, err := scanCatalog(cat)
	if err != nil {
		return nil, history.Summary{}, err
	}
	until := scanNow().UTC()
	since := until.AddDate(0, 0, -days)
	a := defaultAdapter()
	src, dir := a.History()
	src = skillReads{src}
	// Once logged in, sync uploads every call the hook logged, from before the login too (the first sync sends the
	// whole audit log). Each of those reached the portal already, so it is left out; a call the hook never logged (paws
	// not registered there, or not yet) still counts.
	if creds != nil {
		// This login's syncs uploaded only the records after creds.SyncFrom (sync's cursor outlives a logout, so an
		// earlier account may have received the older ones).
		src = skipSynced{Source: src, from: syncCursor{TS: creds.SyncFrom, N: creds.SyncFromN}}
	}
	fmt.Fprintf(os.Stderr, "Reading the %s sessions in %s from %s to %s...\n", a.Title(), dir, since.Format(time.DateOnly), until.Format(time.DateOnly))
	sum, err := history.Summarize(src, since, until, days, catalog)
	if err != nil {
		return nil, sum, fmt.Errorf("reading %s: %w", dir, err)
	}
	body, err := json.MarshalIndent(historyUpload{Summary: sum, CatalogVersion: cat.Version}, "", "  ")
	return append(body, '\n'), sum, err
}

// Bounds on what a catalog may ask a scan to count. Each action's count is information about the laptop's past
// commands, so a catalog of thousands of narrow patterns could rebuild text from them; the live catalog is far
// below these (169 skills, 217 actions, at most 3 per skill), and a scan refuses a catalog over them.
const (
	maxScanSkills          = 500
	maxScanActions         = 1000
	maxScanActionsPerSkill = 20
)

// scanCatalog turns the catalog's actions into matchers, built as the portal config builds a skill's rule files (so a
// scan matches what the gate would), with the catalog's skill names and the MCP tool patterns its triggers name.
func scanCatalog(cat catalogTriggers) (history.Catalog, error) {
	out := history.Catalog{Skills: map[string]bool{}}
	actions := 0
	for _, s := range cat.Skills {
		actions += len(s.Actions)
		if len(s.Actions) > maxScanActionsPerSkill {
			return out, fmt.Errorf("the paws catalog sent %d triggers for %s; a scan counts at most %d per skill", len(s.Actions), printable(s.Name), maxScanActionsPerSkill)
		}
	}
	if len(cat.Skills) > maxScanSkills || actions > maxScanActions {
		return out, fmt.Errorf("the paws catalog sent %d skills with %d triggers; a scan counts at most %d skills and %d triggers", len(cat.Skills), actions, maxScanSkills, maxScanActions)
	}
	for _, s := range cat.Skills {
		out.Skills[s.Name] = true
		for i, a := range s.Actions {
			out.MCPPatterns = append(out.MCPPatterns, history.MCPPatterns(a.Tool)...)
			name := ruleFileName(s.Name, a.ID, i)
			data, err := managedRuleFile(s.Name, name, a, "", "")
			if err != nil {
				continue
			}
			r, err := rule.Parse(data, name+".md")
			if err != nil || rule.Kind(r.Event) != rule.KindTool {
				continue // a session-start trigger has no tool call to count
			}
			out.Triggers = append(out.Triggers, history.Trigger{Skill: s.Name, Action: actionID(a.ID), Matches: r.Matches})
		}
	}
	return out, nil
}

// actionID is a catalog action's id as text: the backend sends a number or a string.
func actionID(id any) string {
	if v, ok := id.(float64); ok {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	s, _ := id.(string)
	return s
}

func postHistory(creds *credentials, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, creds.API+"/paws/v1/history", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	status, resp, err := portalDo(req, creds.Token)
	if err != nil {
		return err
	}
	if out := loggedOutError(status, resp); out != nil {
		return out
	}
	if status != http.StatusOK {
		return portalError(status, resp)
	}
	return nil
}

// maxScanShown is how many matched skills the scan lists; the portal ranks them all.
const maxScanShown = 8

func printScan(sum history.Summary) {
	fmt.Printf("Read %d sessions from %s to %s: %d tool calls.\n", sum.Sessions, sum.Since, sum.Until, sum.ToolCalls)
	if len(sum.Skills) == 0 {
		fmt.Println("No catalog skill matched them.")
	} else {
		fmt.Println("Catalog skills for what Claude did:")
		for i, s := range sum.Skills {
			if i == maxScanShown {
				fmt.Printf("  and %d more\n", len(sum.Skills)-i)
				break
			}
			fmt.Printf("  %-32s %d calls\n", s.Skill, s.Calls)
		}
	}
	fmt.Printf("Sent the counts, not the calls. Pick skills at %s/portal\n", defaultSiteURL)
}
