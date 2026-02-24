import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './context/AuthContext'
import MainLayout from './components/MainLayout'
import Login from './pages/Login'
import ProblemList from './pages/ProblemList'
import ProblemDetail from './pages/ProblemDetail'
import Submit from './pages/Submit'
import ContestList from './pages/ContestList'
import ContestDetail from './pages/ContestDetail'
import UserCenter from './pages/UserCenter'
import AdminDashboard from './pages/AdminDashboard'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, user } = useAuth()
  return isAuthenticated && user?.role === 'admin' ? <>{children}</> : <Navigate to="/" />
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />

      <Route path="/" element={<MainLayout />}>
        <Route index element={<ProblemList />} />
        <Route path="problems" element={<ProblemList />} />
        <Route path="problems/:id" element={<ProblemDetail />} />
        <Route path="problems/:id/submit" element={<Submit />} />

        <Route path="contests" element={<ContestList />} />
        <Route path="contests/:id" element={<ContestDetail />} />

        <Route path="user" element={
          <PrivateRoute>
            <UserCenter />
          </PrivateRoute>
        } />

        <Route path="admin" element={
          <AdminRoute>
            <AdminDashboard />
          </AdminRoute>
        } />
      </Route>
    </Routes>
  )
}

export default App
