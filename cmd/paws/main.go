// Command paws is Husky for AI agents: repos declare gates as .paws/*.md, paws gates each tool call against them.
package main

import (
	"fmt"
	"os"

	"github.com/websentry-ai/paws/internal/cli"
)

// version is stamped by GoReleaser (-X main.version) from the release tag.
var version = "0.2.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cli.Version = version
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
	case "add":
		return cli.Add(args[1:])
	case "remove":
		return cli.Remove(args[1:])
	case "list":
		return cli.List(args[1:])
	case "search":
		return cli.Search(args[1:])
	case "index":
		return cli.Index(args[1:])
	case "pin":
		return cli.Pin(args[1:])
	case "ui":
		return cli.UI(args[1:])
	case "login":
		return cli.Login(args[1:])
	case "logout":
		return cli.Logout(args[1:])
	case "sync":
		return cli.Sync(args[1:])
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
  paws add [--project] [--pick g=o] <pack|cat/binding>...
                                                 Install packs: bindings plus the skills they require.
  paws remove [--project] <pack>                 Uninstall a pack (and skills only it needed).
  paws list [--project]                          Installed packs with local fire counts.
  paws ui [--port N] [--no-open]                 Local page: catalog, installed packs, what fired.
  paws search <text>                             Search the catalog.
  paws index [--portal] [-o file]                The catalog as JSON (website, local UI); --portal: the portal's /catalog.
  paws validate [root]                           Check root/catalog bindings and root/packs manifests.
  paws pin [root]                                Fetch upstream skills and pin their content hashes.
  paws login [--key <one-time-key>|-]            Link this machine to the paws portal (opt-in); --key for no browser (- reads stdin).
  paws logout                                    Forget the portal login and remove its rules.
  paws sync [--dry-run]                          Upload new records and pull portal settings now; --dry-run prints the upload.
  paws version                                   Print version.

Rules are .paws/**/*.md files; paws reads every .paws/ from the working
directory up to the repo root (or $HOME if no repo), plus ~/.paws.

Nothing leaves the machine unless you run paws login. Once logged in, the
async Stop hook uploads records and pulls portal settings after Claude's
turns (throttled); the gate itself never uses the network. The portal
address is $PAWS_PORTAL_API (default https://api.paws.getunbound.ai).
`)
}
