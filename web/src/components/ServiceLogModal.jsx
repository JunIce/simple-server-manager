import { useEffect, useRef, useState } from 'react'
import { api } from '../api'

const LEVELS = ['', 'ERROR', 'WARN', 'INFO', 'DEBUG']

// 服务日志查询弹窗（检索式，无实时）
export default function ServiceLogModal({ service, onClose }) {
  const serviceId = service?.id
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState('')
  const [lines, setLines] = useState([])
  const [limit, setLimit] = useState(200)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [hasSearched, setHasSearched] = useState(false)
  const bottomRef = useRef(null)

  // 打开时默认加载最近的日志
  useEffect(() => {
    if (serviceId) doSearch(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serviceId])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [lines])

  const doSearch = async (initial = false) => {
    if (!serviceId) return
    setLoading(true)
    setError('')
    try {
      const qs = { limit }
      if (!initial || keyword) qs.keyword = keyword
      if (level) qs.level = level
      const results = await api.getLogs(serviceId, qs)
      setLines((results || []).map((r) => ({
        ...r,
        displayTime: new Date(r.timestamp).toLocaleString('zh-CN', { hour12: false }),
      })))
      setHasSearched(true)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  const levelColor = {
    ERROR: 'text-red-600',
    WARN: 'text-amber-600',
    WARNING: 'text-amber-600',
    INFO: 'text-slate-600',
    DEBUG: 'text-blue-600',
    FATAL: 'text-red-700 font-bold',
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-3xl mx-4 h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 头部 */}
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between shrink-0">
          <div>
            <h3 className="text-lg font-bold text-slate-800">日志：{service?.name || serviceId}</h3>
            <p className="text-sm text-slate-500 mt-0.5">{serviceId} · 检索式查看</p>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-700 text-2xl leading-none">×</button>
        </div>

        {/* 搜索栏 */}
        <div className="px-6 pt-4 shrink-0 flex flex-wrap items-center gap-3">
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && doSearch()}
            placeholder="关键字 (多个用空格分隔)"
            className="flex-1 min-w-40 border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            className="border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
          >
            {LEVELS.map((l) => (
              <option key={l} value={l}>{l === '' ? '全部级别' : l}</option>
            ))}
          </select>
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="border border-slate-300 rounded-lg px-3 py-2 text-sm"
          >
            <option value={100}>100 条</option>
            <option value={200}>200 条</option>
            <option value={500}>500 条</option>
            <option value={1000}>1000 条</option>
          </select>
          <button
            onClick={() => doSearch()}
            disabled={loading}
            className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-4 py-2 rounded-lg disabled:opacity-50"
          >
            {loading ? '查询中...' : '查询'}
          </button>
          <button
            onClick={() => { setKeyword(''); setLevel(''); doSearch() }}
            className="text-sm px-3 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50"
          >
            清空
          </button>
        </div>

        {error && (
          <div className="mx-6 mt-3 bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3 shrink-0">
            {error}
          </div>
        )}

        {/* 日志区 */}
        <div className="flex-1 m-6 mb-4 bg-slate-900 rounded-xl overflow-hidden flex flex-col min-h-0">
          <div className="px-4 py-2 bg-slate-800 text-slate-300 text-xs flex items-center justify-between shrink-0">
            <span>{lines.length} 条结果</span>
            <span className="text-slate-500">最多显示 {limit} 条</span>
          </div>
          <div className="flex-1 overflow-auto px-4 py-3 font-mono text-xs leading-relaxed">
            {loading && lines.length === 0 && (
              <div className="text-slate-500 text-center py-10">加载中...</div>
            )}
            {!loading && lines.length === 0 && (
              <div className="text-slate-500 text-center py-10">
                {hasSearched ? '暂无匹配日志' : '输入关键字后点击查询'}
              </div>
            )}
            {lines.map((l, i) => (
              <div key={i} className="flex gap-3 hover:bg-slate-800/50 rounded px-1">
                <span className="text-slate-500 shrink-0">{l.displayTime}</span>
                <span className={`shrink-0 w-12 ${levelColor[l.level] || levelColor.INFO}`}>
                  [{l.level || 'INFO'}]
                </span>
                <span className="text-slate-200 break-all">{l.message}</span>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        </div>
      </div>
    </div>
  )
}
