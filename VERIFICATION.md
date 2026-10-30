# Verification — pocket-web @ 2026-10-31

- Verdict: PASS
- Commit: `aa58e07bfc2fde7eb9dc306c4af7c0b7bdc766c8` (fixes + screenshots; this document is the following commit)
- Remote: `aa58e07bfc2fde7eb9dc306c4af7c0b7bdc766c8`
- Runtime: `webd` on `http://127.0.0.1:8088` serving the built UI + REST API, `CHROMIUM_BIN=/usr/bin/google-chrome`, fixture page on `127.0.0.1:4174`; production bundle from `make build`
- Toolchain: go 1.26.0, bun 1.4.4, node v22.23.3, Chrome 155.0.8059.84

## Test gate
| command | result | notes |
|---|---|---|
| `go test -race ./...` | PASS | 15 Go tests across engine lifecycle, tabs, sessions, bookmarks, downloads, security, OAuth/credentials, migrations |
| `npm test` (node --test) | PASS | i18n key parity across en / zh / ja |
| `tsc --noEmit` | PASS | frontend typecheck |
| `bun build.mjs` / `make build` | PASS | production bundle (41 modules, main.js + main.css) + webd binary |
| `./capture-screenshots.sh` | PASS | 16 PNGs, 16 distinct SHA-256 |

`make test` runs the Go, frontend-unit and typecheck gates; green in ~28 s.

## Runtime
`GET /api/health` on this run: `status: ok`, `mockMode: true`,
`engine: {available: true, name: "ChromiumHeadlessAdapter"}` once `CHROMIUM_BIN` is set,
`downloads: {available: false, reason: "DOWNLOADSD_URL is unset; offline mock mode"}` (honest
unavailable state, as designed).

Real engine check: `POST /api/tabs {"url":"http://127.0.0.1:4174/"}` returned 201 with
`engineError: ""`; `GET /api/tabs` then showed tabs whose `title` is
"Pocket Web Local Test Page" — i.e. the Chromium CDP adapter actually navigated and captured
the page title, and `tabs-desktop.png` shows the tab strip with that engine-backed tab
selected. No fake web-content frame is substituted.

## Screenshot matrix
All 16 images were regenerated this run by `apps/pocket-web/capture-screenshots.sh` from the
live webd origin (`?page=…&theme=…&profile=…&viewport=…&locale=…&activeTab=<engine tab>`).
The script aborts unless there are exactly 16 files with 16 distinct hashes.

| screen | viewport | file | note |
|---|---|---|---|
| `bookmarks.png` | 1280×720 | `apps/pocket-web/docs/screenshots/bookmarks.png` | sha256 8fc13f958380e655 |
| `desktop.png` | 1920×1080 | `apps/pocket-web/docs/screenshots/desktop.png` | sha256 37bbd571fac7492a |
| `downloads.png` | 1280×720 | `apps/pocket-web/docs/screenshots/downloads.png` | sha256 fd73461aec9998a2 |
| `handheld.png` | 480×320 | `apps/pocket-web/docs/screenshots/handheld.png` | sha256 e1c940f990d66d71 |
| `history.png` | 1280×720 | `apps/pocket-web/docs/screenshots/history.png` | sha256 5950c7bd6a41d4d7 |
| `home-dark.png` | 1280×720 | `apps/pocket-web/docs/screenshots/home-dark.png` | sha256 9417e37f0373e3fe |
| `home-desktop.png` | 1280×720 | `apps/pocket-web/docs/screenshots/home-desktop.png` | sha256 5e5f29438edd3a1e |
| `light.png` | 1280×720 | `apps/pocket-web/docs/screenshots/light.png` | sha256 824dbefe651d354b |
| `reader.png` | 640×480 | `apps/pocket-web/docs/screenshots/reader.png` | sha256 f435966b97bde147 |
| `reading-list.png` | 1280×720 | `apps/pocket-web/docs/screenshots/reading-list.png` | sha256 754c3cf2fd618614 |
| `settings.png` | 1280×720 | `apps/pocket-web/docs/screenshots/settings.png` | sha256 2635731493907f95 |
| `tabs-desktop.png` | 1280×720 | `apps/pocket-web/docs/screenshots/tabs-desktop.png` | sha256 722ba5441a058875 |
| `viewport-1280x720.png` | 1280×720 | `apps/pocket-web/docs/screenshots/viewport-1280x720.png` | sha256 dcae35a9c736dfd3 |
| `viewport-320x240.png` | 320×240 | `apps/pocket-web/docs/screenshots/viewport-320x240.png` | sha256 33e05168d1f31adb |
| `viewport-480x320.png` | 480×320 | `apps/pocket-web/docs/screenshots/viewport-480x320.png` | sha256 2fc878b4bdb5e82b |
| `viewport-720x720.png` | 720×720 | `apps/pocket-web/docs/screenshots/viewport-720x720.png` | sha256 90677f717015f72d |

Count: 16 screenshots, 16 unique hashes (0 duplicates). `tools/check_shots.py`: all PNGs valid,
dimensions match the filename suffixes, minimum per-image RGB stddev 15.81, no blank frames.

Runtime probe (`tools/probe.mjs`) on the production origin at 1280×720 and 480×320:
HTTP 200, non-empty document, no console/page errors, no failed or 4xx/5xx requests, no
horizontal overflow.

## Fixed
- `capture-screenshots.sh` hard-coded the binary name `chromium`, which does not exist on this
  Debian host, so the documented capture command failed before taking a single PNG. → resolve
  `CHROME_BIN`, falling back to `google-chrome`/`chromium`/`chromium-browser` and failing with a
  clear message when none is found → `apps/pocket-web/capture-screenshots.sh` → all 16 captures
  above were produced through this path.
- The headless engine silently reported `available:false` because the README never mentioned
  that webd reads `CHROMIUM_BIN`. → documented the variable in `README.md`,
  `apps/pocket-web/README.md` and `docs/SCREENSHOTS.md`; with it set, `POST /api/tabs` returns
  `engineError:""` and real CDP titles.
- Added the missing repository-root `README.md` (run commands, port 8088, screenshot flow,
  environment variables) as required for this repo, which previously had no root README or
  Makefile.

## Blocked / open questions
- Downloads remain unavailable by design (`DOWNLOADSD_URL` unset); the UI shows the honest
  reason instead of faking a download.
- Live Google/GitHub OAuth is not configured; only mock/PKCE-state validation is exercised.
- No WebKitGTK/CEF adapter exists; the engine is the documented headless-Chromium development
  fallback, not a native browser GUI.

## Not verified here
- 掌机真机（RG34XX / PJOS-LINUX）未验证；本轮只在本机 headless Chrome 里跑真实进程。
- Real external websites (DNS/network restrictions); only the local fixture page was navigated.
