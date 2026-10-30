# webd API

All paths below are prefixed `/api`. Errors use `{ "error": { "code": "...", "message": "..." } }`; list responses are bare JSON arrays.

- `GET /health`: status, `engine:{name,available,reason}`, `downloads:{available,reason}`, `mockMode`.
- `GET/POST /tabs`; `GET/DELETE /tabs/{id}`; `PUT /tabs/order` with `{ids:[...]}`; `POST /tabs/{id}/navigate` with `{url}`; `/back`, `/forward`, `/reload`, `/duplicate`; `POST /tabs/restore-closed`.
- `GET /tabs/{id}/screenshot`: PNG image of the active page captured via CDP `Page.captureScreenshot`. Returns `503` JSON (`engine_unavailable`) when the tab has no live view, the engine lacks screenshot support, or capture fails.
- `POST /sessions/save`; `POST /sessions/restore-previous`; `POST /sessions/restore-tabs`.
- `GET/POST /bookmarks`; `PUT/DELETE /bookmarks/{id}`; `GET/POST /bookmark-folders`; `POST /bookmarks/import`; `GET /bookmarks/export`.
- `GET/POST /reading-list`; `PUT/DELETE /reading-list/{id}`; supports `q` and `sort=title`.
- `GET /history?q=...`; `DELETE /history` clears all; `DELETE /history/{id}` removes one.
- `GET /downloads`; `GET/PUT /settings`; `GET/DELETE /accounts`.
- `POST /oauth/{google|github}/start` and `/callback`; `POST /auth/mock`.
- `POST /privacy/private-mode` with `{enabled}`, `/clear-history`, `/clear-cookies`, `/clear-cache`, `/delete-account-data`.
- `GET /apps/actions?url=...` reports unavailable contextual apps until a registry is configured.

Tab records contain `id,title,url,active,private,loading`. Bookmark and reading list records include `id,title,url`; reading records also include `read,offlineMetadata,createdAt`. History records include `visitCount,lastVisit`.

The Chromium fallback launches isolated headless processes/profiles and uses CDP for navigation, back/forward/reload, and current page metadata. This is a development fallback; WebKitGTK/CEF are not implemented. OAuth start performs mock PKCE setup; callback validates state/verifier and returns unavailable until provider credentials and code exchange are configured. Download Center uses `DOWNLOADSD_URL` and returns unavailable when no service responds; Pocket Web never downloads files itself.
