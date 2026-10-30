# Pocket Web webd backend

Standalone Go service using SQLite (`modernc.org/sqlite`). Run from this directory with `go run .`; configure `POCKET_WEB_ADDR`, `POCKET_WEB_DATA`, `CHROMIUM_BIN`, and optionally `DOWNLOADSD_URL`.

`GET /api/health` reports engine and download service status. `ChromiumHeadlessAdapter` launches one headless Chromium process with a temporary, isolated profile per tab and uses CDP for navigation, history, reload, and stop commands. It is a development fallback, not WebKitGTK or CEF. Chromium process and profile cleanup is tied to the view lifecycle.

Tabs, sessions, bookmarks/folders, reading list, history, settings, accounts, and Download Center task snapshots use SQLite. Private tabs are held in memory, are excluded from tab/session/history persistence, and are not restored. Session snapshots can be saved and restored through `/api/sessions/save`, `/api/sessions/restore-previous`, and `/api/sessions/restore-tabs`.

The Download Center client calls `DOWNLOADSD_URL/health` and `/api/downloads`; absent or unreachable service status remains unavailable, with no local downloader or fabricated tasks. OAuth exposes mock PKCE state and mock accounts; live provider authorization/callback exchange needs credentials and configuration. Token storage uses AES-256-GCM and `credential.key` mode 0600; API responses never return stored tokens.

Errors are `{ "error": { "code": "...", "message": "..." } }`. Collection routes generally return bare arrays. See [API.md](API.md) for route details and [ROADMAP.md](ROADMAP.md) for unfinished integrations.

