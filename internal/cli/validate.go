package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
		fmt.Fprintln(os.Stderr, "usage: unbound-paws validate [root]")
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
	// A catalog with trigger_cases.json also checks every trigger against its fire and ignore cases.
	if _, err := os.Stat(filepath.Join(root, "trigger_cases.json")); err == nil {
		cases, exempt, err := readCases(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "paws: %v\n", err)
			return 1
		}
		if bad := catalog.CheckTriggerCases(c, cases, exempt); len(bad) > 0 {
			for _, p := range bad {
				fmt.Println(p)
			}
			fmt.Fprintf(os.Stderr, "paws: %d trigger case(s) fail\n", len(bad))
			return 1
		}
	}
	fmt.Printf("ok: %d bindings, %d packs\n", len(c.Bindings), len(c.Packs))
	return 0
}
