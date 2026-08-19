import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import MonacoEditor, { langFromName } from './MonacoEditor'

function fmtSize(size) {
  if (size == null) return '-'
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${size} B`
}

// 文件目录页面：面包屑导航 + 文件/文件夹列表 + 常用操作 + 文件编辑器
export default function Files() {
  const [root, setRoot] = useState('')
  const [path, setPath] = useState('')          // 当前相对路径（/ 分隔）
  const [entries, setEntries] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState(null)  // 正在编辑的文件
  const [clipboard, setClipboard] = useState(null) // {path, isDir}
  const [dialog, setDialog] = useState(null)    // {type:'rename'|'mkdir'|'delete', entry}
  const [busy, setBusy] = useState('')
  const fileInputRef = useRef(null)

  useEffect(() => {
    api.getConfig().then((cfg) => setRoot(cfg.server?.fileRoot || '')).catch(() => {})
  }, [])

  useEffect(() => {
    load(path)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path])

  const load = async (p) => {
    setLoading(true)
    setError('')
    try {
      setEntries(await api.listFiles(p))
    } catch (e) {
      setError(e.message)
      setEntries([])
    } finally {
      setLoading(false)
    }
  }

  const openEntry = (entry) => {
    if (entry.isDir) setPath(entry.path)
    else setEditing(entry)
  }

  const copyToClipboard = (entry) => setClipboard({ path: entry.path, isDir: entry.isDir })

  const paste = async () => {
    if (!clipboard) return
    setBusy('粘贴')
    setError('')
    try {
      const base = clipboard.path.split('/').pop()
      let name = base
      if (entries.some((e) => e.name === base)) {
        const dot = base.lastIndexOf('.')
        const stem = dot > 0 ? base.slice(0, dot) : base
        const ext = dot > 0 ? base.slice(dot) : ''
        name = `${stem} - 副本${ext}`
      }
      const dest = `${path ? path + '/' : ''}${name}`
      await api.copyFile(clipboard.path, dest)
      await load(path)
      setClipboard(null)
    } catch (err) {
      setError(`粘贴失败：${err.message}`)
    } finally {
      setBusy('')
    }
  }

  const onUploadChange = async (e) => {
    const files = Array.from(e.target.files || [])
    if (files.length === 0) return
    setBusy('上传')
    setError('')
    try {
      for (const f of files) await api.uploadFile(path, f)
      await load(path)
    } catch (err) {
      setError(`上传失败：${err.message}`)
    } finally {
      setBusy('')
      e.target.value = ''
    }
  }

  const segments = path ? path.split('/') : []
  const crumbPaths = segments.map((_, i) => segments.slice(0, i + 1).join('/'))

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-slate-800">文件目录</h2>
          <p className="text-sm text-slate-500 mt-1">
            根目录：{root || '未配置（请到「配置」设置根目录）'} · 双击文件可查看/编辑
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setDialog({ type: 'mkdir', path })}
            disabled={busy !== ''}
            className="bg-slate-800 hover:bg-slate-900 text-white text-sm font-medium px-4 py-2 rounded-lg disabled:opacity-50"
          >
            + 新建文件夹
          </button>
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={busy !== ''}
            className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-4 py-2 rounded-lg disabled:opacity-50"
          >
            {busy === '上传' ? '上传中...' : '↑ 上传文件'}
          </button>
          <input ref={fileInputRef} type="file" multiple hidden onChange={onUploadChange} />
          <button
            onClick={paste}
            disabled={!clipboard || busy !== ''}
            className={`${clipboard ? 'bg-emerald-500 hover:bg-emerald-600' : 'bg-slate-100 text-slate-400'} text-white text-sm font-medium px-4 py-2 rounded-lg disabled:cursor-not-allowed transition-colors`}
          >
            {busy === '粘贴' ? '粘贴中...' : clipboard ? `↓ 粘贴 ${clipboard.path.split('/').pop()}` : '↓ 粘贴'}
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3">{error}</div>
      )}

      {/* 面包屑导航 */}
      <nav className="flex items-center gap-1 flex-wrap text-sm">
        <button
          onClick={() => setPath('')}
          className={`px-2.5 py-1 rounded-lg transition-colors ${path === '' ? 'bg-brand-600 text-white' : 'text-brand-600 hover:bg-brand-50'}`}
        >
          📁 根目录
        </button>
        {segments.map((seg, i) => (
          <span key={i} className="flex items-center gap-1">
            <span className="text-slate-300">/</span>
            <button
              onClick={() => setPath(crumbPaths[i])}
              className={`px-2.5 py-1 rounded-lg transition-colors ${i === segments.length - 1 ? 'text-slate-800 font-medium' : 'text-brand-600 hover:bg-brand-50'}`}
            >
              {seg}
            </button>
          </span>
        ))}
      </nav>

      {/* 文件/文件夹列表 */}
      <div className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
        {path !== '' && (
          <button
            onClick={() => setPath(segments.slice(0, -1).join('/'))}
            className="w-full text-left px-4 py-2.5 text-sm text-slate-500 hover:bg-slate-50 border-b border-slate-100"
          >
            ⬆ 返回上级
          </button>
        )}
        {loading && entries.length === 0 ? (
          <div className="text-center text-slate-400 py-12">加载中...</div>
        ) : entries.length === 0 ? (
          <div className="text-center text-slate-400 py-12">空目录</div>
        ) : (
          <div className="divide-y divide-slate-100">
            {entries.map((entry) => (
              <Row
                key={entry.path}
                entry={entry}
                onOpen={openEntry}
                onCopy={copyToClipboard}
                onDownload={() => { if (!entry.isDir) api.downloadFile(entry.path) }}
                onRename={() => setDialog({ type: 'rename', entry })}
                onDelete={() => setDialog({ type: 'delete', entry })}
                clipboard={clipboard}
              />
            ))}
          </div>
        )}
        <div className="px-4 py-2 border-t border-slate-100 text-xs text-slate-400">
          提示：双击进入文件夹 / 打开文件 · 鼠标悬停行尾可操作（复制/下载/重命名/删除）
        </div>
      </div>

      {/* 文件编辑弹窗 */}
      {editing && (
        <FileEditorModal
          file={editing}
          onClose={() => setEditing(null)}
          onChanged={() => load(path)}
        />
      )}

      {/* 重命名/新建文件夹/删除 弹窗 */}
      {dialog && (
        <FileOpDialog
          dialog={dialog}
          onClose={() => setDialog(null)}
          onDone={async () => { setDialog(null); await load(path) }}
          onError={setError}
        />
      )}
    </div>
  )
}

// 列表行
function Row({ entry, onOpen, onCopy, onDownload, onRename, onDelete, clipboard }) {
  const [showActions, setShowActions] = useState(false)
  const btn = 'text-xs font-medium px-2 py-1 rounded transition-colors'
  const isCopied = clipboard && clipboard.path === entry.path
  return (
    <div
      onDoubleClick={() => onOpen(entry)}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => setShowActions(false)}
      className="flex items-center gap-3 px-4 py-2.5 hover:bg-slate-50 cursor-pointer"
    >
      <span className="text-lg shrink-0">{entry.isDir ? '📁' : '📄'}</span>
      <span className={`flex-1 truncate ${entry.isDir ? 'font-medium text-slate-800' : 'text-slate-600'}`}>
        {entry.name}
      </span>
      {!entry.isDir && <span className="text-xs text-slate-400 shrink-0">{fmtSize(entry.size)}</span>}
      {entry.isDir && <span className="text-xs text-slate-400 shrink-0">—</span>}
      <div className={`shrink-0 flex gap-1 ml-2 transition-opacity ${showActions ? 'opacity-100' : 'opacity-0'}`}>
        <button
          onClick={(e) => { e.stopPropagation(); onCopy(entry) }}
          className={`${btn} ${isCopied ? 'bg-emerald-500 text-white' : 'bg-slate-200 hover:bg-slate-300 text-slate-700'}`}
        >
          {isCopied ? '已复制' : '复制'}
        </button>
        {!entry.isDir && (
          <button onClick={(e) => { e.stopPropagation(); onDownload() }} className={`${btn} bg-blue-100 hover:bg-blue-200 text-blue-700`}>
            下载
          </button>
        )}
        <button onClick={(e) => { e.stopPropagation(); onRename() }} className={`${btn} bg-slate-100 hover:bg-slate-200 text-slate-600`}>
          重命名
        </button>
        <button onClick={(e) => { e.stopPropagation(); onDelete() }} className={`${btn} bg-red-100 hover:bg-red-200 text-red-700`}>
          删除
        </button>
      </div>
    </div>
  )
}

// 重命名 / 新建文件夹 弹窗
function FileOpDialog({ dialog, onClose, onDone, onError }) {
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)

  const confirm = async () => {
    const v = name.trim()
    if (!v) return
    setBusy(true)
    onError('')
    try {
      if (dialog.type === 'mkdir') {
        await api.mkdirFile(dialog.path || '', v)
      } else if (dialog.type === 'rename') {
        await api.renameFile(dialog.entry.path, v)
      } else if (dialog.type === 'delete') {
        await api.deleteFile(dialog.entry.path)
      }
      await onDone()
    } catch (e) {
      onError(`${dialog.type === 'delete' ? '删除' : dialog.type === 'rename' ? '重命名' : '新建'}失败：${e.message}`)
    } finally {
      setBusy(false)
    }
  }

  const isDelete = dialog.type === 'delete'

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-sm mx-4 p-6"
        onClick={(e) => e.stopPropagation()}
      >
        {isDelete ? (
          <>
            <h3 className="text-base font-bold text-slate-800 mb-3">确认删除</h3>
            <p className="text-sm text-slate-600 mb-2">
              确定删除 <span className="font-medium text-red-600">{dialog.entry.name}</span> 吗？
              {dialog.entry.isDir && <span className="text-slate-400">（将递归删除目录内所有文件）</span>}
            </p>
          </>
        ) : (
          <>
            <h3 className="text-base font-bold text-slate-800 mb-3">
              {dialog.type === 'rename' ? '重命名' : '新建文件夹'}
            </h3>
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && confirm()}
              placeholder={dialog.type === 'rename' ? '输入新名称' : '输入文件夹名称'}
              className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500 mb-2"
            />
            {dialog.type === 'rename' && (
              <p className="text-xs text-slate-400 mb-2">当前名称：{dialog.entry.name}</p>
            )}
          </>
        )}
        <div className="flex justify-end gap-3 mt-4">
          <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50">
            取消
          </button>
          <button
            onClick={confirm}
            disabled={busy}
            className={`text-white text-sm font-medium px-4 py-2 rounded-lg disabled:opacity-50 ${isDelete ? 'bg-red-500 hover:bg-red-600' : 'bg-brand-600 hover:bg-brand-700'}`}
          >
            {busy ? '处理中...' : isDelete ? '删除' : '确定'}
          </button>
        </div>
      </div>
    </div>
  )
}

// 文件编辑器弹窗：加载内容 → 编辑 → 保存
function FileEditorModal({ file, onClose, onChanged }) {
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [ok, setOk] = useState('')

  useEffect(() => {
    api.readFile(file.path)
      .then(setContent)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [file.path])

  const save = async () => {
    setSaving(true)
    setError(''); setOk('')
    try {
      await api.writeFile(file.path, content)
      setOk('已保存')
      if (onChanged) onChanged()
    } catch (e) {
      setError(`保存失败：${e.message}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white rounded-2xl shadow-xl w-full max-w-4xl mx-4 h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between shrink-0">
          <div>
            <h3 className="text-lg font-bold text-slate-800">{file.name}</h3>
            <p className="text-sm text-slate-500 mt-0.5">{file.path}</p>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-700 text-2xl leading-none">×</button>
        </div>

        {error && (
          <div className="bg-red-50 border-b border-red-200 text-red-700 text-sm px-6 py-3 shrink-0">{error}</div>
        )}
        {ok && (
          <div className="bg-emerald-50 border-b border-emerald-200 text-emerald-700 text-sm px-6 py-3 shrink-0">{ok}</div>
        )}

        <div className="flex-1 overflow-hidden m-4 bg-slate-900 rounded-xl">
          {loading ? (
            <div className="flex items-center justify-center h-full text-slate-400 text-sm">加载中...</div>
          ) : (
            <MonacoEditor
              value={content}
              onChange={setContent}
              language={langFromName(file.name)}
            />
          )}
        </div>

        <div className="px-6 py-4 border-t border-slate-200 flex items-center justify-between shrink-0">
          <span className="text-xs text-slate-400">{content.length} 字符 · 保存将覆盖文件</span>
          <div className="flex gap-3">
            <button onClick={onClose} className="text-sm px-4 py-2 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50">
              关闭
            </button>
            <button onClick={save} disabled={saving}
              className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-5 py-2 rounded-lg disabled:opacity-50">
              {saving ? '保存中...' : '保存'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
