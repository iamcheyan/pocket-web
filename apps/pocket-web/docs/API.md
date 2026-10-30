# HTTP API

The frontend calls `webd` JSON endpoints; it has no direct database or filesystem access. Errors use the common envelope `{"error":{"code":"...","message":"..."}}`, with HTTP status codes matching the error class. The server's route implementation is authoritative; this document describes the required public surface from the project objective.

| Area | Routes |
|---|---|
| Health | `GET /api/health` |
| Tabs | `GET/POST /api/tabs`; `GET /api/tabs/{id}`; `PUT /api/tabs/order`; `POST /api/tabs/{id}/navigate`, `/back`, `/forward`, `/reload`, `/duplicate`; `DELETE /api/tabs/{id}`; `POST /api/tabs/restore-closed` |
| Bookmarks | `GET/POST /api/bookmarks`; `PUT/DELETE /api/bookmarks/{id}`; `GET/POST /api/bookmark-folders`; import/export endpoints |
| Reading list | `GET/POST/PUT/DELETE /api/reading-list[/{id}]` |
| History | `GET /api/history`; `DELETE /api/history/{id}`; `DELETE /api/history` |
| Downloads | `GET /api/downloads` returns `{available,reason,items}` sourced from `downloadsd`; it has no local download queue. |
| Browser view | `GET /api/tabs/{id}/screenshot` returns engine-rendered PNG bytes for the active tab, or a JSON error. |
| Sessions | `POST /api/sessions/save`, `/restore-previous`, `/restore-tabs`. |
| Accounts/auth | `GET/POST/DELETE /api/accounts`; `POST /api/oauth/{google\|github}/start` and `/callback`; `POST /api/auth/mock` |
| Settings | `GET/PUT /api/settings` |
| Privacy | `POST /api/privacy/clear-cookies`, `/clear-cache`, `/delete-account-data` |
| App actions | `GET /api/apps/actions?url=...` |

Authentication endpoints must validate OAuth state and PKCE verifier/challenge. Mock authentication is a demo path and must be identifiable as mock. Download status is obtained from the Download Center client, which can explicitly report unavailable/offline mock operation. Never return credential material in API responses.
