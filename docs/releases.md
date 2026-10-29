# Release paws

## Prepare a release

1. Set a newer version in `npm/package.json` and merge it to `main`.
2. Run **CI and release** from the Actions page with branch `main`, or run:

   ```bash
   gh workflow run ci.yml --ref main
   ```

The workflow rejects versions that are equal to or older than any npm
version, or older than a GitHub release tag. Tests run before any tag,
GitHub release, or npm publication is created. Pushes and pull requests run
checks only.

GoReleaser builds macOS, Linux, and Windows binaries for x64 and arm64. It
uploads the archives and `checksums.txt` to `v<VERSION>`. The npm wrapper
uses that same version and downloads those assets. Stable versions publish
to `latest`. Prereleases publish to `next`.

Make the repository public before a release. The workflow checks visibility
because npm users need anonymous access to GitHub release downloads.

## npm token

The workflow publishes with an npm token, as unbound-cli does. A repository
admin adds it once:

1. On npmjs.com, as the account that owns `unbound-paws`, create a granular
   access token with read and write access to packages, and allow it to
   bypass two-factor authentication so CI can publish.
2. Save it as the `NPM_TOKEN` repository secret:

   ```bash
   gh secret set NPM_TOKEN -R websentry-ai/paws
   ```

The release job stops before tagging if the secret is missing, and runs
`npm whoami` with it first. To release binaries without npm, run the
workflow with **Publish to npm after releasing the binaries** unchecked:

```bash
gh workflow run ci.yml --ref main -f publish_npm=false
```

## Verify the install

CI builds all six archives and tests npm installation on Linux without an
agent login. Run the same smoke check locally with GoReleaser, Go, Node,
and Python 3.11 or newer:

```bash
python3 .github/release/smoke.py
```

The check verifies archive contents and checksums, installs the package with
a fresh npm cache, runs `init`, and confirms the hook binary survives cache
removal. It also rejects corrupted checksums and verifies a retry works
after the checksums are restored. Nothing is published.

With a working Claude Code login and available usage, run the live journey:

```bash
python3 .claude/skills/verify-paws/scripts/drive.py run --npm launch-journey
```

This builds all release archives, serves them locally, installs the packed
npm wrapper with a fresh npm cache, and drives a Claude Code session through
the writing pack's commit gate. It does not publish anything.

After publishing, use a new device with Node 18 or newer and Claude Code:

```bash
npx --yes unbound-paws@latest version
npx --yes unbound-paws@latest init
```

For a prerelease, use `@next`. Ask Claude Code to install the writing pack
and make a commit. Confirm that the humanizer skill loads before the commit.

## Retry a failed npm publication

If binaries were released but npm publication failed, fix npm authentication
and rerun the same workflow run. It reuses the existing GitHub release.
An existing tag must point at the tested commit. After a version has been
published to npm, bump `npm/package.json` before another release.
