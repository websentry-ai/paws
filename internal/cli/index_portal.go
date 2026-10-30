package cli

import (
	"io/fs"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/websentry-ai/paws/internal/catalog"
	"github.com/websentry-ai/paws/internal/pack"
	"github.com/websentry-ai/paws/internal/rule"
)

// portalCatalogSkill is one skill in the portal's /catalog (paws index --portal).
type portalCatalogSkill struct {
	Name           string          `json:"name"`
	Kind           string          `json:"kind"`
	Description    string          `json:"description"`
	License        string          `json:"license"`
	Upstream       *portalUpstream `json:"upstream"` // null for skills this repo ships
	Version        string          `json:"version"`
	ContentHash    string          `json:"content_hash"`
	DefaultActions []portalAction  `json:"default_actions"`
	Tags           []string        `json:"tags"`
}

type portalUpstream struct {
	Repo string `json:"repo"`
	Path string `json:"path"`
	Ref  string `json:"ref"`
}

// repoLicense covers the skills this repo ships under catalog/*/skills.
const repoLicense = "Apache-2.0"

// buildPortalCatalog groups bindings by the skill they require. Each binding becomes one of
// the skill's default actions; the first binding (by path) supplies the description and pin.
func buildPortalCatalog(c *catalog.Catalog) map[string]any {
	bySkill := map[string]*portalCatalogSkill{}
	tags := map[string]map[string]bool{}
	for _, rel := range sortedKeys(c.Bindings) {
		b := c.Bindings[rel]
		s := bySkill[b.Action.Skill]
		if s == nil {
			s = newCatalogSkill(c, b)
			bySkill[b.Action.Skill], tags[b.Action.Skill] = s, map[string]bool{}
		}
		s.DefaultActions = append(s.DefaultActions, portalAction{ID: b.Name, Label: bindingLabel(b), Event: b.Event,
			Tool: b.Matcher.Tool, CommandRegex: b.Matcher.CommandRegex, FileGlob: b.Matcher.FileGlob, MCPAction: b.Matcher.MCPAction})
		for _, t := range b.Tags {
			tags[b.Action.Skill][t] = true
		}
	}
	skills := []*portalCatalogSkill{}
	for _, name := range sortedKeys(bySkill) {
		s := bySkill[name]
		s.Tags = sortedKeys(tags[name])
		skills = append(skills, s)
	}
	return map[string]any{"skills": skills}
}

// newCatalogSkill fills a skill's pin: the upstream ref, or for a skill this repo ships its
// content hash (the hash `paws sync` checks before installing it from the binary).
func newCatalogSkill(c *catalog.Catalog, b *catalog.Binding) *portalCatalogSkill {
	s := &portalCatalogSkill{Name: b.Action.Skill, Kind: "skill", Description: firstLine(b.Body), DefaultActions: []portalAction{}}
	if u := b.Upstream; u.Repo != "" {
		s.Upstream = &portalUpstream{Repo: u.Repo, Path: u.Path, Ref: u.Ref}
		s.Version, s.ContentHash, s.License = u.Ref[:7], u.ContentHash, b.Provenance.License
		return s
	}
	s.License = repoLicense
	if sub, err := fs.Sub(c.FS, catalog.LocalSkillDir(b)); err == nil {
		if h, err := pack.HashSkill(sub); err == nil {
			s.ContentHash, s.Version = h, strings.TrimPrefix(h, "sha256:")[:7]
		}
	}
	return s
}

// bindingLabel is the short "runs before ..." text: the clause after "before" or "when" in the
// binding's description ("Loads humanizer before a commit message is written." gives "A commit
// message is written"), cut to 80 characters at a comma. A session-start binding, or a
// description without that clause, falls back to the matcher read as words.
func bindingLabel(b *catalog.Binding) string {
	if rule.Kind(b.Event) == rule.KindSession {
		return displayAction(b.Rule)
	}
	desc := firstLine(b.Body)
	for _, kw := range []string{" before ", " when "} {
		_, rest, ok := strings.Cut(desc, kw)
		if !ok {
			continue
		}
		rest, _, _ = strings.Cut(rest, ", so ")
		rest, _, _ = strings.Cut(rest, " (")
		rest = strings.TrimSuffix(strings.TrimSpace(rest), ".")
		if len(rest) > 80 {
			if i := strings.LastIndex(rest[:80], ", "); i > 0 {
				rest = rest[:i]
			}
		}
		if rest != "" && len(rest) <= 80 {
			r, n := utf8.DecodeRuneInString(rest)
			return string(unicode.ToUpper(r)) + rest[n:]
		}
	}
	return displayAction(b.Rule)
}
