# Backend dependencies and availability

- Go module uses `modernc.org/sqlite`, a pure-Go SQLite driver.
 - Chromium can be selected through `CHROMIUM_BIN` (default executable name `chromium`). Each tab launches a headless process with a private temporary profile and navigates via the Chrome DevTools Protocol.
 - Chromium is a development fallback only; it does not mean WebKitGTK or CEF is integrated. WebKitGTK/CEF libraries, `webkit2gtk-driver`, and a native display are unavailable on HX90.
- Optional Download Center integration is configured with `DOWNLOADSD_URL`. The client checks `/health` and calls `/api/downloads`; unreachable or unconfigured service is reported unavailable. No downloader is implemented.
- OAuth providers are not configured. Mock PKCE state and mock account flows are available; live callbacks/exchange need provider credentials and redirect configuration.
- `credential.key` is generated under `POCKET_WEB_DATA` (default user config `pocket-web`) with 0600 permissions and protects AES-GCM token ciphertext.
