# Database

`webd` owns SQLite persistence using the pure Go `modernc.org/sqlite` driver. The frontend accesses data only through the REST API.

## Tables

| Table | Purpose |
|---|---|
| `tabs` | Tab id, title, URL, position, active flag, and private flag for non-private tab rows |
| `sessions` | Snapshot of persisted tabs (`tabs_json`) and creation time |
| `bookmarks` | Bookmark id, title, URL, folder reference, creation time |
| `bookmark_folders` | Folder id, name, optional parent |
| `reading_list` | Independent reading entries with `read` state and `offline_metadata` |
| `history` | Unique URL, title, aggregated `visit_count`, and `last_visit` |
| `downloads` | Mirror of task records returned by Download Center, not a local queue |
| `settings` | JSON-encoded user/service settings, including `privateMode` |
| `accounts` | Mock/provider account records; no token plaintext |
| `oauth_tokens` | AES-256-GCM ciphertext keyed by account id |

## Behavior

- Migration runs at server startup and is covered by a temporary-database test.
- Reading list is a separate table and separate API; it is not derived from bookmarks.
- History uses URL conflict aggregation: repeat navigation increments `visit_count` and updates `last_visit`.
- `DELETE /api/history` clears all rows; `DELETE /api/history/{id}` removes one and returns 404 if absent.
- Tab close captures a recently-closed URL for restore; duplicate resolves the source URL from persistence or the active engine view.
- `POST /api/sessions/save` stores non-private tabs. Restore recreates a new engine view per saved tab, persists the tab rows, and reports the restored list.
- Private mode is persisted in `settings`, and every individually private (or globally private) tab excludes its URL/title from tab-history rows, session snapshots, and the API's history list. Tests verify no history/tab/session rows are written for private navigation.
- Clear-cache/cookies closes engine views, removes temporary Chromium profiles, and clears stored tabs/sessions. Clear-history deletes history rows. Delete-account-data clears accounts and encrypted tokens.

OAuth token bytes are ciphertext at rest through `CredentialStore`; the 32-byte key lives in a `0600` file under the data directory. Tests verify ciphertext differs from plaintext and that responses/database don't expose the token. See [AUTH](AUTH.md).

Deterministic demo bookmarks, reading entries, and history are seeded once into a new database via the `demo_seeded` settings marker.
