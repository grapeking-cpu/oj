import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { getUserInfo, login as apiLogin, logout as apiLogout, register as apiRegister, type RegisterParams } from '@/api'

interface User {
  id: number
  username: string
  nickname?: string
  avatar?: string
  role: string
  rating: number
  submit_count?: number
  accept_count?: number
}

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  loading: boolean  // 添加 loading 状态
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  register: (params: RegisterParams) => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

// 检查本地是否存在有效 token 的辅助函数
const hasValidToken = (): boolean => {
  // 检查 localStorage
  const localToken = localStorage.getItem('accessToken')
  if (localToken) return true
  // 检查 sessionStorage
  const sessionToken = sessionStorage.getItem('accessToken')
  if (sessionToken) return true
  // 如果使用 cookie 方式，假设 cookie 存在就算有效（由后端验证）
  return document.cookie.includes('accessToken=') || document.cookie.includes('token=')
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const location = useLocation()

  // 判断是否在登录页面或注册页面
  const isAuthPage = location.pathname === '/login' || location.pathname === '/register'

  // 使用 useCallback 避免函数引用变化导致 effect 重复执行
  const refreshUser = useCallback(async () => {
    // 登录页不调用 user info
    if (isAuthPage) {
      setLoading(false)
      return
    }

    // 无 token 时不请求 user info，直接设为未登录
    if (!hasValidToken()) {
      setUser(null)
      setLoading(false)
      return
    }

    try {
      const data = await getUserInfo()
      setUser(data as User)
      console.log('[Auth] 登录成功:', data)
    } catch (err: unknown) {
      const error = err as { statusCode?: number }
      // 只有 401 才认为是未登录，其他错误可能是网络问题
      if (error.statusCode === 401) {
        console.log('[Auth] 未登录')
        setUser(null)
        // 401 时清理本地 token
        localStorage.removeItem('accessToken')
        sessionStorage.removeItem('accessToken')
      } else {
        console.error('[Auth] 获取用户信息失败:', err)
        // 网络错误时保留当前用户状态，不设为 null
      }
    } finally {
      setLoading(false)
    }
  }, [isAuthPage])

  // 只在非登录页、且加载完成（loading=true）时执行一次
  useEffect(() => {
    // 登录/注册页面不需要检查登录态
    if (isAuthPage) {
      setLoading(false)
      return
    }
    // 页面加载时尝试恢复登录态
    refreshUser()
  }, [isAuthPage, refreshUser])

  const login = async (username: string, password: string) => {
    await apiLogin({ username, password })
    // 登录成功后直接获取用户信息（不使用 refreshUser，因为此时还在登录页，isAuthPage=true 会跳过）
    try {
      const data = await getUserInfo()
      setUser(data as User)
      console.log('[Auth] 登录成功:', data)
    } catch (err) {
      console.error('[Auth] 登录后获取用户信息失败:', err)
    }
  }

  const register = async (params: RegisterParams) => {
    await apiRegister(params)
    // 注册成功后自动登录
    await login(params.username, params.password)
  }

  const logout = async () => {
    try {
      await apiLogout()
    } finally {
      setUser(null)
      localStorage.removeItem('accessToken')
      sessionStorage.removeItem('accessToken')
    }
  }

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      loading,
      login,
      logout,
      register,
      refreshUser
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
