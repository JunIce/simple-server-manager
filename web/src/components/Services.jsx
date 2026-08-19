import { useEffect, useState } from 'react'
import { api } from '../api'
import { StatusBadge } from './Dashboard'
import { fmtMB, trimDecimals } from '../format'
import ServiceLogModal from './ServiceLogModal'
import DirPicker from './DirPicker'

const EMPTY_FORM = {
  id: '',
  name: '',
  type: 'generic',
  command: '',
  args: '',
  workingDir: '',
  environments: '',
  autoRestart: true,
  java: { jdk: '', jarFile: '', mainClass: '', jvmOptions: '', memoryLimit: '' },
  mysql: { binaryPath: '', configFile: '', dataDir: '', port: 3306 },
  redis: { configFile: '', port: 6379, bindAddress: '127.0.0.1' },
}

export default function Services() {
  const [services, setServices] = useState([])
  const [metrics, setMetrics] = useState({})
  const [jdks, setJdks] = useState([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState(EMPTY_FORM)
  const [logService, setLogService] = useState(null)

  const load = async () => {
    try {
      const list = await api.listServices()
      setServices(list)
      // 批量拉取指标
      const m = {}
      for (const s of list) {
        try {
          const r = await api.getMetrics(s.id)
          if (r?.metrics) m[s.id] = r.metrics
        } catch (e) { /* ignore */ }
      }
      setMetrics(m)
      setError('')
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => {
    load()
    const t = setInterval(load, 5000)
    return () => clearInterval(t)
  }, [])

  // 加载 JDK 列表用于 Java 服务绑定选择
  useEffect(() => {
    api.getConfig().then((cfg) => setJdks(cfg.jdks || [])).catch(() => {})
  }, [])

  const act = async (label, fn) => {
    setBusy(label)
    setError('')
    try {
      await fn()
      await load()
    } catch (e) {
      setError(`${label} 失败：${e.message}`)
    } finally {
      setBusy('')
    }
  }

  const openCreate = () => {
    setForm(EMPTY_FORM)
    setEditing(false)
    setShowForm(true)
  }

  const openEdit = (svc) => {
    const c = svc.config
    setForm({
      id: svc.id,
      name: c.name,
      type: c.type,
      command: c.command,
      args: (c.args || []).join('\n'),
      workingDir: c.workingDir,
      environments: (c.environments || []).join(','),
      autoRestart: c.autoRestart,
      java: c.java || EMPTY_FORM.java,
      mysql: c.mysql || EMPTY_FORM.mysql,
      redis: c.redis || EMPTY_FORM.redis,
    })
    setEditing(true)
    setShowForm(true)
  }

  const submit = async () => {
    // 参数：每行一个参数，支持 xxx=xxx 形式
    const args = form.args.split(/\r?\n/).map((s) => s.trim()).filter(Boolean)
    const environments = form.environments.split(',').map((s) => s.trim()).filter(Boolean)
    const svc = {
      id: form.id,
      name: form.name,
      type: form.type,
      command: form.command,
      args,
      workingDir: form.workingDir,
      environments,
      autoRestart: form.autoRestart,
      restartPolicy: { maxRetries: 3, backoffType: 'exponential', backoffDelay: '2s', maxBackoff: '60s' },
      healthCheck: { httpEndpoint: '', intervalSec: 5, maxFailures: 3, timeoutMS: 3000 },
      java: form.type === 'java' ? {
        jdk: form.java.jdk,
        jarFile: form.java.jarFile,
        mainClass: form.java.mainClass,
        jvmOptions: form.java.jvmOptions ? form.java.jvmOptions.split(/\s+/).filter(Boolean) : [],
        memoryLimit: form.java.memoryLimit,
      } : undefined,
      mysql: form.type === 'mysql' ? {
        binaryPath: form.mysql.binaryPath,
        configFile: form.mysql.configFile,
        dataDir: form.mysql.dataDir,
        port: Number(form.mysql.port),
      } : undefined,
      redis: form.type === 'redis' ? {
        configFile: form.redis.configFile,
        port: Number(form.redis.port),
        bindAddress: form.redis.bindAddress,
      } : undefined,
    }
    if (editing) {
      await act('更新', () => api.updateService(form.id, svc))
    } else {
      await act('注册', () => api.createService(svc))
    }
    if (!error) {
      setShowForm(false)
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-slate-800">服务管理</h2>
          <p className="text-sm text-slate-500 mt-1">
            {services.length} 个服务 · 每 5 秒刷新
          </p>
        </div>
        <button
          onClick={openCreate}
          className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          + 注册服务
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3">
          {error}
        </div>
      )}

      {showForm && (
        <ServiceForm
          form={form}
          setForm={setForm}
          jdks={jdks}
          editing={editing}
          busy={busy}
          onCancel={() => setShowForm(false)}
          onSubmit={submit}
        />
      )}

      <div className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-xs uppercase">
            <tr>
              <th className="text-left px-4 py-3">服务</th>
              <th className="text-left px-4 py-3">类型</th>
              <th className="text-left px-4 py-3">状态</th>
              <th className="text-left px-4 py-3">PID</th>
              <th className="text-left px-4 py-3">CPU</th>
              <th className="text-left px-4 py-3">内存</th>
              <th className="text-right px-4 py-3">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {services.map((s) => {
              const m = metrics[s.id]
              return (
                <tr key={s.id} className="hover:bg-slate-50">
                  <td className="px-4 py-3">
                    <div className="font-medium text-slate-800">{s.name || s.id}</div>
                    <div className="text-xs text-slate-400">{s.id}</div>
                  </td>
                  <td className="px-4 py-3 text-slate-600">{s.type}</td>
                  <td className="px-4 py-3"><StatusBadge status={s.status} /></td>
                  <td className="px-4 py-3 text-slate-600">{s.pid || '-'}</td>
                  <td className="px-4 py-3 text-slate-600">
                    {m ? `${trimDecimals(m.cpuUsage)}%` : '-'}
                  </td>
                  <td className="px-4 py-3 text-slate-600">
                    {m ? `${fmtMB(m.memoryKB / 1024)}` : '-'}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1.5">
                      <OpBtn label="日志" color="ink" onClick={() => setLogService(s)} />
                      <OpBtn label="启动" color="emerald" disabled={s.status === 'running' || s.status === 'starting'}
                        onClick={() => act('启动', () => api.startService(s.id))} />
                      <OpBtn label="停止" color="amber" disabled={s.status !== 'running' && s.status !== 'starting'}
                        onClick={() => act('停止', () => api.stopService(s.id))} />
                      <OpBtn label="重启" color="blue" disabled={s.status !== 'running'}
                        onClick={() => act('重启', () => api.restartService(s.id))} />
                      <OpBtn label="编辑" color="slate" onClick={() => openEdit(s)} />
                      <OpBtn label="删除" color="red"
                        onClick={() => {
                          if (window.confirm(`确认注销服务 ${s.id}？`)) {
                            act('删除', () => api.deleteService(s.id))
                          }
                        }} />
                    </div>
                  </td>
                </tr>
              )
            })}
            {services.length === 0 && (
              <tr>
                <td colSpan="7" className="text-center text-slate-400 py-10">
                  暂无服务，点击右上角「注册服务」
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {logService && (
        <ServiceLogModal service={logService} onClose={() => setLogService(null)} />
      )}
    </div>
  )
}

function OpBtn({ label, color, disabled, onClick }) {
  const map = {
    emerald: 'bg-emerald-500 hover:bg-emerald-600',
    amber: 'bg-amber-500 hover:bg-amber-600',
    blue: 'bg-blue-500 hover:bg-blue-600',
    slate: 'bg-slate-200 hover:bg-slate-300 text-slate-700',
    ink: 'bg-slate-700 hover:bg-slate-800',
    red: 'bg-red-100 hover:bg-red-200 text-red-700',
  }
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={`text-xs font-medium px-2.5 py-1 rounded text-white transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${map[color]}`}
    >
      {label}
    </button>
  )
}

function ServiceForm({ form, setForm, editing, busy, jdks, onCancel, onSubmit }) {
  const [showDirPicker, setShowDirPicker] = useState(false)

  const set = (k) => (e) => {
    const v = e.target.value
    if (k.startsWith('java.')) {
      const [group, field] = k.split('.')
      setForm((f) => ({ ...f, [group]: { ...f[group], [field]: v } }))
    } else if (k === 'autoRestart') {
      setForm((f) => ({ ...f, autoRestart: e.target.checked }))
    } else {
      setForm((f) => ({ ...f, [k]: v }))
    }
  }

  const inputCls =
    'w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'
  const labelCls = 'block text-xs font-medium text-slate-500 mb-1'

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onCancel}>
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-xl mx-4 max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between shrink-0">
          <h3 className="text-lg font-bold text-slate-800">
            {editing ? `编辑服务 ${form.id}` : '注册服务'}
          </h3>
          <button onClick={onCancel} className="text-slate-400 hover:text-slate-700 text-2xl leading-none">×</button>
        </div>

        <div className="px-6 py-4 flex-1 overflow-auto space-y-4">
          {/* 基础信息 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field label={editing ? '服务 ID' : '服务 ID *'} cls={labelCls}>
              <input className={inputCls} value={form.id} onChange={set('id')} disabled={editing} placeholder="my-java-app" />
            </Field>
            <Field label="名称" cls={labelCls}>
              <input className={inputCls} value={form.name} onChange={set('name')} placeholder="My Java Application" />
            </Field>
          </div>

          {/* 类型 + JDK */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Field label="类型" cls={labelCls}>
              <select className={inputCls} value={form.type} onChange={set('type')}>
                <option value="java">Java</option>
                <option value="generic">Generic</option>
              </select>
            </Field>
            {form.type === 'java' && (
              <Field label="JDK 版本" cls={labelCls}>
                <select className={inputCls} value={form.java.jdk} onChange={set('java.jdk')}>
                  <option value="">默认 (未指定)</option>
                  {jdks.map((j) => (
                    <option key={j.id} value={j.id}>
                      {j.id}{j.default ? ' (默认)' : ''} — {j.version}
                    </option>
                  ))}
                </select>
              </Field>
            )}
          </div>

          {/* 根目录 */}
          <Field label="根目录 (工作目录)" cls={labelCls}>
            <div className="flex gap-2">
              <input
                className={inputCls}
                value={form.workingDir}
                onChange={set('workingDir')}
                placeholder="C:\\app"
                readOnly
              />
              <button
                onClick={() => setShowDirPicker(true)}
                className="shrink-0 text-sm px-3 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50"
              >
                浏览
              </button>
            </div>
          </Field>

          {/* 执行命令 */}
          <Field label="执行命令" cls={labelCls}>
            <input
              className={inputCls}
              value={form.command}
              onChange={set('command')}
              placeholder={form.type === 'java' ? '可留空（使用所选 JDK 的 java）' : '如 ping / powershell'}
            />
          </Field>

          {/* Java 专属：Jar/主类 */}
          {form.type === 'java' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Field label="Jar 包路径" cls={labelCls}>
                <input className={inputCls} value={form.java.jarFile} onChange={set('java.jarFile')} placeholder="app.jar" />
              </Field>
              <Field label="主类" cls={labelCls}>
                <input className={inputCls} value={form.java.mainClass} onChange={set('java.mainClass')} placeholder="com.example.Main" />
              </Field>
            </div>
          )}

          {/* 参数：key=value 每行一个 */}
          <Field label="参数 (每行一个，支持 xxx=xxx 形式)" cls={labelCls}>
            <textarea
              className={`${inputCls} font-mono`}
              value={form.args}
              onChange={set('args')}
              rows={4}
              placeholder={'--port=8080\n-Dname=value\n--config=app.yaml'}
            />
          </Field>

          {/* Java JVM 参数 */}
          {form.type === 'java' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Field label="JVM 参数" cls={labelCls}>
                <input className={inputCls} value={form.java.jvmOptions} onChange={set('java.jvmOptions')} placeholder="-Xms256m -Xmx512m" />
              </Field>
              <Field label="内存限制" cls={labelCls}>
                <input className={inputCls} value={form.java.memoryLimit} onChange={set('java.memoryLimit')} placeholder="512m" />
              </Field>
            </div>
          )}

          {/* 环境标记 + 自动重启 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-end">
            <Field label="环境标记 (逗号分隔)" cls={labelCls}>
              <input className={inputCls} value={form.environments} onChange={set('environments')} placeholder="dev,test,prod" />
            </Field>
            <label className="flex items-center gap-2 text-sm text-slate-700 pb-2">
              <input type="checkbox" checked={form.autoRestart} onChange={set('autoRestart')}
                className="h-4 w-4 rounded border-slate-300 text-brand-600 focus:ring-brand-500" />
              自动重启
            </label>
          </div>
        </div>

        <div className="px-6 py-4 border-t border-slate-200 flex justify-end gap-3 shrink-0">
          <button onClick={onCancel}
            className="text-sm px-4 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50">
            取消
          </button>
          <button onClick={onSubmit} disabled={busy !== ''}
            className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-5 py-2 rounded-lg disabled:opacity-50">
            {busy || (editing ? '保存' : '注册')}
          </button>
        </div>

        {showDirPicker && (
          <DirPicker
            value={form.workingDir}
            onSelect={(path) => {
              setForm((f) => ({ ...f, workingDir: path }))
              setShowDirPicker(false)
            }}
            onClose={() => setShowDirPicker(false)}
          />
        )}
      </div>
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

