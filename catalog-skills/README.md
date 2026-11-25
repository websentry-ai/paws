# Catalog skills

The skills Unbound writes for the paws catalog, one folder per skill: `catalog-skills/<category>/<skill>/`.

- **The source of truth for updates.** Each catalog entry links to its folder here. The backend's daily update check compares a user's pinned commit with the latest commit that changed the folder, and offers the difference as an update to review.
- **Not part of the CLI.** The `paws` binary and the npm package do not read or include this folder. paws installs these skills from the backend.
- **Exact copies.** Each folder holds only the files paws installs (`SKILL.md`, `config.yaml`). Its content hash (sha256 over each file's `path\0length\0bytes`, in path order) equals the hash stored in the catalog. Do not add other files to a skill folder: they change the hash and ship to users.

To change a skill, edit its folder and open a PR. Users get the change as an update once it merges to `main`.
