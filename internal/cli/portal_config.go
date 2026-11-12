package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/pack"
	"github.com/websentry-ai/paws/internal/rule"
)

// The portal config reaches the gate as plain rule files under $PAWS_HOME/portal/, which
// paws already loads with the rest of ~/.paws. Only `paws sync`, `paws login`, the async
// Stop hook and the background sync a new session starts fetch it.

// portalState is $PAWS_HOME/portal/state.json.
type portalState struct {
	AppliedHash string    `json:"applied_config_hash,omitempty"`
	Skills      []string  `json:"skills,omitempty"`  // skills the applied config turns on
	LastUpload  time.Time `json:"last_upload"`       // the last portal round: upload and config check together
	Revoked     bool      `json:"revoked,omitempty"` // the portal answered 401; login was forgotten
}

// portalConfig is GET /config: the account's skills that are on.
type portalConfig struct {
	ConfigHash string        `json:"config_hash"`
	Skills     []portalSkill `json:"skills"`
}

type portalSkill struct {
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	Load        string         `json:"load"` // full | rule
	RuleText    string         `json:"rule_text"`
	Version     string         `json:"version"`
	ContentHash string         `json:"content_hash"`
	Upstream    *rule.Upstream `json:"upstream"`
	Actions     []portalAction `json:"actions"`
	Status      string         `json:"status"`
}

// portalAction is one action a skill runs before, as /config and /catalog carry it.
type portalAction struct {
	ID           any    `json:"id"`
	Label        string `json:"label"`
	Event        string `json:"event"`
	Tool         string `json:"tool"`
	CommandRegex string `json:"command_regex,omitempty"`
	FileGlob     string `json:"file_glob,omitempty"`
	MCPAction    string `json:"mcp_action,omitempty"`
	// Example is a call the trigger fires on, from the catalog's reviewed cases; the website shows it.
	Example string `json:"example,omitempty"`
}

// applyResult is what applying one config did.
type applyResult struct {
	Hash    string
	Applied []string // skills whose rule files are in place
	Skipped []string // "<skill>: <reason>"
	Pending bool     // a skill waits for its SKILL.md; keep checking instead of recording the hash
}

func portalDir() string {
	dir := gate.UserDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "portal")
}

