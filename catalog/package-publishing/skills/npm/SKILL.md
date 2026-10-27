---
name: npm
description: 'Mandatory guardrails for every npm publish-side operation (publish, unpublish, deprecate, version bump, dist-tag, access, token management) via the npm, yarn, or pnpm CLIs. Enforces the package-publishing baseline with npm mechanics: npm version bumps, npm pack --dry-run tarball review, provenance publishes with granular tokens, and deprecate-over-unpublish. Triggers PROACTIVELY on any npm publish/version/unpublish command — do NOT wait for the user to invoke it.'
---

# npm

npm-specific mechanics for the package-publishing baseline (N1–N5). Apply automatically on every publish-side command — never skip. All publishes target {{SANCTIONED_REGISTRY}} under the {{ORG_SCOPE}} scope. Posture: reviewed dry-run, one-line statement, publish — confirmation only for unpublish, latest-tag rollback, and flipping a private package public.

## Version bumps (implements N1)

Never hand-edit `"version"` in package.json. Bump with the command that matches the change (it also creates the git commit and tag):

    npm version patch    # bug fix, no API change
    npm version minor    # backwards-compatible feature
    npm version major    # breaking change — name it in the changelog entry

Before the bump, write the release's changelog entry in {{CHANGELOG_CONVENTION}}. A version on the registry is immutable — check first, and if it exists, bump again; never republish over it:

    npm view <pkg>@<version> version --registry {{SANCTIONED_REGISTRY}}

## Dry-run review (implements N2)

Before every publish, inspect exactly what will leave the machine:

    npm pack --dry-run

Read the `Tarball Contents` list line by line. Abort and fix `files` in package.json (or `.npmignore`) if it contains: `.env*`, `.npmrc`, keys or certs (`*.pem`, `id_rsa*`), build junk (`coverage/`, `*.log`, `.DS_Store`), or source/config not meant to ship. Then rehearse the full publish path:

    npm publish --dry-run

State the file count and unpacked size, then publish — the reviewed dry-run is the gate; no second confirmation. No reviewed dry-run in this session → no publish.

## Registry, visibility, identity (implements N4)

Read the manifest's publish intent before every publish and state what it says:

    node -e "const p=require('./package.json'); console.log(JSON.stringify({name:p.name, private:p.private, publishConfig:p.publishConfig}, null, 2))"

- `"private": true` → publishing this flips a private package public: explicit same-session confirmation, restated with the name — "Publishing <pkg> PUBLICLY — it's marked private — confirm?"
- `publishConfig.registry` missing or ≠ {{SANCTIONED_REGISTRY}} → set it in package.json; never rely on the user's ambient `.npmrc`.
- `name` not under {{ORG_SCOPE}} → stop and ask before publishing an unscoped or foreign-scoped package.
- A package already meant to be public (`--access public`, no private flag): state it in one line and publish.

Verify identity against the target registry, not the default one:

    npm whoami --registry {{SANCTIONED_REGISTRY}}

## Provenance and credentials (implements N3)

The default publish shape ({{PROVENANCE_POLICY}}):

    npm publish --provenance --access restricted

`--provenance` works only in supported CI (GitHub Actions, GitLab CI). Publishing where it isn't supported → drop the flag and say in one line that provenance is absent. One recommendation on tokens, stated once: automation uses a granular access token scoped to the specific package(s) and {{ORG_SCOPE}}, delivered via the CI secret store (`NODE_AUTH_TOKEN`) — never a classic/personal token, a literal token in committed config, or `npm login` inside automation.

## Deprecate, don't unpublish (implements N5)

The paved road for retiring a version points consumers at the fix:

    npm deprecate <pkg>@"<range>" "Broken <why> — upgrade to <version> instead"

`npm unpublish` breaks every downstream build pinning that version, and npm forbids republishing the same name+version afterward. Only on explicit user confirmation, restated with impact first: "Unpublishing <pkg>@<version> — every lockfile pinning it breaks — confirm?" The override is recorded in the audit log. Moving `latest` backwards is a de facto unpublish — same confirmation:

    npm dist-tag add <pkg>@<version> latest    # requires the same restated confirmation

## Escalation

