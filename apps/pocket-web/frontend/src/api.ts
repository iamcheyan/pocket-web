declare const __POCKET_WEB_API_BASE__: string | undefined;
const base = typeof __POCKET_WEB_API_BASE__ === 'string' ? __POCKET_WEB_API_BASE__ : '';
export const initialParams = new URLSearchParams(window.location.search);

export type Entity = { id: string; title?: string; url?: string; active?: boolean; loading?: boolean; createdAt?: string; updatedAt?: string; status?: string; [key: string]: unknown };
export type Health = { status?: string; engine?: { name?: string; available?: boolean; reason?: string }; downloads?: { available?: boolean; reason?: string }; mockMode?: boolean; [key: string]: unknown };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${base}/api${path}`, { ...init, headers: { 'Content-Type': 'application/json', ...init?.headers } });
  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try { const body = await response.json(); message = body?.error?.message ?? message; } catch { /* response has no JSON body */ }
    throw new Error(message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

function list<T>(data: unknown): T[] {
  if (Array.isArray(data)) return data as T[];
  if (data && typeof data === 'object') {
    const values = Object.values(data as Record<string, unknown>).find(Array.isArray);
    if (values) return values as T[];
  }
  return [];
}

export const api = {
  health: () => request<Health>('/health'),
  tabs: async () => list<Entity>(await request<unknown>('/tabs')),
  createTab: (url = '') => request<Entity>('/tabs', { method: 'POST', body: JSON.stringify({ url }) }),
  navigate: (id: string, url: string) => request<unknown>(`/tabs/${encodeURIComponent(id)}/navigate`, { method: 'POST', body: JSON.stringify({ url }) }),
  tabAction: (id: string, action: 'back' | 'forward' | 'reload' | 'duplicate') => request<unknown>(`/tabs/${encodeURIComponent(id)}/${action}`, { method: 'POST', body: '{}' }),
  closeTab: (id: string) => request<void>(`/tabs/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  reorderTabs: (ids: string[]) => request<unknown>('/tabs/order', { method: 'PUT', body: JSON.stringify({ ids }) }),
  restoreClosed: () => request<Entity>('/tabs/restore-closed', { method: 'POST', body: '{}' }),
  screenshotURL: (id: string) => `${base}/api/tabs/${encodeURIComponent(id)}/screenshot`,
  screenshot: async (id: string): Promise<string | null> => {
    const response = await fetch(api.screenshotURL(id));
    if (!response.ok) return null;
    if (!(response.headers.get('Content-Type') ?? '').startsWith('image/')) return null;
    const blob = await response.blob();
    if (!blob.size) return null;
    return new Promise<string>(resolve => {
      const reader = new FileReader();
      reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
      reader.onerror = () => resolve('');
      reader.readAsDataURL(blob);
    }).then(data => data || null);
  },
  bookmarks: async () => list<Entity>(await request<unknown>('/bookmarks')),
  addBookmark: (url: string, title: string) => request<Entity>('/bookmarks', { method: 'POST', body: JSON.stringify({ url, title }) }),
  updateBookmark: (id: string, changes: Partial<Entity>) => request<Entity>(`/bookmarks/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(changes) }),
  deleteBookmark: (id: string) => request<void>(`/bookmarks/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  importBookmarks: (bookmarks: Entity[]) => request<unknown>('/bookmarks/import', { method: 'POST', body: JSON.stringify(bookmarks) }),
  exportBookmarks: () => request<unknown>('/bookmarks/export'),
  reading: async () => list<Entity>(await request<unknown>('/reading-list')),
  addReading: (url: string, title: string) => request<Entity>('/reading-list', { method: 'POST', body: JSON.stringify({ url, title }) }),
  setPrivateMode: (enabled: boolean) => request<unknown>('/privacy/private-mode', { method: 'POST', body: JSON.stringify({ enabled }) }),
  removeReading: (id: string) => request<void>(`/reading-list/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  updateReading: (id: string, changes: Partial<Entity>) => request<Entity>(`/reading-list/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(changes) }),
  history: async (query = '') => list<Entity>(await request<unknown>(`/history${query ? `?q=${encodeURIComponent(query)}` : ''}`)),
  deleteHistory: (id: string) => request<void>(`/history/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  clearHistory: () => request<void>('/history', { method: 'DELETE' }),
  downloads: async () => { const result = await request<unknown>('/downloads'); return list<Entity>(result); },
  settings: () => request<Record<string, unknown>>('/settings'),
  saveSettings: (settings: Record<string, unknown>) => request<Record<string, unknown>>('/settings', { method: 'PUT', body: JSON.stringify(settings) }),
  privacy: (action: 'clear-cookies' | 'clear-cache' | 'delete-account-data') => request<unknown>(`/privacy/${action}`, { method: 'POST', body: '{}' }),
  accounts: async () => list<Entity>(await request<unknown>('/accounts')),
  mockAuth: (provider: string, email?: string) => request<unknown>('/auth/mock', { method: 'POST', body: JSON.stringify({ provider, email }) }),
  oauthStart: (provider: 'google' | 'github') => request<Record<string, unknown>>(`/oauth/${provider}/start`, { method: 'POST', body: '{}' }),
  oauthCallback: (provider: 'google' | 'github', values: Record<string, unknown>) => request<unknown>(`/oauth/${provider}/callback`, { method: 'POST', body: JSON.stringify(values) }),
  folders: async () => list<Entity>(await request<unknown>('/bookmark-folders')),
  createFolder: (name: string) => request<Entity>('/bookmark-folders', { method: 'POST', body: JSON.stringify({ name }) }),
  appActions: (url: string) => request<unknown>(`/apps/actions?url=${encodeURIComponent(url)}`),
};
