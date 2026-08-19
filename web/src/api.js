// API 客户端：REST + WebSocket 封装。

const BASE = '/api'

async function request(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  }
  if (body !== undefined) {
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(BASE + path, opts)
  const data = await res.json().catch(() => null)
  if (!res.ok || (data && data.code !== 0)) {
    const msg = (data && data.message) || `HTTP ${res.status}`
    throw new Error(msg)
  }
  return data ? data.data : null
}

// parseResp 解析非 JSON 请求（如上传）的响应。
async function parseResp(res) {
  const data = await res.json().catch(() => null)
  if (!res.ok || (data && data.code !== 0)) {
    const msg = (data && data.message) || `HTTP ${res.status}`
    throw new Error(msg)
  }
  return data ? data.data : null
}

export const api = {
  listServices: () => request('GET', '/services'),
  getService: (id) => request('GET', `/services/${id}`),
  createService: (svc) => request('POST', '/services', svc),
  updateService: (id, svc) => request('PUT', `/services/${id}`, svc),
  deleteService: (id) => request('DELETE', `/services/${id}`),
  startService: (id) => request('POST', `/services/${id}/start`),
  stopService: (id) => request('POST', `/services/${id}/stop`),
  restartService: (id) => request('POST', `/services/${id}/restart`),
  getStatus: (id) => request('GET', `/services/${id}/status`),
  getLogs: (id, params = {}) => {
    const qs = new URLSearchParams()
    if (params.keyword) qs.set('keyword', params.keyword)
    if (params.level) qs.set('level', params.level)
    if (params.regex) qs.set('regex', params.regex)
    if (params.limit) qs.set('limit', params.limit)
    if (params.caseSensitive) qs.set('caseSensitive', 'true')
    const s = qs.toString()
    return request('GET', `/services/${id}/logs${s ? '?' + s : ''}`)
  },
  getMetrics: (id) => request('GET', `/services/${id}/metrics`),
  getSystemInfo: () => request('GET', '/system/info'),
  getSystemStats: () => request('GET', '/system/stats'),
  getTopProcesses: (limit = 10) => request('GET', `/system/topcpu?limit=${limit}`),
  killProcess: (pid) => request('POST', `/system/kill/${pid}`),
  getConfig: () => request('GET', '/config'),
  updateConfig: (cfg) => request('PUT', '/config', cfg),
  discoverJDKs: () => request('POST', '/config/discover-jdks'),
  listDirs: (path) => request('GET', `/dirs${path ? '?path=' + encodeURIComponent(path) : ''}`),
  listFiles: (path) => request('GET', `/files${path ? '?path=' + encodeURIComponent(path) : ''}`),
  readFile: (path) => request('GET', `/files/content?path=${encodeURIComponent(path)}`),
  writeFile: (path, content) => request('PUT', `/files/content?path=${encodeURIComponent(path)}`, content),
  downloadFile: (path) => {
    window.location.href = `/api/files/download?path=${encodeURIComponent(path)}`
  },
  uploadFile: (dir, file) => {
    const fd = new FormData()
    fd.append('file', file)
    return fetch(`/api/files/upload?path=${encodeURIComponent(dir)}`, { method: 'POST', body: fd }).then(parseResp)
  },
  renameFile: (path, newName) => request('POST', '/files/rename', { path, newName }),
  deleteFile: (path) => request('DELETE', `/files/delete?path=${encodeURIComponent(path)}`),
  mkdirFile: (path, name) => request('POST', '/files/mkdir', { path, name }),
  copyFile: (source, dest) => request('POST', '/files/copy', { source, dest }),
}

// WebSocket 工具：连接并分发消息。
export function connectWS(path, serviceId, handlers) {
  const params = serviceId ? `?serviceId=${encodeURIComponent(serviceId)}` : ''
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(`${proto}//${location.host}${path}${params}`)
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      if (handlers.onMessage) handlers.onMessage(msg)
    } catch (e) {
      if (handlers.onError) handlers.onError(e)
    }
  }
  ws.onclose = () => handlers.onClose && handlers.onClose()
  ws.onerror = (e) => handlers.onError && handlers.onError(e)
  return ws
}