Rule vs user intent ("just publish it, skip the dry-run"): state the risk in one line ("publishing without a tarball review is how .env files end up on the public registry") and proceed on the user's override — noting it's recorded. Confirmation is reserved for unpublish, latest rollback, and private→public flips.

## Pre-flight checklist (before every publish-side call)

- [ ] Bump via `npm version <type>` matching the change; `npm view` shows version is new; changelog written
- [ ] `npm pack --dry-run` Tarball Contents reviewed for secrets/.env/junk; count and size stated
- [ ] `--provenance` used or its absence stated; granular token via CI secret, no classic/personal token
- [ ] Publish intent read and stated; name under {{ORG_SCOPE}}; `npm whoami` against {{SANCTIONED_REGISTRY}}; private→public flip confirmed
- [ ] No unpublish or latest-tag rollback without stated impact and confirmation; `npm deprecate` preferred

## Category baseline: Package Publishing

These rules apply to **every** package-publishing tool. An instance skill (npm, PyPI, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A publish is a broadcast, not a save: within minutes the artifact sits in mirrors, caches, and lockfiles the org will never see, and most registries never give it back. The posture is narrate-and-proceed after the dry-run; confirmation is reserved for the two things a registry makes irreversible — unpublishing, and flipping a private package public.

### Rule N1: Semver is a contract

Version numbers are promises to downstream consumers, not counters:

- The bump matches the change: breaking → major, backwards-compatible feature → minor, fix → patch. Use the tool's bump command per instance mechanics — never hand-edit the version field.
- **Never republish or overwrite an existing version.** If the version exists on the registry, bump again; a mutated version is a supply-chain incident.
- Every release carries a changelog entry in {{CHANGELOG_CONVENTION}}, written before the version bump. A major bump names the breaking change explicitly.

### Rule N2: Dry-run before publish

Inspect the exact artifact contents before anything leaves the machine, using the instance skill's dry-run mechanics:

1. Produce the artifact's file list without publishing.
2. Review it line by line for secrets and credentials, `.env` files, build junk, and source or config not meant to ship.
3. State the file count and size, then publish — the reviewed dry-run is the gate; no second confirmation after it.

A publish without a reviewed dry-run in the same session does not happen.

### Rule N3: Provenance and scoped credentials

- Publish from CI with provenance/signing where the registry supports it ({{PROVENANCE_POLICY}}); when publishing without provenance, say so in one line.
- One recommendation, stated once: automation tokens are scoped to the specific package(s) being published — never personal tokens in scripts, CI env, or committed config, and never an interactive login inside automation.

### Rule N4: Right registry, right visibility

- Packages go to {{SANCTIONED_REGISTRY}} under the org scope ({{ORG_SCOPE}}) — set the registry in the package's own publish config per instance mechanics, never rely on ambient user config.
- Read the manifest's publish intent (private flag, publish config) before every publish and state what it says.
- A manifest marked private stays private: publishing it publicly flips a private package public — that requires explicit same-session confirmation (exposure class). A package already meant to be public publishes on a one-line statement.

### Rule N5: Unpublishing breaks people

Removing a published version breaks every downstream build that pins it:

- **Deprecation with a message is the paved road** — point consumers at the replacement per instance mechanics.
- Unpublish/yank requires explicit user confirmation with the downstream impact stated first, restated before executing: "Unpublishing <pkg>@<version> — every build pinning it breaks — confirm?" The override lands in the audit log.
- Retargeting a release tag/alias to an older version is a de facto unpublish — same confirmation.

### Pre-flight checklist (run before every publish-side call)

- [ ] Bump matches the change; version doesn't already exist; changelog entry written (N1)
- [ ] Dry-run file list reviewed for secrets/.env/junk; count and size stated (N2)
- [ ] Provenance per policy or its absence stated; token scoped, not personal (N3)
- [ ] Registry sanctioned; scope correct; publish intent read and stated; private→public flip confirmed (N4)
- [ ] No unpublish/yank without stated impact and confirmation; deprecate preferred (N5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<org-scope>`: No default; no publish target can be assumed. Ask the user for the npm scope (e.g. `@acme`) before publishing, and never publish an unscoped or foreign-scoped package without asking.
