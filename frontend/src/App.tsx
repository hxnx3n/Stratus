import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './stores/authStore'
import Layout from './components/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import Files from './pages/Files'
import Trash from './pages/Trash'
import Settings from './pages/Settings'
import Admin from './pages/Admin'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, user } = useAuthStore()
  if (!isAuthenticated) return <Navigate to="/login" />
  if (!user?.is_admin) return <Navigate to="/files" />
  return <>{children}</>
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/"
        element={
          <PrivateRoute>
            <Layout />
          </PrivateRoute>
        }
      >
        <Route index element={<Navigate to="/files" replace />} />
        <Route path="files" element={<Files />} />
        <Route path="files/*" element={<Files />} />
        <Route path="trash" element={<Trash />} />
        <Route path="settings" element={<Settings />} />
        <Route
          path="admin"
          element={
            <AdminRoute>
              <Admin />
            </AdminRoute>
          }
        />
      </Route>
    </Routes>
  )
}

export default App
