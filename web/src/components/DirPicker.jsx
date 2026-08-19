import { useEffect, useState } from 'react'
import { api } from '../api'

// 服务端目录浏览选择器
export default function DirPicker({ value, onSelect, onClose }) {
  const [currentPath, setCurrentPath] = useState(value || '')
  const [dirs, setDirs] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // value 变化时同步当前路径
  useEffect(() => {
    setCurrentPath(value || '')
  }, [value])

  useEffect(() => {
    load(currentPath)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPath])

  const load = async (path) => {
    setLoading(true)
    setError('')
    try {
      setDirs(await api.listDirs(path))
    } catch (e) {
      setError(e.message)
      setDirs([])
    } finally {
      setLoading(false)
    }
  }

  const goUp = () => {
    if (!currentPath) return
    setDirs([])
    const idx = currentPath.replace(/[\\/]+$/, '').lastIndexOf('\\')
    const idx2 = currentPath.replace(/[\\/]+$/, '').lastIndexOf('/')
    const cut = Math.max(idx, idx2)
    if (cut <= 0) {
      setCurrentPath('') // 回到盘符列表
    } else {
      setCurrentPath(currentPath.slice(0, cut))
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-lg mx-4 flex flex-col max-h-[70vh]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-slate-200 flex items-center justify-between shrink-0">
          <h3 className="text-base font-bold text-slate-800">选择目录</h3>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-700 text-2xl leading-none">×</button>
        </div>

        {/* 当前路径栏 */}
        <div className="px-5 py-3 shrink-0 flex items-center gap-2">
          <button
            onClick={goUp}
            disabled={!currentPath}
            className="text-sm px-3 py-1.5 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50 disabled:opacity-40"
          >
            ↑ 上级
          </button>
          <input
            value={currentPath}
            onChange={(e) => setCurrentPath(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && load(currentPath)}
            placeholder="输入路径后回车"
            className="flex-1 border border-slate-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>

        {error && (
          <div className="mx-5 mb-2 bg-red-50 border border-red-200 text-red-700 text-xs rounded-lg px-3 py-2 shrink-0">
            {error}
          </div>
        )}

        {/* 目录列表 */}
        <div className="flex-1 overflow-auto px-3 pb-3 min-h-40">
          {loading && dirs.length === 0 ? (
            <div className="text-center text-slate-400 py-8 text-sm">加载中...</div>
          ) : dirs.length === 0 ? (
            <div className="text-center text-slate-400 py-8 text-sm">无子目录</div>
          ) : (
            <div className="border border-slate-200 rounded-lg divide-y divide-slate-100">
              {dirs.map((d) => (
                <button
                  key={d.path}
                  onClick={() => setCurrentPath(d.path)}
                  className="w-full text-left px-3 py-2 text-sm hover:bg-slate-50 flex items-center gap-2"
                >
                  <span className="text-slate-400">📁</span>
                  <span className="text-slate-700 truncate flex-1">{d.name}</span>
                  {d.path === currentPath && <span className="text-brand-600 text-xs">当前</span>}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* 底部：选择当前目录 */}
        <div className="px-5 py-3 border-t border-slate-200 flex items-center justify-between shrink-0">
          <span className="text-xs text-slate-400 truncate flex-1 mr-3">{currentPath || '请选择或输入目录'}</span>
          <button
            onClick={() => currentPath && onSelect(currentPath)}
            disabled={!currentPath}
            className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-5 py-2 rounded-lg disabled:opacity-40"
          >
            选择此目录
          </button>
        </div>
      </div>
    </div>
  )
}
