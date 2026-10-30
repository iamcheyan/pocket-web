# Pocket Web

Pocket Web is a lightweight browser shell for PJOS and small-screen devices. Its React interface talks to a Go `webd` service, which owns browser state, persistence, OAuth, and integrations. Pocket Web does not implement a browser engine, HTML/CSS/JavaScript renderer, or network stack.

## Start

Prerequisites: Go, Bun, Chromium, and a display; this host uses Xvfb for headless capture. From `apps/pocket-web`:

```sh
make build
make dev
```

`make dev` builds the React bundle and starts **webd on `http://127.0.0.1:8088`**, serving both the UI and REST API from one origin. Set `POCKET_WEB_DATA` to choose storage; `POCKET_WEB_ADDR` changes the listen address; `POCKET_WEB_UI` can point at a different built UI directory. The headless engine adapter resolves its binary from `CHROMIUM_BIN` (for example `CHROMIUM_BIN=/usr/bin/google-chrome` on Debian); without it `/api/health` reports `engine.available:false`.

For frontend hot reload, run `cd frontend && npm run dev` with webd running separately. Localhost development origins receive CORS headers.

## Verified behavior

- `make test` covers SQLite migrations, collection CRUD/import-export, tab duplicate/restore, session recovery, private-mode no-persist, OAuth state/PKCE, AES-GCM credential storage without plaintext responses, Download Center unavailable/available adapters, and BrowserEngine lifecycle.
- Chromium is the **development fallback**, reached through a separate headless Chromium process and CDP session per tab. A reachable local page was loaded and its title/history captured; this is not WebKitGTK or CEF integration and it is not a native GUI.
- `downloadsd` is absent on this host. `/api/downloads` returns `available:false` with a reason; the client never downloads files itself.
- Live Google/GitHub OAuth credentials are unset. Mock authentication and PKCE start/callback validation are available; the callback reports provider unavailability instead of claiming a real login.
- Deterministic sample bookmarks, reading-list entries, and history are inserted once into a new database.

## Runtime limitations

WebKitGTK 4.0/4.1 and `webkit2gtk-driver` are unavailable on HX90, and no WebKitGTK adapter is shipped. No native browser GUI is claimed. External websites could not be reached during this development run because DNS was unavailable; a real local HTTP page was used to validate CDP navigation. See [DEPENDENCIES](docs/DEPENDENCIES.md), [ROADMAP](docs/ROADMAP.md), and [SCREENSHOTS](docs/SCREENSHOTS.md).

## Documents

- [Architecture](docs/ARCHITECTURE.md)
- [Database](docs/DATABASE.md)
- [API](docs/API.md)
- [Authentication](docs/AUTH.md)
- [Dependencies and host status](docs/DEPENDENCIES.md)
- [Roadmap](docs/ROADMAP.md)
- [Screenshot capture](docs/SCREENSHOTS.md)
