# unbound-paws

npm installer for [paws](https://github.com/websentry-ai/paws): skills that
actually fire. paws is a Claude Code hook that makes sure the skill you
picked is loaded before the matching action runs.

```bash
npx unbound-paws init     # registers the hook in this repo's .claude/settings.json
```

The package downloads the paws binary for your platform from the GitHub
release that matches its version, checks it against the release's
`checksums.txt`, and runs it. `paws init` copies the binary to
`~/.paws/bin/paws` and registers that path, so clearing the npm cache does not
break the hook.

Apache-2.0.
