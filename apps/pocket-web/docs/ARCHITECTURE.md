# Architecture

Pocket Web separates the TypeScript/React/Vite interface from the Go `webd` service. The browser UI calls `webd` over its JSON API; `webd` owns browser state, persistence, authentication, and integrations.

```text
React + TypeScript + Vite
          │ REST JSON
          ▼
        webd (Go)
        ├── SQLite (modernc.org/sqlite)
        ├── BrowserEngine → Chromium headless development adapter
        ├── OAuth gateway / credential store
        ├── Download Center client
        └── PJAppRegistry / AppLaunchAPI
```

The frontend must not read SQLite or the filesystem, nor keep authoritative browser state. The backend provides the API contract and coordinates tabs, sessions, bookmarks, reading list, history, settings, and system integrations. SQLite is the durable store. A browser engine adapter owns a separate view/context for each tab.

Pocket Web is a browser shell, not a browser engine. It does not implement HTML/CSS/JavaScript parsing, rendering, or networking. On this development host the available path is Chromium headless; this adapter is a development fallback and does not constitute WebKitGTK or CEF integration. The page surface in screenshots must identify the actual engine and mock/real state.

## Runtime boundaries

- `BrowserEngine` and `BrowserView` isolate engine-specific lifecycle and navigation.
- `DownloadCenterClient` delegates task creation and status to `downloadsd`; Pocket Web does not download files or maintain a competing queue.
- OAuth provider callbacks validate state and use PKCE. Credential persistence encrypts tokens through the credential store.
- Contextual application actions go through replaceable app-registry and launch interfaces.
- Mock adapters provide deterministic demos where external credentials or services are unavailable. Mock behavior must be labeled and must not report successful real integration.

## Privacy boundary

Private browsing state is memory-only: private tab URLs and visits must not be persisted in history or session records. Clear-history, clear-cache, clear-cookies, and account-data deletion operations must remove their corresponding stored data and engine profile data where supported. Logs and API responses must not expose access tokens or cookies.

## Data flow

A UI action calls an API endpoint. `webd` validates it, updates its service state and database as appropriate, then responds with JSON. Engine navigation callbacks update history for non-private tabs. The UI renders returned state; it does not infer or persist authoritative browser state itself.
