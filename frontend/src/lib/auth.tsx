import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { api, type User } from './api'

type AuthCtx = {
  user: User | null
  ready: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const Ctx = createContext<AuthCtx>({
  user: null,
  ready: false,
  login: async () => {},
  logout: async () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let alive = true
    api.tryRestore().then((u) => {
      if (alive) {
        setUser(u)
        setReady(true)
      }
    })
    return () => {
      alive = false
    }
  }, [])

  const login = async (username: string, password: string) => {
    setUser(await api.login(username, password))
  }
  const logout = async () => {
    await api.logout()
    setUser(null)
  }

  return <Ctx.Provider value={{ user, ready, login, logout }}>{children}</Ctx.Provider>
}

export const useAuth = () => useContext(Ctx)

export function canManage(role?: string) {
  return role === 'manager' || role === 'superadmin'
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, ready } = useAuth()
  const loc = useLocation()
  if (!ready) {
    return (
      <div style={{ height: '100vh', display: 'grid', placeItems: 'center', color: '#8A7A66' }}>
        กำลังโหลด…
      </div>
    )
  }
  if (!user) return <Navigate to="/login" state={{ from: loc }} replace />
  return <>{children}</>
}
