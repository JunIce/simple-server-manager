import { useEffect, useState } from 'react'
import { api, connectWS } from '../api'
import { fmtMB, trimDecimals } from '../format'

function Bar({ value, color = 'bg-brand-500' }) {
  const v = Math.max(0, Math.min(100, value || 0))
  return (
    <div className="h-2 w-full rounded-full bg-slate-200 overflow-hidden">
      <div
        className={`h-full rounded-full ${color} transition-all duration-700`}
        style={{ width: `${v}%` }}
      />
    </div>
  )
}

function StatCard({ title, value, suffix, bar, barColor, sub, onClick, clickable }) {
  const cls = 'bg-white rounded-xl shadow-sm border border-slate-200 p-5' +
    (clickable ? ' cursor-pointer hover:shadow-md transition-shadow' : '')
  return (
    <div className={cls} onClick={onClick}>
      <div className="flex items-center justify-between">
        <div className="text-sm text-slate-500">{title}</div>
        {clickable && <span className="text-xs text-brand-600">查看详情 ›</span>}
      </div>
      <div className="mt-2 text-2xl font-bold text-slate-800">
        {value}
        {suffix && <span className="text-sm font-normal text-slate-500 ml-1">{suffix}</span>}
      </div>
      {bar !== undefined && <div className="mt-3"><Bar value={bar} color={barColor} /></div>}
      {sub && <div className="mt-2 text-xs text-slate-400">{sub}</div>}
    </div>
  )
}

