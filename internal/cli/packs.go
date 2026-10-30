package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/websentry-ai/paws"
	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/pack"
	"github.com/websentry-ai/paws/internal/rule"
)

// packFlags are the flags shared by the pack commands.
type packFlags struct {
	project bool
	portal  bool // index: the portal's /catalog shape
	catalog string
	picks   map[string]string
	out     string
	args    []string
}

func parsePackFlags(args []string, allowed ...string) (packFlags, error) {
	f := packFlags{picks: map[string]string{}}
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			f.args = append(f.args, a)
			continue
		}
		if flag, val, ok := strings.Cut(a, "="); ok && strings.HasPrefix(a, "--") {
			// --pick=g=o and --catalog=dir: split once and reuse the two-arg path below.
			args = append(args[:i+1], append([]string{val}, args[i+1:]...)...)
			a = flag
		}
		if !ok[a] {
			return f, fmt.Errorf("unknown flag %q", a)
		}
		switch a {
		case "--project":
			f.project = true
			continue
		case "--portal":
			f.portal = true
			continue
		}
		if i+1 >= len(args) {
			return f, fmt.Errorf("%s needs a value", a)
		}
		i++
		switch a {
		case "--catalog":
			f.catalog = args[i]
		case "-o":
			f.out = args[i]
		case "--pick":
			g, o, found := strings.Cut(args[i], "=")
			if !found {
				return f, fmt.Errorf("--pick wants group=option, got %q", args[i])
			}
			f.picks[g] = o
		}
	}
	return f, nil
}

// loadCatalog reads the embedded catalog, or a directory given with --catalog.
func loadCatalog(dir string) (*catalog.Catalog, error) {
	var c *catalog.Catalog
	var probs []catalog.Problem
	// Full validation either way: names become paths and refs reach git, so a catalog that
	// fails any check installs nothing.
	if dir == "" {
		c, probs = catalog.ValidateFS(paws.Catalog, "embedded")
	} else {
		c, probs = catalog.Validate(dir)
	}
	if len(probs) > 0 {
		for _, p := range probs {
			fmt.Fprintln(os.Stderr, "paws: catalog:", p)
		}
		return nil, fmt.Errorf("catalog has %d problem(s)", len(probs))
	}
	return c, nil
}

func target(project bool) pack.Target {
	if project {
		cwd, _ := os.Getwd()
		return pack.ProjectTarget(projectRoot(cwd))
	}
	return pack.UserTarget(gate.UserDir())
}

// Add installs packs or single bindings: `paws add [--project] [--pick group=option] NAME...`.
func Add(args []string) int {
	f, err := parsePackFlags(args, "--project", "--catalog", "--pick")
	if err == nil && len(f.args) == 0 {
		err = fmt.Errorf("name a pack or binding to add")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\nusage: paws add [--project] [--pick group=option] <pack|category/binding>...\n", err)
		return 2
	}
	c, err := loadCatalog(f.catalog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	t := target(f.project)
	for _, name := range f.args {
		packName, bindings, picks, err := pack.Resolve(c, name, f.picks, askPick)
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			return 1
		}
		res, err := pack.Add(c, packName, bindings, picks, t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: add %s: %v\n", name, err)
			return 1
		}
		fmt.Printf("Added %s (%d binding(s)) to %s\n", res.Pack, len(res.Bindings), pack.InstallDir(t, res.Pack))
		for _, s := range res.Installed {
			fmt.Printf("  skill %s installed in %s\n", s, t.SkillsDir)
		}
		for _, s := range res.Kept {
			fmt.Printf("  skill %s already in %s (not installed by paws); using it as is\n", s, t.SkillsDir)
		}
	}
	if !hookRegistered() {
		fmt.Println("paws is not registered as a hook here yet: run `paws init` (this repo) or `paws init --global`.")
	}
	return 0
}

// askPick prompts for a pick_one option when stdin is a terminal; otherwise it takes the default.
func askPick(g catalog.PickOne) string {
	if st, err := os.Stdin.Stat(); err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return ""
	}
	var names []string
	for n := range g.Options {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Printf("%s", g.Name)
	if g.Description != "" {
		fmt.Printf(" (%s)", g.Description)
	}
	fmt.Printf(": pick one of %s [%s]: ", strings.Join(names, ", "), g.Default)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}

// hookRegistered reports whether this repo's or the global settings run paws.
func hookRegistered() bool {
	cwd, _ := os.Getwd()
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	if claude == "" {
		home, _ := os.UserHomeDir()
		claude = filepath.Join(home, ".claude")
	}
	for _, p := range []string{
		filepath.Join(projectRoot(cwd), ".claude", "settings.json"),
		filepath.Join(projectRoot(cwd), ".claude", "settings.local.json"),
		filepath.Join(claude, "settings.json"),
	} {
		data, err := os.ReadFile(p)
		if err == nil && strings.Contains(string(data), strings.TrimSpace(pawsMarker)) {
			return true
		}
	}
	return false
}

