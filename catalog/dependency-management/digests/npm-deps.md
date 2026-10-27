# npm-deps digest

Adding a package: npm install/i/add <pkg>, yarn add, pnpm add or bun add. Apply before retrying. Publishing (`npm publish`, version, dist-tag) is a different skill.

## ci vs install
- Existing project with package-lock.json, no dependency change intended → `npm ci`. Bare `npm install` there can rewrite the lockfile; don't.
- Deliberately adding/changing a dependency → `npm install <pkg>`, then review the package-lock.json diff; commit it with the package.json change.
- Lockfile merge conflict: resolve package.json, then `npm install` to re-resolve. Never delete the lockfile.

## Before adopting a new package
- Well-known (react, lodash, express): quick advisories check, one line ("lodash: no advisories — installing").
- Unfamiliar: `npm audit --omit=dev` and `npm view <pkg> time.modified license`. One line: advisories, last modified (stale >18 months is a flag), license (permissive MIT/Apache-2.0/BSD/ISC; copyleft or missing → flag first).

## Install scripts and npx
- Unfamiliar package, first contact: `npm install <pkg> --ignore-scripts`; read the `scripts` block in `node_modules/<pkg>/package.json` and what it invokes; then `npm rebuild <pkg>` deliberately.
- `npx <pkg>` downloads and runs registry code. Well-known runners (create-next-app, prettier, tsc) flow freely. Unfamiliar target: one-line check above, and pin `npx <pkg>@<version>`, never a floating tag.

## Registry and pinning
- First install of the session: `npm config get registry` must be https://registry.npmjs.org; anything else → stop and ask.
- Manifest uses caret ranges; the lockfile carries exact versions.

## Scope
- Never `sudo npm`. EACCES = root-owned prefix or wrong Node install; fix it (nvm/volta prefix).
- `npm install -g`: one-line note (package, why global), then proceed. Prefer project-local devDependencies + local `npx`.

## Baseline (all package managers)
- Lockfile is law: clean-install mode in existing projects; lockfile changes only with the manifest change that caused them, diff reviewed (new transitive packages, changed resolved URLs); conflicts re-resolved from the merged manifest, never by deleting the lockfile.
- Audit scales with familiarity: well-known → one-line advisories check; unfamiliar → also maintenance signal (years untouched is a risk) and license against MIT/Apache-2.0/BSD/ISC; copyleft or missing license → flag before proceeding.
- Install/build scripts are code execution: scripts off on first contact with an unfamiliar package, inspect, then enable. Well-known packages install normally.
- No git+http(s), tarball or direct-URL dependencies from unsanctioned hosts.
- Never sudo the package manager: a permission error means the wrong environment. Project-local by default; a global install gets a one-line note (developer CLIs only).
- User says skip a rule: state the risk in one line, proceed on their say-so, note the override is recorded. Sudo has no override.

## Checklist
- [ ] Existing project → `npm ci`; dependency change → lockfile diff reviewed
- [ ] New package: advisories line; unfamiliar → `npm view` workup, `--ignore-scripts` then `npm rebuild`
- [ ] Registry is https://registry.npmjs.org; no git/tarball URLs
- [ ] No sudo; `-g` noted; unfamiliar npx targets checked and version-pinned
