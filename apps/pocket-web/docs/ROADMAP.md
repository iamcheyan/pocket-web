# Roadmap and acceptance status

Status below reflects direct runtime evidence, not intended behavior.

- [x] Chromium headless CDP adapter with one process/profile per tab; real local page loaded with title and history capture.
- [x] Tab create/navigate/duplicate/reorder/restore-closed, session save/restore, and private-mode persistence filtering.
- [x] Bookmark folders and CRUD endpoints, reading list independent of bookmarks, history aggregation/search/delete/clear.
- [x] OAuth PKCE challenge/state validation, single-use state, AES-256-GCM credential persistence, no plaintext token in responses/database checked by tests.
- [x] Download Center unavailable response and remote-task synchronization against an HTTP test service.
- [x] One-command build/start serving the React UI and API from webd; frontend typecheck and locale key parity.
- [x] Genuine UI screenshots captured from the running app through Chromium/Xvfb and dimension-checked.
- [ ] Integrate and verify WebKitGTK (preferred) or CEF, including per-tab native views and navigation callbacks. Not available on this host; no adapter is claimed.
- [ ] Connect to a real `downloadsd` deployment and verify category/status synchronization. Only an explicit unavailable response and a protocol-compatible test server were exercised.
- [ ] Configure real Google and GitHub applications and complete live redirect/token exchange. Mock PKCE/authentication is implemented and provider unavailability is reported honestly.
- [ ] Verify native display rendering, touch/gamepad hardware input, and real DAP/handheld hardware layouts. Responsive profiles and colors are implemented and screenshot-tested at target dimensions.
- [ ] JS/CSS error-path coverage beyond current backend integration tests and frontend compiler/locale checks.

Current host constraints are recorded in [DEPENDENCIES](DEPENDENCIES.md). No unchecked item may be represented as complete.
