// Package paws embeds the catalog and packs this binary ships, so `paws add` works offline
// and installs exactly the bindings that were reviewed with this release.
package paws

import "embed"

// Catalog holds catalog/ and packs/ from the repo root.
//
//go:embed catalog packs
var Catalog embed.FS

// Skill is the paws skill (skills/paws): how Claude Code writes and checks rules.
// paws init installs it beside the hook.
//
//go:embed skills/paws
var Skill embed.FS
