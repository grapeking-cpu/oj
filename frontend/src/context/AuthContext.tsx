import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { getUserInfo, login as apiLogin, logout as apiLogout } from '@/api'

interface User {
  id: number
  username: string
  nickname?: string
  avatar?: string
  role: string
  rating: number
}

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (token) {
      refreshUser()
    } else {
      setLoading(false)
    }
  }, [])

  const refreshUser = async () => {
    try {
      const data = await getUserInfo()
      setUser(data)
    } catch {
      localStorage.removeItem('token')
    } finally {
      setLoading(false)
    }
  }

  const login = async (username: string, password: string) => {
    const { token, user_id } = await apiLogin(username, password)
    localStorage.setItem('token', token)
    await refreshUser()
  }

  const logout = async () => {
    try {
      await apiLogout()
    } finally {
      localStorage.removeItem('token')
      setUser(null)
    }
  }

  if (loading) {
    return null
  }

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      login,
      logout,
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
