import { useState } from 'react'
import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'
import { formatBytes } from '../lib/utils'
import {
  Cloud,
  Files,
  Trash2,
  Settings,
  LogOut,
  Shield,
  HardDrive,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'

export default function Layout() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()
  const [collapsed, setCollapsed] = useState(false)

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const storageUsed = user?.storage_used || 0
  const storageQuota = user?.storage_quota || 1
  const storagePercent = Math.min((storageUsed / storageQuota) * 100, 100)

  const navItems = [
    { to: '/files', icon: Files, label: '내 파일' },
    { to: '/trash', icon: Trash2, label: '휴지통' },
    { to: '/settings', icon: Settings, label: '설정' },
  ]

  if (user?.is_admin) {
    navItems.push({ to: '/admin', icon: Shield, label: '관리자' })
  }

  return (
    <div className="flex h-screen bg-gray-100">
      <aside className={`${collapsed ? 'w-16' : 'w-64'} bg-white shadow-md flex flex-col transition-all duration-300 relative`}>
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="absolute -right-3 top-6 w-6 h-6 bg-white border rounded-full shadow flex items-center justify-center hover:bg-gray-100 z-10"
        >
          {collapsed ? (
            <ChevronRight className="w-4 h-4 text-gray-600" />
          ) : (
            <ChevronLeft className="w-4 h-4 text-gray-600" />
          )}
        </button>

        <div className="p-4 border-b">
          <div className="flex items-center gap-2">
            <Cloud className="w-8 h-8 text-primary-600 flex-shrink-0" />
            {!collapsed && <span className="text-xl font-bold text-gray-800">Stratus</span>}
          </div>
        </div>

        <nav className="flex-1 p-4">
          <ul className="space-y-2">
            {navItems.map((item) => (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  title={collapsed ? item.label : undefined}
                  className={({ isActive }) =>
                    `flex items-center gap-3 px-4 py-2 rounded-lg transition-colors ${isActive
                      ? 'bg-primary-100 text-primary-700'
                      : 'text-gray-600 hover:bg-gray-100'
                    } ${collapsed ? 'justify-center px-2' : ''}`
                  }
                >
                  <item.icon className="w-5 h-5 flex-shrink-0" />
                  {!collapsed && <span>{item.label}</span>}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <div className={`p-4 border-t ${collapsed ? 'px-2' : ''}`}>
          {!collapsed && (
            <div className="mb-4">
              <div className="flex items-center gap-2 mb-2">
                <HardDrive className="w-4 h-4 text-gray-500" />
                <span className="text-sm font-medium text-gray-700">저장공간</span>
              </div>
              <div className="w-full h-2 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all duration-300 ${storagePercent > 90
                    ? 'bg-red-500'
                    : storagePercent > 70
                      ? 'bg-yellow-500'
                      : 'bg-primary-500'
                    }`}
                  style={{ width: `${storagePercent}%` }}
                />
              </div>
              <p className="text-xs text-gray-500 mt-1">
                {formatBytes(storageUsed)} / {formatBytes(storageQuota)} 사용 중
              </p>
            </div>
          )}

          <button
            onClick={handleLogout}
            title={collapsed ? '로그아웃' : undefined}
            className={`flex items-center gap-2 w-full px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors ${collapsed ? 'justify-center px-2' : ''}`}
          >
            <LogOut className="w-5 h-5 flex-shrink-0" />
            {!collapsed && <span>로그아웃</span>}
          </button>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