// managed reports whether a rule file is one paws wrote from the portal config. The gate
// names rule files by their resolved path, so a symlinked PAWS_HOME is compared resolved too.
func managed(source string) bool {
	dir := portalDir()
	if dir == "" {
		return false
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(source, dir+sep) {
		return true
	}
	resolved, err := filepath.EvalSymlinks(dir)
	return err == nil && strings.HasPrefix(source, resolved+sep)
}

// loadPortalState reads state.json; missing or unreadable is a fresh state (a refetch fixes it).
func loadPortalState() portalState {
	var st portalState
	if dir := portalDir(); dir != "" {
		data, _ := os.ReadFile(filepath.Join(dir, "state.json"))
		json.Unmarshal(data, &st)
	}
	return st
}

func savePortalState(st portalState) error {
	dir := portalDir()
	if dir == "" {
		return errors.New("no home directory for the portal state; set PAWS_HOME")
	}
	data, _ := json.MarshalIndent(st, "", "  ")
	return writeAtomic(filepath.Join(dir, "state.json"), append(data, '\n'), 0o600)
}

// clearPortal removes every rule file and digest the portal config wrote, and state.json.
func clearPortal() error {
	dir := portalDir()
	if dir == "" {
		return nil
	}
	if err := writeManaged(dir, nil); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, "state.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// forgetIfRevoked logs this machine out when err is a 401: the token is dead, so the portal's
// rules, state and credentials go, as with `paws logout`, and state.json remembers why. It
// reports whether it did.
func forgetIfRevoked(err error) bool {
	if !errors.Is(err, errRevoked) {
		return false
	}
	clearPortal()
	if path := credentialsPath(); path != "" {
		os.Remove(path)
	}
	savePortalState(portalState{Revoked: true})
	return true
}

// writeAtomic writes through a temp file in the same directory and a rename, so a reader (a
// gate running in parallel) sees the old file or the new one, never half of one.
func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*") // no .md suffix: never loaded as a rule
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, perm)
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

// pullConfig fetches and applies the config when its hash differs from the applied one. known
// is a hash the caller already has (from a /sync response); "" asks GET /config/hash. It
// returns nil when nothing changed. The applied hash is recorded only when every skill could be
// fetched, so a network failure is retried at the next check.
func pullConfig(creds *credentials, st *portalState, known, cwd string) (*applyResult, error) {
	hash := known
	if hash == "" {
		var h struct {
			ConfigHash string `json:"config_hash"`
		}
		if err := portalGet(creds, "/config/hash", &h); err != nil {
			return nil, err
		}
		hash = h.ConfigHash
	}
	if hash == "" || hash == st.AppliedHash && !portalDrifted(st, cwd) {
		return nil, nil
	}
	var cfg portalConfig
	if err := portalGet(creds, "/config", &cfg); err != nil {
		return nil, err
	}
	useSkillFiles(creds.API) // the backend that sent the config holds its skills' files
	res, err := applyConfig(cfg, cwd)
	st.Skills = res.Applied
	if err == nil && !res.Pending {
		st.AppliedHash = cfg.ConfigHash
	}
	return &res, err
}

// portalDrifted reports whether this laptop lost part of the applied config since it was
// applied, say by a user deleting a skill or a rule file: a skill with no rule files left, or a
// full skill (no digest) whose SKILL.md is gone. The config is then re-applied though its hash
// hasn't changed.
func portalDrifted(st *portalState, cwd string) bool {
	dir := portalDir()
	if dir == "" {
		return false
	}
	skillsDir := pack.UserTarget(gate.UserDir()).SkillsDir
	projectSkills := pack.ProjectTarget(projectRoot(cwd)).SkillsDir
	for _, name := range st.Skills {
		if rules, _ := filepath.Glob(filepath.Join(dir, name+"--*.md")); len(rules) == 0 {
			return true
		}
		if _, err := os.Stat(filepath.Join(dir, "digests", name+".md")); err == nil {
			continue // load=rule: nothing to install
		}
		if !hasSkill(name, skillsDir, projectSkills) {
			return true
		}
	}
	return false
}

func portalGet(creds *credentials, path string, v any) error {
	status, body, err := getJSON(creds.API+"/paws/v1"+path, creds.Token)
	switch {
	case err != nil:
		return err
	case status == http.StatusUnauthorized:
		return errRevoked
	case status != http.StatusOK:
		return portalError(status, body)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("portal: malformed %s response", path)
	}
	return nil
}

// applyConfig makes $PAWS_HOME/portal/ hold exactly the config's rule files: one per skill
// and action, plus a digest per load=rule skill. Skills install first, and a full skill's rules
// are written only once its SKILL.md is in place (user skills, or cwd's project), so a rule is
// never live before the skill it requires can load.
func applyConfig(cfg portalConfig, cwd string) (applyResult, error) {
	res := applyResult{Hash: cfg.ConfigHash}
	dir := portalDir()
	if dir == "" {
		return res, errors.New("no home directory for portal rules; set PAWS_HOME")
	}
	skillsDir := pack.UserTarget(gate.UserDir()).SkillsDir
	projectSkills := pack.ProjectTarget(projectRoot(cwd)).SkillsDir
	want := map[string][]byte{} // path under dir -> content
	var errs []error
	for _, s := range cfg.Skills {
		if s.Status == "off" {
			continue
		}
		files, skip, err := skillFiles(s, skillsDir, dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("skill %s: %w", s.Name, err))
		}
		if len(files) > 0 && s.Load != "rule" && !hasSkill(s.Name, skillsDir, projectSkills) {
			files, skip = nil, "skill not installed"
			res.Pending = true
		}
		if skip != "" {
			res.Skipped = append(res.Skipped, s.Name+": "+skip)
		}
		if len(files) == 0 {
			continue
		}
		for p, data := range files {
			want[p] = data
		}
		res.Applied = append(res.Applied, s.Name)
	}
	sort.Strings(res.Applied)
	if err := writeManaged(dir, want); err != nil {
		errs = append(errs, err)
	}
	return res, errors.Join(errs...)
}

