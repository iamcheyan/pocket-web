# DEVICE VERIFICATION — pocket-web

Status: **DEVICE-PASS** for the captured states. Local host verification is in
`apps/pocket-web/POCKETJS-VERIFICATION.md`.

## Device

- Anbernic RG34XX, Base OS 1.5.2 (h700), kernel 4.9.170 aarch64.
- Access: Debian host -> HX90 -> USB ADB.
- Display: `/dev/fb0`, 720x480 visible page, 32 bpp.
- Runtime: `/mnt/sdcard/System/pocketjs/bin/pocket-linux-kms` (existing boot runtime, not rebuilt or replaced).
- Package: `pocket-web` sha256 `0cff02ac0f82aeaf59b319f24d4807f281d78d4a55453301e5899936da898b1f`.
- Deployed to `/storage/pjs/pocket-web/app.pocket` only.

## Procedure

1. Recorded boot launcher pid `13799` and `boot.pocket`
   sha256 `6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`.
2. Paused the launcher supervisor, stopped the launcher, and started the
   package with the same runtime and `/dev/fb0`.
3. Created a virtual keyboard with `/dev/uinput` and drove the native view
   tree through a FIFO (`uinputd-c`); no host browser and no HTML.
4. Captured `/dev/fb0` with `dd` -> base64 -> PIL.
5. Killed the package, restored the launcher, and read the pid
   `14184` and boot hash back.

Result: launcher pid changed, boot package sha256 stayed
`6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`. No write to `/storage/System`, `/storage/pocketjs`,
`/storage/baseos.conf` or `/storage/content`.

## Frames

The images are the device framebuffer. The UI was rasterized by the PocketJS
native renderer on the RG34XX, not by host Chrome.

| file | sha256 (first 16) |
|---|---|
| `docs/device/device-web-detail.png` | `e1cd77f50c0d7972` |
| `docs/device/device-web-fb-after.png` | `d59130e3329b8496` |
| `docs/device/device-web-fb-before.png` | `d59130e3329b8496` |
| `docs/device/device-web-home-focus.png` | `2146149780339f3f` |
| `docs/device/device-web-home.png` | `8d40fbea9869f2fb` |
| `docs/device/device-web-settings-toggled.png` | `ad30a30e04be55a2` |
| `docs/device/device-web-settings.png` | `1ca406660b12cbab` |

`device-web-fb-before.png` and `device-web-fb-after.png` are the boot
launcher before and after; they are byte-identical here, which is the intended
restore evidence.

Captured states: 5 states: home, focus, detail, settings, settings toggle. A detail action followed by START leaves the app on the detail tab, so the action state is not claimed.
