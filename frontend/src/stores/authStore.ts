import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { api } from '../lib/api'

interface User {
  id: string
  email: string
  username: string
  is_admin: boolean
  storage_quota: number
  storage_used: number
}

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, username: string, password: string) => Promise<void>
  logout: () => void
  fetchUser: () => Promise<void>
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      isAuthenticated: false,

      login: async (email: string, password: string) => {
        const response = await api.post('/api/auth/login', { email, password })
        const { access_token, user } = response.data
        set({ token: access_token, user, isAuthenticated: true })
        api.defaults.headers.common['Authorization'] = `Bearer ${access_token}`
      },

      register: async (email: string, username: string, password: string) => {
        await api.post('/api/auth/register', { email, username, password })
      },

      logout: () => {
        set({ token: null, user: null, isAuthenticated: false })
        delete api.defaults.headers.common['Authorization']
      },

      fetchUser: async () => {
        const token = get().token
        if (!token) return
        api.defaults.headers.common['Authorization'] = `Bearer ${token}`
        try {
          const response = await api.get('/api/auth/me')
          set({ user: response.data.user })
        } catch {
          get().logout()
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ token: state.token }),
      onRehydrateStorage: () => (state) => {
        if (state?.token) {
          state.isAuthenticated = true
          state.fetchUser()
        }
      },
    }
  )
)