// skillFiles installs a skill when it comes from the catalog and returns its rule files.
// skip says why some or all of it was left out; err is a failure worth retrying.
func skillFiles(s portalSkill, skillsDir, dir string) (map[string][]byte, string, error) {
	if !catalog.ValidSkillName(s.Name) {
		return nil, "invalid skill name", nil
	}
	files := map[string][]byte{}
	digest := ""
	var installErr error
	if s.Load == "rule" {
		text := strings.TrimSpace(s.RuleText) + "\n"
		if text == "\n" {
			return nil, "load=rule with no rule text", nil
		}
		if len(text) > rule.MaxDigest {
			return nil, fmt.Sprintf("rule text is %d bytes, over the %d-byte limit", len(text), rule.MaxDigest), nil
		}
		digest = "digests/" + s.Name + ".md"
		files[digest] = []byte(text)
	} else if skip, err := prepareSkill(s, skillsDir); skip != "" {
		return nil, skip, err
	} else {
		installErr = err
	}
	rules, skips := actionFiles(s, digest, dir)
	if len(rules) == 0 {
		return nil, strings.Join(append(skips, "no usable actions"), "; "), installErr
	}
	for p, data := range rules {
		files[p] = data
	}
	return files, strings.Join(skips, "; "), installErr
}

// hasSkill reports whether a skill's SKILL.md is in one of the skills directories.
func hasSkill(name string, skillsDirs ...string) bool {
	for _, d := range skillsDirs {
		if st, err := os.Stat(filepath.Join(d, name, "SKILL.md")); err == nil && st.Mode().IsRegular() {
			return true
		}
	}
	return false
}

// prepareSkill installs a full skill before its rules go live. A non-empty skip means its
// rules must not be written; an error with no skip means an older copy serves meanwhile.
func prepareSkill(s portalSkill, skillsDir string) (string, error) {
	outcome, err := installPortalSkill(s, skillsDir)
	_, serr := os.Stat(filepath.Join(skillsDir, s.Name))
	switch {
	case outcome == pack.UpstreamMismatch && serr == nil:
		// An update that didn't check out: the installed version keeps serving, and the error
		// keeps this config from being reported as applied.
		return "", errors.New("content hash mismatch at the pinned ref; not installed; keeping the installed version")
	case outcome == pack.UpstreamMismatch:
		return "content hash mismatch at the pinned ref; not installed", nil
	case outcome == badPin:
		return "its pin is not a github.com repo at a commit with a content hash; not installed", nil
	case err == nil:
		return "", nil
	case serr != nil:
		return "not installed: " + err.Error(), err
	}
	return "", err
}

// actionFiles renders each of a skill's actions as a rule file; skips name the ones that don't parse.
func actionFiles(s portalSkill, digest, dir string) (map[string][]byte, []string) {
	files := map[string][]byte{}
	var skips []string
	for i, a := range s.Actions {
		name := ruleFileName(s.Name, a.ID, i)
		data, err := managedRuleFile(s.Name, name, a, digest, dir)
		if err != nil {
			skips = append(skips, fmt.Sprintf("action %q: %v", a.Label, err))
			continue
		}
		files[name+".md"] = data
	}
	return files, skips
}

// badPin: the config names an upstream skill without a github.com repo, a commit ref and a content hash.
const badPin = "bad-pin"

// installPortalSkill installs a catalog skill from the paws backend: the files the config's content hash names,
// checked against it. An upstream skill must name a github.com repo at a commit, where the backend got the files; a
// custom skill (no hash) is the user's to provide, "" with no error.
func installPortalSkill(s portalSkill, skillsDir string) (string, error) {
	want, source := s.ContentHash, "catalog:"+s.Name
	if u := s.Upstream; u != nil && u.Repo != "" {
		if want == "" {
			want = u.ContentHash
		}
		if want == "" || !strings.HasPrefix(u.Repo, "github.com/") || !catalog.RepoRe.MatchString(u.Repo) || !catalog.RefRe.MatchString(u.Ref) {
			return badPin, nil
		}
		source = u.Repo + "@" + u.Ref
	} else if want == "" {
		return "", nil
	}
	return pack.InstallSkillByHash(portalCtx, skillsDir, s.Name, source, want)
}

