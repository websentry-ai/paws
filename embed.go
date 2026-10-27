// Package paws embeds the catalog and packs this binary ships, so `paws add` works offline
// and installs exactly the bindings that were reviewed with this release.
package paws

import "embed"

// Catalog holds catalog/ and packs/ from the repo root.
//
//go:embed catalog packs
var Catalog embed.FS
