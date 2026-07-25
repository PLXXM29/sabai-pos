import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { api, type User } from './api'

type AuthCtx = {
  user: User | null
  ready: boolean
  login: (username: string, password: string) => Promise<User>
  logout: () => Promise<void>
}

const Ctx = createContext<AuthCtx>({
  user: null,
  ready: false,
  login: async () => {
    throw new Error('AuthProvider is missing')
  },
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
    const u = await api.login(username, password)
    setUser(u)
    return u
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

/** Routes the server will refuse to a cashier. */
const MANAGER_ROUTES = ['/dashboard']

/**
 * Where to send someone after they sign in.
 *
 * Returning to the page that bounced you to the login screen is the right
 * default, but only if you are allowed to be there. Signing out of the
 * dashboard and back in as a cashier used to land on the dashboard — a page
 * that immediately tells you that you cannot see it. The first screen after
 * signing in should be one that works.
 */
export function landingFor(role: string | undefined, wanted?: string) {
  if (!wanted || wanted === '/login') return '/'
  if (!canManage(role) && MANAGER_ROUTES.some((p) => wanted.startsWith(p))) return '/'
  return wanted
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
