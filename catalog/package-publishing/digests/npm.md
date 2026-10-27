# npm digest

npm publish-side command: publish, unpublish, deprecate, version, dist-tag, access, token (npm, yarn or pnpm publish). Apply before retrying.

Target registry https://registry.npmjs.org, scope `<org-scope>` (no default: ask the user for the scope; never publish an unscoped or foreign-scoped package without asking). Posture: reviewed dry-run, one-line statement, publish. Confirm only unpublish, `latest` rollback and private→public flips.

## Version bumps
- Never hand-edit `"version"`. Use `npm version patch|minor|major` matching the change (breaking → major; name it in the changelog).
- Write the changelog entry first, in CHANGELOG.md in keep-a-changelog format.
- Versions are immutable: `npm view <pkg>@<version> version --registry https://registry.npmjs.org`; if it exists, bump again. Never republish over it.

## Dry-run review (no reviewed dry-run this session → no publish)
1. `npm pack --dry-run`: read Tarball Contents line by line. Abort and fix `files` / `.npmignore` on `.env*`, `.npmrc`, `*.pem`, `id_rsa*`, `coverage/`, `*.log`, `.DS_Store`, or source/config not meant to ship.
2. `npm publish --dry-run`.
3. State file count and unpacked size, then publish. No second confirmation.

## Registry, visibility, identity
- Read and state publish intent (`name`, `private`, `publishConfig` from package.json) before every publish.
- `"private": true` → publishing flips it public: explicit same-session confirmation, "Publishing <pkg> PUBLICLY — it's marked private — confirm?"
- `publishConfig.registry` missing or different from https://registry.npmjs.org → set it in package.json; never rely on ambient `.npmrc`.
- Already-public package: one line, publish.
- `npm whoami --registry https://registry.npmjs.org`.

## Provenance and tokens
- Default: `npm publish --provenance --access restricted` from CI when the pipeline supports it; otherwise drop the flag and say provenance is absent.
- Say once: automation uses a granular token scoped to the package(s) and `<org-scope>`, via the CI secret store (`NODE_AUTH_TOKEN`). Never classic/personal tokens, literal tokens in committed config, or `npm login` in automation.

## Deprecate, don't unpublish
- Paved road: `npm deprecate <pkg>@"<range>" "Broken <why> — upgrade to <version> instead"`.
- `npm unpublish` only on explicit confirmation with impact first: "Unpublishing <pkg>@<version> — every lockfile pinning it breaks — confirm?" (npm forbids republishing that version afterwards.)
- Moving `latest` backwards (`npm dist-tag add <pkg>@<old> latest`) is a de facto unpublish: same confirmation.

## Escalation
User says skip the dry-run: state the risk in one line ("that's how .env files end up on the public registry") and proceed on their override, noting it.

## Baseline (all registries)
- Semver is a contract; never overwrite a version; changelog before bump.
- Reviewed dry-run before anything leaves the machine.
- Provenance where supported, else say so; scoped automation tokens only.
- Sanctioned registry and scope set in the package's own config; private stays private unless confirmed.
- Unpublish/yank and tag retargeting need stated impact plus confirmation.

## Checklist
- [ ] `npm version <type>` matches change; version new on registry; changelog written
- [ ] Tarball contents reviewed; count and size stated
- [ ] `--provenance` or absence stated; granular CI token
- [ ] Intent read; name under `<org-scope>`; `npm whoami` on the target registry; private→public confirmed
- [ ] No unpublish / latest rollback without impact + confirmation
