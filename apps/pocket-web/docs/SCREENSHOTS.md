# Screenshots

All images under `screenshots/` were captured from the running Pocket Web React UI served by `webd` at `http://127.0.0.1:8088`, using Chromium `--headless=new` under Xvfb. They are not design mockups or hand-written HTML.

## Capture procedure

From `apps/pocket-web`, with webd running and the built UI:

```sh
# webd must have a working headless engine, and a fixture server must be up
CHROMIUM_BIN=/usr/bin/google-chrome POCKET_WEB_UI=$PWD/frontend/dist \
  POCKET_WEB_ADDR=127.0.0.1:8088 go run ./backend &
(cd frontend/fixtures && python3 -m http.server 4174 --bind 127.0.0.1) &

CHROME_BIN=/usr/bin/google-chrome ./capture-screenshots.sh
```

The script uses:

```sh
DISPLAY=:99 "$CHROME" --headless=new --no-sandbox --disable-gpu \
  --window-size=WIDTH,HEIGHT --hide-scrollbars --virtual-time-budget=3000 \
  --screenshot=docs/screenshots/NAME.png \
  'http://127.0.0.1:8088/?page=…&theme=…&profile=…&viewport=…&dpi=1x'
```

The UI reads these deep-link parameters for deterministic states. The dimensions were independently read from each PNG IHDR. The capture script also computes SHA-256 for every PNG and fails unless there are exactly 16 files with 16 distinct hashes.

| File | PNG size | UI state | Browser page area |
|---|---|---|---|
| `home-desktop.png` | 1280×720 | Home, Light, Desktop, English | Chromium headless engine-rendered local test page (real CDP screenshot) |
| `home-dark.png` | 1280×720 | Home, Dark, Desktop, English | Same Chromium headless local page |
| `tabs-desktop.png` | 1280×720 | Two local fixture tabs; engine tab selected | Tab strip plus Chromium-rendered page |
| `bookmarks.png` | 1280×720 | Bookmarks, seeded records | UI/API data only |
| `reading-list.png` | 1280×720 | Reading List, offline metadata | UI/API data only |
| `downloads.png` | 1280×720 | Downloads unavailable state | Explicit Download Center status |
| `light.png` | 1280×720 | Light, Simplified Chinese | Chromium-rendered local page, localized UI |
| `history.png` | 1280×720 | History with visit counts | Seeded plus real CDP navigation |
| `settings.png` | 1280×720 | Settings and privacy controls | UI/API only |
| `handheld.png` | 480×320 | Handheld RG34XX, Dark | Compact density and Chromium page surface |
| `reader.png` | 640×480 | Reader Leaf3C, Light, Japanese | Larger typography and Chromium page surface |
| `desktop.png` | 1920×1080 | Desktop Full Layout | Full layout |
| `viewport-320x240.png` | 320×240 | Smallest viewport, Japanese | Compact responsive mode |
| `viewport-480x320.png` | 480×320 | Handheld viewport, Chinese | Compact responsive mode |
| `viewport-720x720.png` | 720×720 | DAP profile, Japanese | Two-column compact layout |
| `viewport-1280x720.png` | 1280×720 | DAP profile, Chinese | Two-column compact layout |

Engine state: `ChromiumHeadlessAdapter`, a **development fallback using headless Chromium CDP**, not WebKitGTK/CEF and not a native GUI. These screenshot states do not embed a web-content frame; the UI reports the engine honestly and no fake page content is substituted. A reachable localhost page was nevertheless used to verify real CDP navigation and title capture. External sites were unreachable due to DNS.
Download Center status at capture time: `DOWNLOADSD_URL` unset, response `available:false`, reason `DOWNLOADSD_URL is unset; offline mock mode`. OAuth provider credentials were unset; captures show the mock/available-status copy rather than a claimed live login.
