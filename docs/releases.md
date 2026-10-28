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

## Publish the first npm version

The package must exist on npm before you can configure a trusted publisher.
Use your local npm login for the first publication, with npm's interactive
two-factor authentication. This avoids adding a publishing token to GitHub.

1. Merge the release workflow and make the repository public.
2. Run **CI and release** on `main` with **Publish to npm after releasing the
   binaries** unchecked, or run:

   ```bash
   gh workflow run ci.yml --ref main -f publish_npm=false
   ```

3. Wait for the run to succeed. Check out the exact commit used by that run.
4. From `npm/`, run `npm publish --access public`. For a prerelease, add
   `--tag next`. Complete npm's 2FA prompt.
5. Configure trusted publishing before the next release.

The first `npm publish` creates `unbound-paws` under the logged-in npm
account. A separate package-creation step on npmjs.com is not required.

## Link npm to GitHub Actions

Open the [unbound-paws package page](https://www.npmjs.com/package/unbound-paws)
after the first publication and select **Settings**. Under **Trusted Publisher**, select
**GitHub Actions** and enter:

| Field | Value |
| --- | --- |
| Organization or user | `websentry-ai` |
| Repository | `paws` |
| Workflow filename | `ci.yml` |
| Environment name | Leave empty |
| Allowed actions | Allow `npm publish` |

Save the configuration and complete npm's 2FA challenge. The workflow has
`id-token: write` and Node 24, so subsequent releases authenticate through
OIDC. No `NPM_TOKEN` secret is required. If a token secret already exists,
the workflow also supports it.

Configure the trusted publisher when you are ready to run the next release.
npm expires a new configuration if it has no successful publish within two
days. Check the [npm trusted publishing documentation](https://docs.npmjs.com/trusted-publishers/)
for current requirements.

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
