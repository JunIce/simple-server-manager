import { useEffect, useState } from 'react'
import { api } from '../api'
import DirPicker from './DirPicker'

const EMPTY_JDK = { id: '', name: '', version: '', home: '', default: false }
const inputCls =
  'w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'
const labelCls = 'block text-xs font-medium text-slate-500 mb-1'

export default function Settings() {
  const [server, setServer] = useState({ port: 8080, host: '0.0.0.0', dataDir: './data', fileRoot: './data', auth: { enabled: false, token: '' } })
  const [log, setLog] = useState({ outputDir: './data/logs', maxSizeMB: 50, maxFiles: 10, bufferSize: 10000 })
  const [monitor, setMonitor] = useState({ intervalSeconds: 5 })
  const [jdks, setJdks] = useState([])
  const [discovered, setDiscovered] = useState([])
  const [showJdkForm, setShowJdkForm] = useState(false)
  const [editingJdk, setEditingJdk] = useState(null)
  const [jdkForm, setJdkForm] = useState(EMPTY_JDK)
  const [error, setError] = useState('')
  const [ok, setOk] = useState('')
  const [busy, setBusy] = useState('')
  const [showFileRootPicker, setShowFileRootPicker] = useState(false)

  const load = async () => {
    try {
      const cfg = await api.getConfig()
      setServer({ port: cfg.server.port, host: cfg.server.host, dataDir: cfg.server.dataDir, fileRoot: cfg.server.fileRoot || cfg.server.dataDir || './data', auth: { enabled: cfg.server.auth.enabled, token: cfg.server.auth.token || '' } })
      setLog(cfg.log)
      setMonitor(cfg.monitor)
      setJdks(cfg.jdks || [])
      setError('')
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => { load() }, [])

  const setField = (group, key, value) => {
    if (key === 'auth') {
      setServer((s) => ({ ...s, auth: { ...s.auth, ...value } }))
      return
    }
    const setter = { server: setServer, log: setLog, monitor: setMonitor }[group]
    setter((f) => ({ ...f, [key]: value }))
  }

  const save = async () => {
    setBusy('保存')
    setError(''); setOk('')
    try {
      await api.updateConfig({
        server: { ...server, port: Number(server.port), dataDir: server.dataDir, fileRoot: server.fileRoot, auth: { enabled: server.auth.enabled, token: server.auth.token } },
        log: { ...log, maxSizeMB: Number(log.maxSizeMB), maxFiles: Number(log.maxFiles), bufferSize: Number(log.bufferSize) },
        monitor: { intervalSeconds: Number(monitor.intervalSeconds) },
        jdks,
      })
      setOk('配置已保存')
      setError('')
    } catch (e) {
      setError(`保存失败：${e.message}`)
    } finally {
      setBusy('')
    }
  }

  const discover = async () => {
    setBusy('发现')
    setError(''); setOk('')
    try {
      const found = await api.discoverJDKs()
      setDiscovered(found)
      if (found.length === 0) setError('未发现已安装的 JDK')
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy('')
    }
  }

  const alreadyExists = (id) => jdks.some((j) => j.id === id)

  const addDiscovered = (j) => {
    if (alreadyExists(j.id)) { setError(`JDK ${j.id} 已存在`); return }
    setJdks((list) => [...list, j])
    setDiscovered((list) => list.filter((x) => x.id !== j.id))
    setOk('')
  }

  const saveJdk = () => {
    if (!jdkForm.id || !jdkForm.home) { setError('JDK ID 与安装路径为必填'); return }
    setError('')
    setJdks((list) => {
      const exists = list.some((j) => j.id === jdkForm.id && jdkForm.id !== (editingJdk && editingJdk.id))
      if (exists) { setError(`JDK ${jdkForm.id} 已存在`); return list }
      if (editingJdk) {
        return list.map((j) => (j.id === editingJdk.id ? jdkForm : j))
      }
      return [...list, jdkForm]
    })
    if (!error) {
      setShowJdkForm(false)
      setJdkForm(EMPTY_JDK)
      setEditingJdk(null)
    }
  }

  const openEdit = (j) => {
    setJdkForm({ ...j })
    setEditingJdk(j)
    setShowJdkForm(true)
  }

  const openCreate = () => {
    setJdkForm(EMPTY_JDK)
    setEditingJdk(null)
    setShowJdkForm(true)
  }

  const removeJdk = (id) => {
    if (!window.confirm(`确认删除 JDK ${id}？`)) return
    setJdks((list) => list.filter((j) => j.id !== id))
  }

  const toggleDefault = (id) => {
    setJdks((list) => list.map((j) => ({ ...j, default: j.id === id })))
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-slate-800">系统配置</h2>
          <p className="text-sm text-slate-500 mt-1">
            集中管理服务端、日志、监控与 JDK 版本地址
          </p>
        </div>
        <button onClick={save} disabled={busy !== ''}
          className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-5 py-2 rounded-lg transition-colors disabled:opacity-50">
          {busy || '保存配置'}
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3">{error}</div>
      )}
      {ok && (
        <div className="bg-emerald-50 border border-emerald-200 text-emerald-700 text-sm rounded-lg px-4 py-3">{ok}</div>
      )}

      {/* 服务端 / 日志 / 监控 */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <Section title="服务端">
          <Field label="监听端口" cls={labelCls}>
            <input className={inputCls} value={server.port} onChange={(e) => setField('server', 'port', e.target.value)} />
          </Field>
          <Field label="监听地址" cls={labelCls}>
            <input className={inputCls} value={server.host} onChange={(e) => setField('server', 'host', e.target.value)} placeholder="0.0.0.0" />
          </Field>
          <Field label="数据目录" cls={labelCls}>
            <input className={inputCls} value={server.dataDir} onChange={(e) => setField('server', 'dataDir', e.target.value)} placeholder="./data" />
          </Field>
          <Field label="文件根目录" cls={labelCls}>
            <div className="flex gap-2">
              <input className={inputCls} value={server.fileRoot} onChange={(e) => setField('server', 'fileRoot', e.target.value)} placeholder="./data" readOnly />
              <button
                onClick={() => setShowFileRootPicker(true)}
                className="shrink-0 text-sm px-3 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50"
              >
                浏览
              </button>
            </div>
          </Field>
          <div className="flex items-center gap-2">
            <input type="checkbox" checked={server.auth.enabled} onChange={(e) => setField('server', 'auth', { enabled: e.target.checked })}
              className="h-4 w-4 rounded border-slate-300 text-brand-600" />
            <span className="text-sm text-slate-700">启用鉴权 (Token)</span>
          </div>
          {server.auth.enabled && (
            <Field label="Token" cls={labelCls}>
              <input className={inputCls} value={server.auth.token} onChange={(e) => setField('server', 'auth', { token: e.target.value })} />
            </Field>
          )}
        </Section>

        <Section title="日志">
          <Field label="输出目录" cls={labelCls}>
            <input className={inputCls} value={log.outputDir} onChange={(e) => setField('log', 'outputDir', e.target.value)} placeholder="./data/logs" />
          </Field>
          <Field label="单文件上限 (MB)" cls={labelCls}>
            <input className={inputCls} value={log.maxSizeMB} onChange={(e) => setField('log', 'maxSizeMB', e.target.value)} />
          </Field>
          <Field label="保留文件数" cls={labelCls}>
            <input className={inputCls} value={log.maxFiles} onChange={(e) => setField('log', 'maxFiles', e.target.value)} />
          </Field>
          <Field label="环形缓冲条数" cls={labelCls}>
            <input className={inputCls} value={log.bufferSize} onChange={(e) => setField('log', 'bufferSize', e.target.value)} />
          </Field>
        </Section>

        <Section title="监控">
          <Field label="采集周期 (秒)" cls={labelCls}>
            <input className={inputCls} value={monitor.intervalSeconds} onChange={(e) => setField('monitor', 'intervalSeconds', e.target.value)} />
          </Field>
        </Section>
      </div>

      {/* JDK 版本管理 */}
      <div className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100">
          <div>
            <h3 className="text-base font-semibold text-slate-800">JDK 版本</h3>
            <p className="text-xs text-slate-400 mt-0.5">管理系统上不同版本的 JDK 安装地址，Java 服务按需绑定</p>
          </div>
          <div className="flex gap-2">
            <button onClick={discover} disabled={busy !== ''}
              className="bg-slate-800 hover:bg-slate-900 text-white text-sm font-medium px-4 py-2 rounded-lg disabled:opacity-50">
              {busy || '自动发现'}
            </button>
            <button onClick={openCreate}
              className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-4 py-2 rounded-lg">
              + 添加 JDK
            </button>
          </div>
        </div>

        {/* 自动发现结果 */}
        {discovered.length > 0 && (
          <div className="border-b border-slate-100 px-5 py-4 space-y-2">
            <div className="text-sm font-semibold text-slate-700">发现到 {discovered.length} 个 JDK</div>
            {discovered.map((j) => (
              <div key={j.home} className="flex items-center justify-between rounded-lg border border-slate-200 px-4 py-2.5">
                <div>
                  <div className="font-medium text-slate-800">{j.version}</div>
                  <div className="text-xs text-slate-400">{j.home}</div>
                </div>
                {alreadyExists(j.id) ? (
                  <span className="text-xs text-slate-400">已添加</span>
                ) : (
                  <button onClick={() => addDiscovered(j)}
                    className="text-xs font-medium bg-emerald-500 hover:bg-emerald-600 text-white px-3 py-1.5 rounded">
                    添加
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-xs uppercase">
            <tr>
              <th className="text-left px-4 py-3">ID</th>
              <th className="text-left px-4 py-3">名称</th>
              <th className="text-left px-4 py-3">版本</th>
              <th className="text-left px-4 py-3">安装路径</th>
              <th className="text-left px-4 py-3">默认</th>
              <th className="text-right px-4 py-3">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {jdks.map((j) => (
              <tr key={j.id} className="hover:bg-slate-50">
                <td className="px-4 py-3 font-medium text-slate-800">{j.id}</td>
                <td className="px-4 py-3 text-slate-600">{j.name}</td>
                <td className="px-4 py-3 text-slate-600">{j.version}</td>
                <td className="px-4 py-3 text-slate-500 text-xs">{j.home}</td>
                <td className="px-4 py-3">
                  {j.default ? (
                    <span className="text-xs font-medium bg-emerald-100 text-emerald-700 px-2 py-0.5 rounded-full">默认</span>
                  ) : (
                    <button onClick={() => toggleDefault(j.id)} className="text-xs text-slate-400 hover:text-brand-600">设为默认</button>
                  )}
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex justify-end gap-1.5">
                    <button onClick={() => openEdit(j)}
                      className="text-xs font-medium bg-slate-200 hover:bg-slate-300 text-slate-700 px-3 py-1.5 rounded">编辑</button>
                    <button onClick={() => removeJdk(j.id)}
                      className="text-xs font-medium bg-red-100 hover:bg-red-200 text-red-700 px-3 py-1.5 rounded">删除</button>
                  </div>
                </td>
              </tr>
            ))}
            {jdks.length === 0 && (
              <tr>
                <td colSpan="6" className="text-center text-slate-400 py-10">
                  暂无 JDK 配置，点击「自动发现」或「添加 JDK」
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* JDK 表单 */}
      {showJdkForm && (
        <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-5">
          <h3 className="text-base font-semibold text-slate-800 mb-4">
            {editingJdk ? '编辑 JDK' : '添加 JDK'}
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
            <Field label="ID *" cls={labelCls}>
              <input className={inputCls} value={jdkForm.id} onChange={(e) => setJdkForm((f) => ({ ...f, id: e.target.value }))} disabled={!!editingJdk} placeholder="jdk-17" />
            </Field>
            <Field label="名称" cls={labelCls}>
              <input className={inputCls} value={jdkForm.name} onChange={(e) => setJdkForm((f) => ({ ...f, name: e.target.value }))} placeholder="OpenJDK 17" />
            </Field>
            <Field label="版本" cls={labelCls}>
              <input className={inputCls} value={jdkForm.version} onChange={(e) => setJdkForm((f) => ({ ...f, version: e.target.value }))} placeholder="17.0.10" />
            </Field>
            <Field label="安装路径 *" cls={labelCls}>
              <input className={inputCls} value={jdkForm.home} onChange={(e) => setJdkForm((f) => ({ ...f, home: e.target.value }))} placeholder="C:\\Program Files\\Java\\jdk-17" />
            </Field>
            <div className="flex items-end pb-2">
              <label className="flex items-center gap-2 text-sm text-slate-700">
                <input type="checkbox" checked={jdkForm.default} onChange={(e) => setJdkForm((f) => ({ ...f, default: e.target.checked }))}
                  className="h-4 w-4 rounded border-slate-300 text-brand-600" />
                设为默认
              </label>
            </div>
          </div>
          <div className="flex justify-end gap-3 mt-5">
            <button onClick={() => { setShowJdkForm(false); setEditingJdk(null) }}
              className="text-sm px-4 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50">
              取消
            </button>
            <button onClick={saveJdk}
              className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-5 py-2 rounded-lg">
              确定
            </button>
          </div>
        </div>
      )}

      {/* 文件根目录选择器 */}
      {showFileRootPicker && (
        <DirPicker
          value={server.fileRoot}
          onSelect={(path) => { setField('server', 'fileRoot', path); setShowFileRootPicker(false) }}
          onClose={() => setShowFileRootPicker(false)}
        />
      )}
    </div>
  )
}

function Section({ title, children }) {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-5 space-y-4">
      <h3 className="text-base font-semibold text-slate-800">{title}</h3>
      {children}
    </div>
  )
}

function Field({ label, cls, children }) {
  return (
    <div>
      <label className={cls}>{label}</label>
      {children}
    </div>
  )
}
