#!/bin/sh
set -eu
cd "$(dirname "$0")"
BASE_URL="${BASE_URL:-http://127.0.0.1:8088}"
ENGINE_QUERY="&activeTab=$(curl -fsS "$BASE_URL/api/tabs" | python3 -c 'import json,sys; tabs=json.load(sys.stdin); print(next(t["id"] for t in reversed(tabs) if t.get("url","").startswith("http://127.0.0.1:4174")))')"
DISPLAY_NUM="${DISPLAY_NUM:-:99}"
DEST="docs/screenshots"
mkdir -p "$DEST"
rm -f "$DEST"/*.png

capture() {
  name="$1"
  size="$2"
  url="$3"
  DISPLAY="$DISPLAY_NUM" chromium --headless=new --no-sandbox --disable-gpu \
    --window-size="$size" --hide-scrollbars --virtual-time-budget=6000 \
    --screenshot="$PWD/$DEST/$name.png" "$BASE_URL$url"
}

capture home-desktop 1280,720 "/?page=home&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture home-dark 1280,720 "/?page=home&theme=dark&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture bookmarks 1280,720 "/?page=bookmarks&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture reading-list 1280,720 "/?page=reading&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture downloads 1280,720 "/?page=downloads&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture light 1280,720 "/?page=home&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=zh$ENGINE_QUERY"
capture history 1280,720 "/?page=history&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture settings 1280,720 "/?page=settings&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=en$ENGINE_QUERY"
capture tabs-desktop 1280,720 "/?page=home&theme=light&profile=desktop&viewport=1280%20%C3%97%20720&dpi=1x&locale=ja$ENGINE_QUERY"
capture handheld 480,320 "/?page=home&theme=dark&profile=handheld&viewport=480%20%C3%97%20320&dpi=1x&locale=en$ENGINE_QUERY"
capture reader 640,480 "/?page=home&theme=light&profile=reader&viewport=640%20%C3%97%20480&dpi=1x&locale=ja$ENGINE_QUERY"
capture desktop 1920,1080 "/?page=home&theme=light&profile=desktop&viewport=1920%20%C3%97%201080&dpi=1x&locale=en$ENGINE_QUERY"
capture viewport-320x240 320,240 "/?page=home&theme=dark&profile=handheld&viewport=320%20%C3%97%20240&dpi=1x&locale=ja$ENGINE_QUERY"
capture viewport-480x320 480,320 "/?page=home&theme=light&profile=handheld&viewport=480%20%C3%97%20320&dpi=1x&locale=zh$ENGINE_QUERY"
capture viewport-720x720 720,720 "/?page=home&theme=light&profile=dap&viewport=720%20%C3%97%20720&dpi=1x&locale=ja$ENGINE_QUERY"
capture viewport-1280x720 1280,720 "/?page=home&theme=light&profile=dap&viewport=1280%20%C3%97%20720&dpi=1x&locale=zh$ENGINE_QUERY"

python3 - <<'PY'
import glob, hashlib
paths = sorted(glob.glob('docs/screenshots/*.png'))
hashes = {hashlib.sha256(open(p, 'rb').read()).hexdigest() for p in paths}
print(f'{len(paths)} PNG files, {len(hashes)} unique sha256')
if len(paths) != 16 or len(hashes) != 16:
    raise SystemExit('screenshot uniqueness check failed')
PY