// Remove uninstalls a pack: `paws remove [--project] NAME`.
func Remove(args []string) int {
	f, err := parsePackFlags(args, "--project")
	if err == nil && len(f.args) != 1 {
		err = fmt.Errorf("name one pack to remove")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\nusage: paws remove [--project] <pack>\n", err)
		return 2
	}
	skills, err := pack.Remove(f.args[0], target(f.project))
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	fmt.Printf("Removed %s\n", f.args[0])
	for _, s := range skills {
		fmt.Printf("  skill %s removed\n", s)
	}
	return 0
}

// List shows installed packs with local fire counts: `paws list [--project]`.
func List(args []string) int {
	f, err := parsePackFlags(args, "--project")
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 2
	}
	fires := fireCounts(filepath.Join(gate.UserDir(), "audit.log"))
	scopes := []bool{false, true}
	if f.project {
		scopes = []bool{true}
	}
	any := false
	for _, project := range scopes {
		t := target(project)
		st, err := pack.LoadState(t.StateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			return 1
		}
		names := make([]string, 0, len(st.Packs))
		for n := range st.Packs {
			names = append(names, n)
		}
		sort.Strings(names)
		scope := "user"
		if project {
			scope = "project"
		}
		for _, n := range names {
			any = true
			inst := st.Packs[n]
			fmt.Printf("%s  (%s, %s)\n", n, scope, inst.At)
			for _, rel := range inst.Bindings {
				name := ruleName(filepath.Join(pack.InstallDir(t, n), filepath.FromSlash(rel)))
				fmt.Printf("  %-28s fired %d\n", name, fires[name])
			}
		}
	}
	if !any {
		fmt.Println("no packs installed (browse with `paws search`)")
	}
	return 0
}

func ruleName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return filepath.Base(path) + " (missing)"
	}
	r, err := rule.Parse(data, path)
	if err != nil {
		return filepath.Base(path) + " (invalid)"
	}
	return r.Name
}

// fireCounts counts audit records per rule name.
func fireCounts(path string) map[string]int {
	counts := map[string]int{}
	f, err := os.Open(path)
	if err != nil {
		return counts
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var rec struct {
			Rule string `json:"rule"`
		}
		if json.Unmarshal(sc.Bytes(), &rec) == nil && rec.Rule != "" {
			counts[rec.Rule]++
		}
	}
	return counts
}

// Search finds packs and bindings by name, tag, skill or description: `paws search TEXT`.
func Search(args []string) int {
	f, err := parsePackFlags(args, "--catalog")
	if err == nil && len(f.args) == 0 {
		err = fmt.Errorf("give a search term")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\nusage: paws search <text>\n", err)
		return 2
	}
	c, err := loadCatalog(f.catalog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	q := strings.ToLower(strings.Join(f.args, " "))
	hit := func(fields ...string) bool {
		return strings.Contains(strings.ToLower(strings.Join(fields, " ")), q)
	}
	found := 0
	for _, n := range sortedKeys(c.Packs) {
		p := c.Packs[n]
		if hit(p.Name, p.Description, strings.Join(p.Tags, " ")) {
			found++
			fmt.Printf("pack %-20s %s\n     paws add %s\n", p.Name, p.Description, p.Name)
		}
	}
	for _, rel := range sortedKeys(c.Bindings) {
		b := c.Bindings[rel]
		if hit(b.Name, rel, b.Action.Skill, strings.Join(b.Tags, " "), b.Body) {
			found++
			fmt.Printf("binding %-28s %s\n     paws add %s\n", b.Name, firstLine(b.Body), strings.TrimSuffix(rel, ".md"))
		}
	}
	if found == 0 {
		fmt.Printf("nothing matches %q\n", q)
	}
	return 0
}

// Index writes the catalog as JSON for the website and the local UI: `paws index [-o file]`.
// With --portal it writes the portal's /catalog shape: one entry per skill.
func Index(args []string) int {
	f, err := parsePackFlags(args, "--catalog", "-o", "--portal")
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 2
	}
	c, err := loadCatalog(f.catalog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	var index any = buildIndex(c)
	if f.portal {
		index = buildPortalCatalog(c)
	}
	b, _ := json.MarshalIndent(index, "", "  ")
	b = append(b, '\n')
	if f.out == "" {
		os.Stdout.Write(b)
		return 0
	}
	if err := os.WriteFile(f.out, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "paws: %v\n", err)
		return 1
	}
	return 0
}

