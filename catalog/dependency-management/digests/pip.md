# pip digest

pip/pip3 install or download, `uv add`, `uv pip install`. Apply before retrying.

## Venv first, never system pip
- Before any install: `$VIRTUAL_ENV` set or `which pip` inside the project. If not: `python3 -m venv .venv && source .venv/bin/activate`.
- Never `sudo pip`, never the system interpreter. Permission error = no venv; create one.
- Global tools (pipx-style): one-line note (package, why global), then proceed.

## Lockfile discipline
- Existing projects consume pins unchanged: `pip install --require-hashes -r requirements.txt` (mandatory where hashes exist) or `uv sync --frozen`.
- requirements.txt / uv.lock / poetry.lock change only via their compile step (`pip-compile`, `uv lock`), diff reviewed, committed with the manifest change. Never hand-edited or regenerated as a side effect.
- Lock conflict: merge requirements.in / pyproject.toml, re-compile. Never delete the lock.

## Before adopting a new package
- Well-known (requests, numpy, flask): quick osv check, one line ("requests: no advisories — installing").
- Unfamiliar: `pip index versions <pkg>` and `curl -s https://api.osv.dev/v1/query -d '{"package":{"name":"<pkg>","ecosystem":"PyPI"}}'`. One line: advisories, release recency (stale >18 months is a flag), license (`info.license` from `https://pypi.org/pypi/<pkg>/json`; permissive OK, copyleft or missing → flag).

## sdists run code
- Wheels don't execute code; sdists run setup.py/build backends.
- Unfamiliar package, first contact: `pip install <pkg> --only-binary :all:`.
- Only an sdist exists: `pip download <pkg> --no-deps --no-binary :all: -d ./.sdist-inspect`, read setup.py and its build hooks, then install and remove the directory.

## Registry and pinning
- First install of the session: `pip config list`; `global.index-url`/`install.index-url` (unset = pypi.org) must be https://pypi.org/simple. uv: check `UV_INDEX_URL` and `[tool.uv.index]`. Surprise mirror → stop and ask.
- Loose ranges in requirements.in/pyproject.toml; exact pins + hashes in the compiled lock.

## Baseline (all package managers)
- Lockfile is law: clean-install mode in existing projects; lockfile changes only with the manifest change that caused them, diff reviewed (new transitive packages, changed resolved URLs); conflicts re-resolved from the merged manifest, never by deleting the lockfile.
- Audit scales with familiarity: well-known → one-line advisories check; unfamiliar → also maintenance signal (years untouched is a risk) and license against MIT/Apache-2.0/BSD/ISC; copyleft or missing license → flag before proceeding.
- Install/build scripts are code execution: scripts off on first contact with an unfamiliar package, inspect, then enable. Well-known packages install normally.
- No git+http(s), tarball or direct-URL dependencies from unsanctioned hosts.
- Never sudo the package manager: a permission error means the wrong environment. Project-local by default; a global install gets a one-line note (developer CLIs only).
- User says skip a rule: state the risk in one line, proceed on their say-so, note the override is recorded. Sudo has no override.

## Checklist
- [ ] Virtualenv active; no sudo; globals noted
- [ ] Existing project → `--require-hashes` / `uv sync --frozen`; lock changed only via compile, diff reviewed
- [ ] New package: osv line; unfamiliar → recency + license, wheels only or sdist inspected
- [ ] Index is https://pypi.org/simple; no git/direct-URL deps
