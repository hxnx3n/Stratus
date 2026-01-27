import { useState } from 'react'
import { useAuthStore } from '../stores/authStore'
import { api } from '../lib/api'
import { formatBytes } from '../lib/utils'
import { Settings as SettingsIcon, User, HardDrive, Key, Check } from 'lucide-react'
import { AxiosError } from 'axios'

interface ApiError {
  error?: string
}

export default function Settings() {
  const { user } = useAuthStore()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirmPassword) {
      setMessage({ type: 'error', text: '새 비밀번호가 일치하지 않습니다.' })
      return
    }

    setLoading(true)
    setMessage(null)

    try {
      await api.put('/api/auth/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setMessage({ type: 'success', text: '비밀번호가 변경되었습니다.' })
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      setMessage({ type: 'error', text: axiosError.response?.data?.error || '비밀번호 변경 실패' })
    } finally {
      setLoading(false)
    }
  }

  const storagePercent = user ? (user.storage_used / user.storage_quota) * 100 : 0

  return (
    <div className="h-full overflow-auto bg-gray-50">
      <header className="bg-white border-b px-6 py-4">
        <div className="flex items-center gap-3">
          <SettingsIcon className="w-6 h-6 text-primary-600" />
          <h1 className="text-2xl font-bold text-gray-900">설정</h1>
        </div>
      </header>

      <div className="max-w-2xl mx-auto p-6 space-y-6">
        {message && (
          <div
            className={`p-4 rounded-lg flex items-center gap-3 ${
              message.type === 'success'
                ? 'bg-green-50 text-green-700 border border-green-200'
                : 'bg-red-50 text-red-700 border border-red-200'
            }`}
          >
            {message.type === 'success' && <Check className="w-5 h-5" />}
            <span>{message.text}</span>
          </div>
        )}

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center gap-3 mb-6">
            <HardDrive className="w-5 h-5 text-gray-500" />
            <h2 className="text-lg font-semibold">저장 공간</h2>
          </div>

          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">
                {formatBytes(user?.storage_used || 0)} 사용 중
              </span>
              <span className="text-gray-600">
                {formatBytes(user?.storage_quota || 0)} 전체
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-3">
              <div
                className={`h-3 rounded-full transition-all ${
                  storagePercent > 90 ? 'bg-red-500' : storagePercent > 70 ? 'bg-yellow-500' : 'bg-primary-600'
                }`}
                style={{ width: `${Math.min(storagePercent, 100)}%` }}
              />
            </div>
            <p className="text-sm text-gray-500">
              {storagePercent.toFixed(1)}% 사용 중
            </p>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center gap-3 mb-6">
            <User className="w-5 h-5 text-gray-500" />
            <h2 className="text-lg font-semibold">프로필</h2>
          </div>

          <div className="space-y-3">
            <div>
              <p className="text-sm font-medium text-gray-700 mb-1">이메일</p>
              <p className="px-4 py-2 border rounded-lg bg-gray-100 text-gray-700">
                {user?.email || '-'}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center gap-3 mb-6">
            <Key className="w-5 h-5 text-gray-500" />
            <h2 className="text-lg font-semibold">비밀번호 변경</h2>
          </div>

          <form onSubmit={handleChangePassword} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                현재 비밀번호
              </label>
              <input
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                required
                className="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                새 비밀번호
              </label>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                className="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                새 비밀번호 확인
              </label>
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                className="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50"
            >
              <Key className="w-4 h-4" />
              <span>비밀번호 변경</span>
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}
