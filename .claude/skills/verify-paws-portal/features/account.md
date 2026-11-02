# Account and laptops

The user manages the account from Settings: the weekly email, a one-time key for another laptop, the
browser confirmation when `paws login` opens it, deleting synced data, and disconnecting every laptop.

## Sub-features

- `weekly-email` the `Weekly email` checkbox persists.
- `key` `New one-time key` shows `paws login --key pk_…` for a second laptop.
- `connect-card` `paws login` prints `Open <site>/connect?code=XXXX-XXXX` and opens it; the card `Connect this laptop?` shows the same code with `Connect` and `Cancel`.
- `cancel` `Cancel` stops the CLI without credentials.
- `delete-data` `Delete synced data` zeroes counts and keeps skills.
- `disconnect-all` `Disconnect all`: each laptop logs out on its next sync and removes its portal rule files.

## How to get to it (user POV)

- `/portal/settings`; the `/connect?code=` page opened by `paws login`.

## Driving it with portal.sh

Preconditions:

- `onboard` passed. This feature ends with every laptop disconnected: run `updates` before it.

- **Weekly email.** Run `portal.sh run account`. Toggle the checkbox in label `Weekly email`; `GET /me` flips; reload keeps it; toggle back.
- **Key.** Click `New one-time key`, run the shown command with a second `PAWS_HOME`. `Logged in as verify@…`; its sync pulls git.
- **Cancel.** Start `paws login` (third `PAWS_HOME`), read the code, open the URL. Card `Connect this laptop?` shows the code; `opened.txt` (the `open` recorder) has the URL. Click `Cancel`: the CLI exits non-zero; no `credentials.json`.
- **Connect.** Start again, click `Connect`. The CLI prints `Logged in as verify@…`.
- **Delete data.** Click `Delete synced data` twice (ask, confirm). `Synced data deleted.`; headline zero; skills kept.
- **Disconnect.** Click `Disconnect all` twice. `GET /skills` `connected: false`; laptop 1's next `Stop` removes `credentials.json` and every `PAWS_HOME/portal/*.md`; laptop 2's `paws sync` asks to log in. Screenshots `30-connect-confirm` … `32-disconnected`.

## Gotchas

- Without the `open` recorder, `paws login` opens a real browser tab. `portal.sh up` installs the recorder; never run `paws login` from these scripts outside that env.
- Confirmations are in-page second buttons with the same name; click `.last()`.
