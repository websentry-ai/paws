// Command paws is Husky for AI agents: repos declare gates as .paws/*.md, paws gates each tool call against them.
package main

import (
	"fmt"
	"os"

	"github.com/websentry-ai/paws/internal/cli"
)

const version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "gate":
		return cli.Gate()
	case "init":
		return cli.Init(args[1:])
	case "uninstall":
		return cli.Uninstall(args[1:])
	case "explain":
		return cli.Explain(args[1:])
	case "validate":
		return cli.Validate(args[1:])
	case "version", "--version", "-v":
		fmt.Println("paws " + version)
		return 0
	case "help", "--help", "-h":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "paws: unknown command %q\n\n", args[0])
		usage()
		return 2
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `paws — Pre-tool Agent Workflow System

Usage:
  paws gate                                      Run as a Claude Code hook (reads event JSON on stdin).
  paws init [--global | --settings <path>]       Register paws in a Claude Code settings.json.
  paws uninstall [--global | --settings <path>]  Remove paws from a settings.json.
  paws explain [--tool T] [--event E] <command>  Dry-run: show matching rules and the decision.
  echo '<event json>' | paws explain -           Dry-run a full event JSON.
  paws validate [root]                           Check root/catalog bindings and root/packs manifests.
  paws version                                   Print version.

Rules are .paws/**/*.md files; paws reads every .paws/ from the working
directory up to the repo root (or $HOME if no repo), plus ~/.paws.
`)
}
