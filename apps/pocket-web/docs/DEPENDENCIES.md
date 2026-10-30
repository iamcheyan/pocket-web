# Dependencies and host status

## Application

- Go `webd` service using pure-Go `modernc.org/sqlite`.
- React + TypeScript UI. It talks only to webd over HTTP; no Electron, no direct SQLite/filesystem access from the UI.
- Chromium is the development rendering fallback, driven via CDP with one headless process and temporary profile per tab.
- Bun is used for the offline-compatible frontend bundle in this environment; TypeScript performs the typecheck. The project previously declared Vite as the intended dev/build system, but Vite and its plugin were not available in the local package cache on this machine.

## Host availability observed

| Dependency | State |
|---|---|
| Chromium | available (`Chromium 152.0.7977.75`) |
| Xvfb | available; used for headless screenshot capture |
| WebKitGTK 4.0/4.1 | unavailable through `pkg-config` |
| `webkit2gtk-driver` | unavailable |
| CEF | not installed/verified |
| Download Center (`downloadsd`) | inactive; `DOWNLOADSD_URL` unset |
| Google/GitHub OAuth credentials | unset; mock flow used |
| Native display | no active display at initial inspection |

## Browser engine reality

`ChromiumHeadlessAdapter` launches `chromium --headless=new` per tab with an isolated `--user-data-dir` and talks CDP over the DevTools WebSocket. Navigation, reload, history navigation, title/URL retrieval, and lifecycle cleanup are implemented. A real localhost page was loaded and its title and history visit were observed. This is **not** WebKitGTK/CEF integration and is not a native GUI browser. No WebKitGTK adapter is claimed.

External public web pages could not be reached during verification because DNS resolution failed. A localhost HTTP page was used to prove CDP page loading. The UI/screenshots report this engine and its fallback status rather than pretending a native engine.

## Download Center

The client checks `DOWNLOADSD_URL/health`, then reads `GET /api/downloads` and mirrors returned task identifiers/status/category into the local `downloads` table. It does not implement a downloader or queue. With the URL unset, `/api/downloads` returns:

```json
{"available":false,"reason":"DOWNLOADSD_URL is unset; offline mock mode","items":[]}
```

A test HTTP server verified the available-path synchronization behavior; no real `downloadsd` deployment was exercised.

## NixOS/system changes

No NixOS configuration or system package change was made. Project dependencies were resolved from local module/package caches where available; Nix network fetches for WebKitGTK failed.
