// Package paws embeds what this binary ships besides its code: the paws skill and the local UI. The catalog is not
// one of them; it lives on the paws backend, which `paws add` and portal sync download from.
package paws

import "embed"

// Skill is the paws skill (skills/paws): how Claude Code writes and checks rules.
// paws init installs it beside the hook.
//
//go:embed skills/paws
var Skill embed.FS

// Web is the local UI that `paws ui` serves: the React app in web/, built into web/dist
// and committed so `go build` needs no Node.
//
//go:embed all:web/dist
var Web embed.FS