// ruleFileName is "<skill>--<action id>", the id cut to file-name characters; the action's
// position stands in for a missing id. An id that had to change gets a short hash of the
// original, so two ids that cut to the same text still get different files.
func ruleFileName(skill string, id any, i int) string {
	raw := ""
	switch v := id.(type) {
	case float64: // a JSON number
		raw = strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		raw = v
	}
	s := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, raw)
	if len(s) > 64 {
		s = s[:64]
	}
	if s != raw {
		sum := sha256.Sum256([]byte(raw))
		s += "-" + hex.EncodeToString(sum[:4])
	}
	if s == "" {
		s = fmt.Sprint(i + 1)
	}
	return skill + "--" + s
}

// managedRule is the frontmatter of a portal rule file.
type managedRule struct {
	Name    string `yaml:"name"`
	Event   string `yaml:"event"`
	Matcher struct {
		Tool         string `yaml:"tool,omitempty"`
		CommandRegex string `yaml:"command_regex,omitempty"`
		FileGlob     string `yaml:"file_glob,omitempty"`
		MCPAction    string `yaml:"mcp_action,omitempty"`
	} `yaml:"matcher,omitempty"`
	Action struct {
		Type   string `yaml:"type"`
		Skill  string `yaml:"skill"`
		OnMiss string `yaml:"on_miss,omitempty"`
		Digest string `yaml:"digest,omitempty"`
	} `yaml:"action"`
}

// managedRuleFile renders one action as a require-skill rule, the label as its body, and checks
// it parses as the gate will parse it.
func managedRuleFile(skill, name string, a portalAction, digest, dir string) ([]byte, error) {
	var m managedRule
	m.Name, m.Event = name, a.Event
	if m.Event == "" {
		m.Event = "pre-tool"
	}
	m.Matcher.Tool, m.Matcher.CommandRegex, m.Matcher.FileGlob, m.Matcher.MCPAction = a.Tool, a.CommandRegex, a.FileGlob, a.MCPAction
	m.Action.Type, m.Action.Skill = rule.ActionRequireSkill, skill
	if digest != "" {
		m.Action.OnMiss, m.Action.Digest = rule.OnMissDigest, digest
	}
	fm, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}
	label := strings.Join(strings.Fields(a.Label), " ")
	data := []byte("---\n# Written by paws from the portal config; the next sync replaces edits.\n" + string(fm) + "---\n" + label + "\n")
	r, err := rule.Parse(data, filepath.Join(dir, name+".md"))
	if err != nil {
		return nil, err
	}
	if rule.Kind(r.Event) == rule.KindTool && r.Matches("Skill", "Skill") {
		return nil, errors.New("it would gate the Skill tool, the call that loads the skill (no tool or matcher set?); not applied")
	}
	return data, nil
}

// writeManaged writes want into dir (digests before the rules that point at them) and then
// removes rule and digest files the config no longer has, rules before their digests.
func writeManaged(dir string, want map[string][]byte) error {
	paths := sortedKeys(want)
	sort.SliceStable(paths, func(i, j int) bool {
		return strings.HasPrefix(paths[i], "digests/") && !strings.HasPrefix(paths[j], "digests/")
	})
	for _, p := range paths {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if old, err := os.ReadFile(full); err == nil && bytes.Equal(old, want[p]) {
			continue
		}
		if err := writeAtomic(full, want[p], 0o644); err != nil {
			return err
		}
	}
	for _, pattern := range []string{"*.md", "digests/*.md"} {
		stale, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, f := range stale {
			rel, _ := filepath.Rel(dir, f)
			if _, keep := want[filepath.ToSlash(rel)]; !keep {
				if err := os.Remove(f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
