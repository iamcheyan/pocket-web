# Backend integration status

- The service uses pure Go SQLite (`modernc.org/sqlite`) for tabs, sessions, bookmarks/folders, reading list, history, settings, accounts and externally returned downloads.
- Chromium process/profile isolation exists. CDP transport and lifecycle callbacks are unavailable; navigation methods return explicit errors. WebKitGTK and CEF are unimplemented.
- OAuth exposes mock PKCE state/verifier and mock accounts. Live Google/GitHub authorization URLs, redirects, code exchange and provider credentials are not configured. AES-GCM credential storage uses a generated 32-byte `credential.key` file with mode 0600.
- Download Center client uses `DOWNLOADSD_URL`, calls health/list/create endpoints, and reports unavailable when not responding. It never downloads files locally.
- Contextual actions currently report applications unavailable pending PJOS registry/launch integration.
- Session snapshots persist nonprivate tabs only. Private tabs are not written to tabs/history/session tables. Startup crash recovery and rebuilding engine views from persisted sessions remain incomplete.
