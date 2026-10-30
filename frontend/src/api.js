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
