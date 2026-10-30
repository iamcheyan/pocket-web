# POCKETJS VERIFICATION — pocket-web

Round: PocketJS refactor. Device acceptance is a later pass.

## Shape

- Pocket app: `apps/pocket-web/pocket/`
- Frozen React reference: `apps/pocket-web/frontend/` (design reference; not extended).
- TSX files in the port: **6** (`main.tsx`, `app.tsx`, `ui.tsx`, three screens).
- Native tree only: `View`/`Text` from `@pocketjs/framework/solid/components`; no DOM, iframe, texture screenshot, or `dangerouslySetInnerHTML`.

## Manifest

- id: `dev.pocket-web.pocket`
- entry: `src/main.tsx`
- output: `pocket-web`
- viewport: `360×240 @2x` → physical `720×480`
- capabilities: `text.glyphs.baked`, `input.buttons`
- target: `rocknix-h700-kms`

## Admission / compile

```sh
cd apps/pocket-web/pocket
./setup-sdk.sh
SDK=${POCKETJS_SDK:-$HOME/development/.pocketjs-star-drift-sdk}
bun "$SDK/tools/pocket.ts" check \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json
bun "$SDK/tools/pocket.ts" compile \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json --output out.pocket
```

Admission: `sha256:46f0a9eab20104150a70d5dae0644a52d87674ddf12647e118881725569e0ef4`; TypeScript passes for the app module graph. Package: `out.pocket`, sha256 `f72e4cc31c376da8a26842a52a01cbe301fe877298f8069120204099e34996dd`.

## Frames

Rendered with the SDK wasm core at `renderScaled(2)` (720×480), not a browser:

```sh
cd "$POCKET_VERIFY"
bun tools/render-pocket.ts \
  --dir repos/pocket-web/apps/pocket-web/pocket \
  --script repos/pocket-web/apps/pocket-web/pocket/frames.json \
  --out repos/pocket-web/docs/pocket
```

| file | state | sha256 (first 16) |
|---|---|---|
| `docs/pocket/screen-home.png` | home | `17852c43d0319e4f` |
| `docs/pocket/screen-home-focus.png` | home-focus | `5e81e5c8389978fa` |
| `docs/pocket/screen-detail.png` | detail | `8e67bcbc1d9bde8d` |
| `docs/pocket/screen-detail-action.png` | detail-action | `dfe21c2654cc61a8` |
| `docs/pocket/screen-settings.png` | settings | `1967678c61a2fae1` |
| `docs/pocket/screen-settings-toggled.png` | settings-toggled | `e8e5203f86f75153` |

All six files have distinct bytes; the renderer rejects duplicate frames. The
tape is: move focus, open the item, run the first detail action, switch to
settings, move a row, toggle it.

Screens: Tabs/library, page shell, browser settings. Interactions: BACK/RELOAD/STAR/PRIVATE update tab state; no page is rendered.

## Adapter

- **fixture** — this port runs from JSON baked into the package and labels the
  header `fixture`. `client.ts` also contains the `http` and `bridge` adapters
  behind the same `list()/get()/action()` surface.
- `http` uses `@pocketjs/framework/net-api`; valid only in the host dev profile.
- `bridge` uses `host.__pvc(JSON.stringify(request))`; the runtime forwards to
  the Go daemon on `127.0.0.1:<port>` with a hard timeout.
- Device gap: No browser engine and no webd bridge. This round is an explicit window shell; real pages need the runtime's engine adapter.
- Daemon contract: /api/tabs, /api/bookmarks, /api/reading-list; engine=none.

## Not done / open questions

- Device run, saves/annotations and the bridge itself are out of this round.
- Real media/audio, OAuth, emulator, engine and system writes stay unimplemented
  where the original product already marked them mock or unavailable.
- Long lists use the six-item fixture surface; paging/virtual lists are a
  follow-up once the bridge provides paginated data.