type indexBinding struct {
	Path        string           `json:"path"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Event       string           `json:"event"`
	Action      string           `json:"action"`
	Skill       string           `json:"skill,omitempty"`
	Tags        []string         `json:"tags"`
	Upstream    *rule.Upstream   `json:"upstream,omitempty"`
	Provenance  *rule.Provenance `json:"provenance,omitempty"`
	Add         string           `json:"add"`
}

type indexPack struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Bindings    []string          `json:"bindings"`
	PickOne     []catalog.PickOne `json:"pick_one,omitempty"`
	Add         string            `json:"add"`
}

func buildIndex(c *catalog.Catalog) map[string]any {
	var packs []indexPack
	for _, n := range sortedKeys(c.Packs) {
		p := c.Packs[n]
		packs = append(packs, indexPack{p.Name, p.Description, p.Tags, p.Bindings, p.PickOne, "paws add " + p.Name})
	}
	var bindings []indexBinding
	for _, rel := range sortedKeys(c.Bindings) {
		b := c.Bindings[rel]
		ib := indexBinding{Path: rel, Name: b.Name, Description: firstLine(b.Body), Event: b.Event, Action: b.Action.Type,
			Skill: b.Action.Skill, Tags: b.Tags, Add: "paws add " + strings.TrimSuffix(rel, ".md")}
		if b.Upstream.Repo != "" {
			u, pv := b.Upstream, b.Provenance
			ib.Upstream, ib.Provenance = &u, &pv
		}
		bindings = append(bindings, ib)
	}
	return map[string]any{"packs": packs, "bindings": bindings}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n\n")
	return strings.Join(strings.Fields(line), " ")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var contentHashLine = regexp.MustCompile(`^(\s+)content_hash:.*$`)
var refLine = regexp.MustCompile(`^(\s+)ref:.*$`)

// setContentHash writes hash into the upstream: block of a binding's frontmatter, replacing an
// existing content_hash or adding one after ref. Nothing outside that block is touched.
func setContentHash(data, hash string) (string, bool) {
	lines := strings.Split(data, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return data, false
	}
	start, end := -1, len(lines)
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t\r") == "---" {
			if end == len(lines) {
				end = i
			}
			break
		}
		if strings.TrimRight(lines[i], " \t\r") == "upstream:" {
			start = i
		}
	}
	if start < 0 {
		return data, false
	}
	blockEnd := start + 1 // first line after the indented block
	for blockEnd < end && (strings.HasPrefix(lines[blockEnd], " ") || strings.HasPrefix(lines[blockEnd], "\t")) {
		blockEnd++
	}
	refAt := -1
	for i := start + 1; i < blockEnd; i++ {
		if m := contentHashLine.FindStringSubmatch(lines[i]); m != nil {
			lines[i] = m[1] + "content_hash: " + hash
			return strings.Join(lines, "\n"), true
		}
		if refLine.MatchString(lines[i]) {
			refAt = i
		}
	}
	if refAt < 0 {
		return data, false
	}
	indent := refLine.FindStringSubmatch(lines[refAt])[1]
	lines = append(lines[:refAt+1], append([]string{indent + "content_hash: " + hash}, lines[refAt+1:]...)...)
	return strings.Join(lines, "\n"), true
}

// Pin fetches every upstream skill a catalog directory references and writes its content
// hash into the binding: `paws pin [root]`. For catalog maintainers.
func Pin(args []string) int {
	root := "."
	if len(args) == 1 {
		root = args[0]
	} else if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "usage: paws pin [root]")
		return 2
	}
	c, probs := catalog.Load(root)
	for _, p := range probs {
		fmt.Fprintln(os.Stderr, "paws:", p)
	}
	hashes := map[string]string{} // repo@ref:path -> hash
	failed := 0
	for _, rel := range sortedKeys(c.Bindings) {
		b := c.Bindings[rel]
		u := b.Upstream
		if u.Repo == "" {
			continue
		}
		key := u.Repo + "@" + u.Ref + ":" + u.Path
		hash, ok := hashes[key]
		if !ok {
			_, tmp, h, err := pack.FetchUpstream(context.Background(), u.Repo, u.Ref, u.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "paws: %s: %v\n", rel, err)
				failed++
				continue
			}
			os.RemoveAll(tmp)
			hash, hashes[key] = h, h
		}
		path := filepath.Join(catalog.CatalogDir(root), filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			failed++
			continue
		}
		out, ok := setContentHash(string(data), hash)
		if !ok {
			fmt.Fprintf(os.Stderr, "paws: %s: no upstream.ref line to pin beside\n", rel)
			failed++
			continue
		}
		if out != string(data) {
			if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "paws: %v\n", err)
				failed++
				continue
			}
			fmt.Printf("pinned %s %s\n", rel, hash)
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}
