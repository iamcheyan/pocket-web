// Web client: tabs, history, bookmarks and reading list over webd.
// There is no browser engine on the device: this is an explicit window shell.
import { createPortClient } from "./client-kit";
import type { AppConfig, Metric, Section, SettingRow } from "./model";

const sections: Section[] = [
  {
    id: "tabs", title: "TABS", jp: "タブ", items: [
      { id: "tab1", title: "PocketJS Docs", subtitle: "pocketjs.dev/docs", meta: "active", badge: "TAB", body: "https://pocketjs.dev/docs/native-contract", lines: [{ label: "engine", value: "none" }, { label: "source", value: "mock" }, { label: "tls", value: "ok" }] },
      { id: "tab2", title: "Local fixture", subtitle: "127.0.0.1:4174", meta: "loaded", badge: "TAB", body: "http://127.0.0.1:4174/fixture.html", lines: [{ label: "engine", value: "none" }, { label: "source", value: "fixture" }, { label: "tls", value: "n/a" }] },
      { id: "tab3", title: "New tab", subtitle: "about:blank", meta: "idle", badge: "TAB", body: "about:blank", lines: [{ label: "engine", value: "none" }, { label: "source", value: "-" }, { label: "tls", value: "n/a" }] },
    ],
  },
  {
    id: "marks", title: "LIBRARY", jp: "ライブラリ", items: [
      { id: "bm1", title: "Bookmarks", subtitle: "18 saved · 4 folders", meta: "synced", badge: "BM", body: "Bookmark folders and export." },
      { id: "rl1", title: "Reading list", subtitle: "6 unread · 12 read", meta: "sorted", badge: "READ", body: "Articles queued for offline reading." },
      { id: "hist", title: "History", subtitle: "248 visits · today 12", meta: "private off", badge: "HIST", body: "History and private-mode state." },
    ],
  },
];

const metrics: Metric[] = [
  { label: "TABS", value: "3" },
  { label: "MARKS", value: "18" },
  { label: "ENGINE", value: "none" },
];

const settings: SettingRow[] = [
  { id: "engine", label: "Engine", jp: "エンジン", value: "NONE", options: ["NONE"] },
  { id: "private", label: "Private mode", jp: "プライベート", value: "OFF", options: ["OFF", "ON"] },
  { id: "downloads", label: "Downloads", jp: "ダウンロード", value: "PAUSED", options: ["PAUSED", "ACTIVE"] },
];

export const APP: AppConfig = {
  title: "WEB",
  jp: "ブラウザ",
  kind: "browser",
  page: "WEB",
  tabs: ["タブ", "ページ", "設定"],
  actions: ["BACK", "RELOAD", "STAR", "PRIVATE"],
  settingsTitle: "SHELL",
  settingsJp: "ブラウザ設定",
  deviceGap: "no engine or webd bridge.",
  sourceNote: "/api/tabs · /api/bookmarks",
};

export const client = createPortClient({
  resource: "web",
  fixture: { sections, metrics, settings },
  endpoints: {
    list: "/api/tabs",
    get: (id) => `/api/tabs/${encodeURIComponent(id)}`,
    action: (name) => `/api/${name === "STAR" ? "bookmarks" : "tabs"}/${encodeURIComponent(name.toLowerCase())}`,
  },
  act(name, payload, bundle) {
    const item = bundle.sections.flatMap((section) => section.items).find((entry) => entry.id === payload?.id);
    if (!item) return { ok: false, message: "no tab selected" };
    if (name === "BACK") { item.state = "back"; return { ok: true, message: `${item.title}: history back`, }; }
    if (name === "RELOAD") { item.state = "reloaded"; return { ok: true, message: `${item.title}: reload requested · engine unavailable` }; }
    if (name === "STAR") { item.badge = "BM"; return { ok: true, message: `${item.title} bookmarked` }; }
    if (name === "PRIVATE") { item.state = "private"; return { ok: true, message: "private mode toggled" }; }
    return { ok: false, message: `${name} unavailable` };
  },
});
