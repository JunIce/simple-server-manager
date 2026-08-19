import { NavLink, Routes, Route } from 'react-router-dom'
import Dashboard from './components/Dashboard'
import Services from './components/Services'
import Files from './components/Files'
import Settings from './components/Settings'

const NAV = [
  { to: '/', label: '主机监控', icon: '📊', end: true },
  { to: '/services', label: '服务管理', icon: '⚙️' },
  { to: '/files', label: '文件目录', icon: '📁' },
  { to: '/settings', label: '配置', icon: '⚙️' },
]

export default function App() {
  return (
    <div className="flex h-full">
      {/* 侧边栏 */}
      <aside className="w-52 bg-slate-900 text-slate-300 flex flex-col shrink-0">
        <div className="px-5 py-4 border-b border-slate-800">
          <h1 className="text-white font-bold text-lg tracking-tight">
            service-manager
          </h1>
          <p className="text-xs text-slate-500 mt-0.5">服务管理系统</p>
        </div>
        <nav className="flex-1 py-3">
          {NAV.map((t) => (
            <NavLink
              key={t.to}
              to={t.to}
              end={t.end}
              className={({ isActive }) =>
                `w-full flex items-center gap-3 px-5 py-2.5 text-sm transition-colors ${
                  isActive
                    ? 'bg-slate-800 text-white border-l-2 border-brand-500'
                    : 'hover:bg-slate-800/50 hover:text-white'
                }`
              }
            >
              <span>{t.icon}</span>
              <span>{t.label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="px-5 py-3 text-[11px] text-slate-600 border-t border-slate-800">
          服务管理
        </div>
      </aside>

      {/* 主内容区：按路由渲染 */}
      <main className="flex-1 overflow-auto">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/services" element={<Services />} />
          <Route path="/files" element={<Files />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<Dashboard />} />
        </Routes>
      </main>
    </div>
  )
}
