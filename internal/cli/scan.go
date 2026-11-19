package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

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

// scanNow is replaceable in tests: the clock the window ends at when no login bounds it.
var scanNow = time.Now

// Scan is `paws scan [--days 30|90] [--dry-run]`.
func Scan(args []string) int {
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
	st := loadPortalState()
	st.HistoryScannedAt, st.HistoryDays = scanNow().UTC(), days
	if err := savePortalState(st); err != nil {
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
	// Once logged in with paws registered for every project, the hooks upload each call as it happens, so the scan
	// stops where they started and no call counts twice. Without the global hook, calls since login in projects paws
	// isn't registered in would reach the portal from nowhere, so the scan keeps them.
	if creds != nil && hasPawsHook(filepath.Join(userClaudeDir(), "settings.json")) {
		if at, err := time.Parse(time.RFC3339, creds.CreatedAt); err == nil && at.Before(until) {
			until = at
		}
	}
	since := until.AddDate(0, 0, -days)
	src := hook.Transcripts{Dir: filepath.Join(userClaudeDir(), "projects")}
	fmt.Fprintf(os.Stderr, "Reading the Claude Code sessions in %s from %s to %s...\n", src.Dir, since.Format(time.DateOnly), until.Format(time.DateOnly))
	sum, err := history.Summarize(src, since, until, days, catalog)
	if err != nil {
		return nil, sum, fmt.Errorf("reading %s: %w", src.Dir, err)
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
