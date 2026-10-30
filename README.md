# pocket-web

Pocket Web — a lightweight browser shell for PJOS / small-screen devices. React UI served by
the Go `webd` service, which owns tabs, history, bookmarks, reading list, sessions, OAuth state
and the optional Chromium-headless engine adapter.

- App docs: [`apps/pocket-web/README.md`](apps/pocket-web/README.md)
- Architecture / API / auth / DB: `apps/pocket-web/docs/`
- Screenshots: `apps/pocket-web/docs/screenshots/` (index: `apps/pocket-web/docs/SCREENSHOTS.md`)

## Run

```sh
cd apps/pocket-web

make setup          # go mod download + bun install
make test           # go test -race, frontend unit test, tsc
make build          # webd binary + frontend bundle
make dev            # webd on http://127.0.0.1:8088 (serves UI + API)
```

`webd` serves the built UI and the REST API from one origin on **port 8088**. Environment:
`POCKET_WEB_ADDR`, `POCKET_WEB_DATA`, `POCKET_WEB_UI`, and `CHROMIUM_BIN` (the headless engine
adapter needs this on hosts where the binary is not named `chromium`; e.g.
`CHROMIUM_BIN=/usr/bin/google-chrome`).

## Screenshots

The capture script needs a running `webd`, a Chromium binary (`CHROME_BIN` or
`google-chrome`/`chromium` on PATH), and a local fixture page on `127.0.0.1:4174` with an open
tab pointing at it:

```sh
(cd apps/pocket-web/frontend/fixtures && python3 -m http.server 4174 --bind 127.0.0.1) &
cd apps/pocket-web && CHROME_BIN=/usr/bin/google-chrome ./capture-screenshots.sh
```

It writes 16 PNGs to `apps/pocket-web/docs/screenshots/` and fails unless all 16 hashes are
distinct. See `VERIFICATION.md` for the 2026-10-31 run.
