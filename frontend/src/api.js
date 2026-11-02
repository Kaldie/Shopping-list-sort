// Called on any 401 so the app can show the sign-in screen.
let onUnauthorized = () => {}
export const setUnauthorizedHandler = (fn) => (onUnauthorized = fn)

async function req(method, path, body) {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return null
  if (res.status === 401 && !path.startsWith('/auth') && path !== '/me') onUnauthorized()
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data
}

export const api = {
  stores: () => req('GET', '/stores'),
  createStore: (name, sections) => req('POST', '/stores', { name, sections }),
  updateStore: (id, patch) => req('PUT', `/stores/${id}`, patch),
  deleteStore: (id) => req('DELETE', `/stores/${id}`),
  layout: (id) => req('GET', `/stores/${id}/layout`),
  sort: (id, items) => req('POST', `/stores/${id}/sort`, { items }),
  authConfig: () => req('GET', '/auth/config'),
  me: () => req('GET', '/me'),
  loginGoogle: (credential) => req('POST', '/auth/google', { credential }),
  loginDev: (name) => req('POST', '/auth/dev', { name }),
  logout: () => req('POST', '/auth/logout'),
  forkStore: (id) => req('POST', `/stores/${id}/fork`),
  unforkStore: (id) => req('DELETE', `/stores/${id}/fork`),
  parseReceipt: async (text, listId) => (await req('POST', '/receipt/parse', { text, listId })).matches,
  applyReceipt: (storeId, listId, lines) => req('POST', `/stores/${storeId}/receipt`, { listId, lines }),
  trip: (id, items) => req('POST', `/stores/${id}/trips`, { items }),
  lists: () => req('GET', '/lists'),
  getList: (id) => req('GET', `/lists/${id}`),
  createList: (name, items) => req('POST', '/lists', { name, items }),
  updateList: (id, patch) => req('PUT', `/lists/${id}`, patch),
  deleteList: (id) => req('DELETE', `/lists/${id}`),
  completeList: (id, items) => req('POST', `/lists/${id}/complete`, items ? { items } : undefined),
  saveAsTemplate: (id, name) => req('POST', `/lists/${id}/save-as-template`, { name }),
  templates: () => req('GET', '/templates'),
  updateTemplate: (id, patch) => req('PUT', `/templates/${id}`, patch),
  deleteTemplate: (id) => req('DELETE', `/templates/${id}`),
  newListFromTemplate: (id) => req('POST', `/templates/${id}/new-list`),
  stats: (templateId, window) =>
    req('GET', `/stats?${new URLSearchParams({ templateId: templateId || '', window: window || '' })}`),
  async ocr(file) {
    const fd = new FormData()
    fd.append('image', file)
    const res = await fetch('/api/ocr', { method: 'POST', body: fd })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || res.statusText)
    return data.lines
  },
}

export const toLines = (text) =>
  text.split('\n').map((l) => l.trim()).filter(Boolean)

// Letters (incl. accents), digits and single spaces only: no *, -, ?, bullets.
export const cleanName = (s) =>
  s.replace(/['’]/g, '').replace(/[^\p{L}\p{N}\s]/gu, ' ').replace(/\s+/g, ' ').trim()

// Weights and volumes stay part of the name ("100 g feta"), other leading
// numbers are a count: "3 paprika's", "2x melk", "1 x vlees".
const UNIT = /^(g|gr|gram|kg|kilo|ml|cl|dl|l|liter|stuks?)\b/i

// "2 melk" → { name: 'melk', qty: 2 }, or null when nothing is left after cleaning.
export function parseItem(line) {
  let name = cleanName(line)
  let qty = 1
  const m = name.match(/^(\d{1,2})\s*x?\s+(.+)$/i)
  if (m && !UNIT.test(m[2])) {
    qty = Number(m[1])
    name = m[2]
  }
  return name ? { name, qty: qty || 1 } : null
}

// One entry per item (case-insensitive); repeats become a counter.
export function mergeItems(items) {
  const out = []
  for (const it of items) {
    if (!it) continue
    const key = it.name.toLowerCase()
    const hit = out.find((o) => o.name.toLowerCase() === key)
    if (hit) hit.qty += it.qty
    else out.push({ ...it })
  }
  return out
}
