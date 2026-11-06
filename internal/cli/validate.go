package cli

import (
	"fmt"
	"os"

	"github.com/websentry-ai/paws/internal/catalog"
)

// Validate checks every binding under <root>/catalog and every manifest under <root>/packs.
func Validate(args []string) int {
	root := "."
	switch len(args) {
	case 0:
	case 1:
		root = args[0]
	default:
		fmt.Fprintln(os.Stderr, "usage: paws validate [root]")
		return 2
	}
	if st, err := os.Stat(catalog.CatalogDir(root)); err != nil || !st.IsDir() {
		fmt.Fprintf(os.Stderr, "paws: no catalog/ directory under %s\n", root)
		return 2
	}
	c, probs := catalog.Validate(root)
	for _, p := range probs {
		fmt.Println(p)
	}
	if len(probs) > 0 {
		fmt.Fprintf(os.Stderr, "paws: %d problem(s)\n", len(probs))
		return 1
	}
	fmt.Printf("ok: %d bindings, %d packs\n", len(c.Bindings), len(c.Packs))
	return 0
}
