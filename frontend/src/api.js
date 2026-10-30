async function req(method, path, body) {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return null
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
  parseReceipt: async (text) => (await req('POST', '/receipt/parse', { text })).items,
  trip: (id, items) => req('POST', `/stores/${id}/trips`, { items }),
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