export default function Dashboard() {
  const [stats, setStats] = useState(null)
  const [sysInfo, setSysInfo] = useState(null)
  const [services, setServices] = useState([])
  const [error, setError] = useState('')
  const [modal, setModal] = useState(null) // 'cpu' | 'mem'
  const [topProcs, setTopProcs] = useState([])
  const [loadingTop, setLoadingTop] = useState(false)
  const [killError, setKillError] = useState('')

  useEffect(() => {
    let ws = null
    let timer = null

    const loadStatic = async () => {
      try {
        const [si, svcs] = await Promise.all([
          api.getSystemInfo(),
          api.listServices(),
        ])
        setSysInfo(si)
        setServices(svcs)
      } catch (e) {
        setError(e.message)
      }
    }
    loadStatic()
    timer = setInterval(loadStatic, 5000)

    ws = connectWS('/ws/system/stats', null, {
      onMessage: (msg) => {
        if (msg.type === 'stats' && msg.data) setStats(msg.data)
      },
      onError: () => {},
    })

    return () => {
      if (ws) ws.close()
      clearInterval(timer)
    }
  }, [])

  // 打开进程详情弹窗：拉取 Top10 进程，每 3 秒刷新
  useEffect(() => {
    if (!modal) return
    let timer = null
    const load = async () => {
      setLoadingTop(true)
      try {
        setTopProcs(await api.getTopProcesses(10))
        setKillError('')
      } catch (e) {
        setKillError(e.message)
      } finally {
        setLoadingTop(false)
      }
    }
    load()
    timer = setInterval(load, 3000)
    return () => clearInterval(timer)
  }, [modal])

  const doKill = async (proc) => {
    if (!window.confirm(`确认终止进程 ${proc.name} (PID ${proc.pid})？`)) return
    try {
      await api.killProcess(proc.pid)
      setTopProcs((prev) => prev.filter((p) => p.pid !== proc.pid))
    } catch (e) {
      setKillError(`终止失败：${e.message}`)
    }
  }

  const running = services.filter((s) => s.status === 'running').length
  const failed = services.filter((s) => s.status === 'failed').length

  return (
    <div className="p-6 space-y-6">
      <div>
        <h2 className="text-xl font-bold text-slate-800">主机监控</h2>
        <p className="text-sm text-slate-500 mt-1">
          {sysInfo ? `${sysInfo.hostname} · ${sysInfo.os} ${sysInfo.arch} · ${sysInfo.cpus} 核 · 运行 ${fmtUptime(sysInfo.uptimeSec)}` : '加载中...'}
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3">
          连接失败：{error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        <StatCard
          title="CPU 使用率"
          value={stats ? stats.cpuUsage?.toFixed(1) : '--'}
          suffix="%"
          bar={stats?.cpuUsage}
          barColor="bg-blue-500"
          sub={`${stats?.cpus ?? sysInfo?.cpus ?? '--'} 核`}
          clickable
          onClick={() => setModal('cpu')}
        />
        <StatCard
          title="内存"
          value={stats ? fmtMB(stats.memoryUsed) : '--'}
          suffix={`/ ${stats ? fmtMB(stats.memoryTotal) : '--'}`}
          bar={stats?.memoryPct}
          barColor="bg-emerald-500"
          sub={stats ? `已用 ${stats.memoryPct?.toFixed(1)}%` : ''}
          clickable
          onClick={() => setModal('cpu')}
        />
        <StatCard
          title="进程数"
          value={stats?.processes ?? '--'}
          suffix="个"
          bar={stats ? (stats.processes / 1000) * 100 : undefined}
          barColor="bg-amber-500"
          sub="系统进程总数"
        />
        <StatCard
          title="服务状态"
          value={running}
          suffix={`运行 / ${services.length}`}
          bar={services.length ? (running / services.length) * 100 : 0}
          barColor="bg-violet-500"
          sub={failed > 0 ? `${failed} 个失败` : '全部正常'}
        />
      </div>

      {/* 磁盘 */}
      <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-5">
        <h3 className="text-sm font-semibold text-slate-700 mb-3">磁盘使用</h3>
        {stats?.disks?.length ? (
          <div className="space-y-3">
            {stats.disks.map((d) => (
              <div key={d.mount}>
                <div className="flex justify-between text-sm mb-1">
                  <span className="font-medium text-slate-700">{d.mount}</span>
                  <span className="text-slate-500">
                    {fmtMB(d.usedMB)} / {fmtMB(d.totalMB)} · {d.usagePct?.toFixed(1)}%
                  </span>
                </div>
                <Bar
                  value={d.usagePct}
                  color={d.usagePct > 90 ? 'bg-red-500' : d.usagePct > 75 ? 'bg-amber-500' : 'bg-emerald-500'}
                />
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-slate-400">暂无磁盘数据</p>
        )}
      </div>

      {/* 服务概览 */}
      <div className="bg-white rounded-xl shadow-sm border border-slate-200 p-5">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-slate-700">托管服务</h3>
          <span className="text-xs text-slate-400">每 5 秒刷新</span>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
          {services.map((s) => (
            <div
              key={s.id}
              className="flex items-center justify-between rounded-lg border border-slate-200 px-4 py-3"
            >
              <div>
                <div className="font-medium text-slate-800">{s.name || s.id}</div>
                <div className="text-xs text-slate-400 mt-0.5">
                  {s.type} · PID {s.pid || '-'}
                </div>
              </div>
              <StatusBadge status={s.status} />
            </div>
          ))}
          {services.length === 0 && (
            <p className="text-sm text-slate-400 col-span-full py-4 text-center">
              暂无服务，请到「服务管理」注册
            </p>
          )}
        </div>
      </div>

      {/* 进程详情弹窗 */}
      {modal && (
        <ProcessModal
          stats={stats}
          sysInfo={sysInfo}
          loading={loadingTop}
          procs={topProcs}
          error={killError}
          onKill={doKill}
          onClose={() => setModal(null)}
        />
      )}
    </div>
  )
}

function ProcessModal({ stats, sysInfo, loading, procs, error, onKill, onClose }) {
  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-2xl mx-4 max-h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
          <div>
            <h3 className="text-lg font-bold text-slate-800">CPU 使用详情</h3>
            <p className="text-sm text-slate-500 mt-0.5">
              当前使用率 {stats?.cpuUsage?.toFixed(1) ?? '--'}% ·{' '}
              {stats?.cpus ?? sysInfo?.cpus ?? '--'} 核 · 每 3 秒刷新
            </p>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-700 text-2xl leading-none">
            ×
          </button>
        </div>

        <div className="px-6 py-4 flex-1 overflow-auto">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3 mb-3">
              {error}
            </div>
          )}
          {loading && procs.length === 0 ? (
            <p className="text-center text-slate-400 py-10">加载中...</p>
          ) : (
            <table className="w-full text-sm">
              <thead className="text-xs text-slate-500 uppercase">
                <tr className="border-b border-slate-200">
                  <th className="text-left py-2 pr-3">#</th>
                  <th className="text-left py-2 pr-3">进程</th>
                  <th className="text-left py-2 pr-3">PID</th>
                  <th className="text-left py-2 pr-3">CPU</th>
                  <th className="text-left py-2 pr-3">内存</th>
                  <th className="text-right py-2">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {procs.map((proc, i) => (
                  <tr key={proc.pid}>
                    <td className="py-2 pr-3 text-slate-400">{i + 1}</td>
                    <td className="py-2 pr-3 font-medium text-slate-800 break-all">{proc.name}</td>
                    <td className="py-2 pr-3 text-slate-600">{proc.pid}</td>
                    <td className="py-2 pr-3 text-slate-700">{trimDecimals(proc.cpuUsage)}%</td>
                    <td className="py-2 pr-3 text-slate-600">{fmtMB(proc.memoryKB / 1024)}</td>
                    <td className="py-2 text-right">
                      <button
                        onClick={() => onKill(proc)}
                        className="text-xs font-medium bg-red-100 hover:bg-red-200 text-red-700 px-2.5 py-1 rounded"
                      >
                        Kill
                      </button>
                    </td>
                  </tr>
                ))}
                {procs.length === 0 && (
                  <tr><td colSpan="6" className="text-center text-slate-400 py-8">暂无进程数据</td></tr>
                )}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  )
}

export function StatusBadge({ status }) {
  const map = {
    running: 'bg-emerald-100 text-emerald-700',
    starting: 'bg-blue-100 text-blue-700',
    stopping: 'bg-amber-100 text-amber-700',
    stopped: 'bg-slate-100 text-slate-600',
    failed: 'bg-red-100 text-red-700',
    unknown: 'bg-slate-100 text-slate-500',
  }
  return (
    <span className={`text-xs font-medium px-2.5 py-1 rounded-full ${map[status] || map.unknown}`}>
      {status}
    </span>
  )
}

function fmtUptime(sec) {
  if (!sec) return '--'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (d > 0) return `${d}天 ${h}小时`
  if (h > 0) return `${h}小时 ${m}分`
  return `${m}分钟`
}
