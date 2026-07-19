import { useEffect, useState } from 'react'
import { NavLink, Outlet, Route, Routes } from 'react-router-dom'
import { useLiveQuery } from 'dexie-react-hooks'
import { El } from './styled'
import * as I from './icons'
import { db } from './lib/db'
import { useAuth, RequireAuth } from './lib/auth'
import Login from './pages/Login'
import Cashier from './pages/Cashier'
import Storefront from './pages/Storefront'
import Inventory from './pages/Inventory'
import Dashboard from './pages/Dashboard'

function useOnline() {
  const [online, setOnline] = useState(navigator.onLine)
  useEffect(() => {
    const on = () => setOnline(true)
    const off = () => setOnline(false)
    window.addEventListener('online', on)
    window.addEventListener('offline', off)
    return () => {
      window.removeEventListener('online', on)
      window.removeEventListener('offline', off)
    }
  }, [])
  return online
}

function Sidebar() {
  const { user, logout } = useAuth()
  const online = useOnline()
  const pending = useLiveQuery(() => db.pending.where('status').equals('pending').count(), [], 0)

  const link = (to: string, label: string, icon: JSX.Element, end?: boolean) => (
    <NavLink to={to} end={end} style={{ textDecoration: 'none' }}>
      {({ isActive }) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderRadius: 999, fontSize: 14, fontWeight: 600, background: isActive ? '#E89B2D' : 'transparent', color: isActive ? '#0F3B39' : '#CBDEDB', transition: 'background .15s' }}>
          {icon}
          <span>{label}</span>
        </div>
      )}
    </NavLink>
  )

  return (
    <nav style={{ width: 196, flexShrink: 0, background: '#0F3B39', display: 'flex', flexDirection: 'column', padding: '16px 12px', gap: 4 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '6px 4px 18px 4px' }}>
        <div style={{ width: 38, height: 38, borderRadius: 13, background: '#E89B2D', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, boxShadow: 'inset 0 -2px 0 rgba(0,0,0,.12)' }}>
          <I.Logo w={21} h={21} stroke="#0F3B39" />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ color: '#F7F1E6', fontWeight: 700, fontSize: 15, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>มาร์ทมงคลชัย</div>
          <div style={{ color: '#7FA9A4', fontSize: 11, fontFamily: "'Space Grotesk',sans-serif" }}>POS · v2.4</div>
        </div>
      </div>
      {link('/', 'แคชเชียร์', <I.CartIcon w={18} h={18} style={{ flexShrink: 0 }} />, true)}
      {link('/store', 'หน้าร้าน', <I.StoreIcon w={18} h={18} style={{ flexShrink: 0 }} />)}
      {link('/inventory', 'สต็อกสินค้า', <I.BoxIcon w={18} h={18} style={{ flexShrink: 0 }} />)}
      {link('/dashboard', 'แดชบอร์ด', <I.ChartIcon w={18} h={18} style={{ flexShrink: 0 }} />)}
      <div style={{ flex: 1 }} />

      {/* connection + sync status */}
      <div style={{ background: '#134E4A', borderRadius: 14, padding: '10px 12px', display: 'flex', flexDirection: 'column', gap: 4 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 7, fontSize: 12, color: online ? '#7FD8A0' : '#E5B7AE' }}>
          <span style={{ width: 8, height: 8, borderRadius: '50%', background: online ? '#7FD8A0' : '#C23A2B' }} />
          {online ? 'ออนไลน์' : 'ออฟไลน์'}
        </div>
        {(pending ?? 0) > 0 && (
          <div style={{ fontSize: 11, color: '#E4D3B4' }}>รอ sync {pending} บิล</div>
        )}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '10px 6px 2px 6px' }}>
        <div style={{ width: 28, height: 28, borderRadius: '50%', background: '#E89B2D', color: '#0F3B39', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 13, flexShrink: 0 }}>{user?.username?.[0]?.toUpperCase() ?? '?'}</div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ color: '#CBDEDB', fontSize: 12, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{user?.username}</div>
          <div style={{ color: '#7FA9A4', fontSize: 11 }}>{user?.role}</div>
        </div>
        <El as="button" onClick={() => logout()} title="ออกจากระบบ" s="border:none;background:#134E4A;color:#9FC3BF;width:26px;height:26px;border-radius:8px;cursor:pointer;display:flex;align-items:center;justify-content:center;flex-shrink:0;" hover="background:#1B5F5A;color:#fff;">
          <I.Close w={14} h={14} />
        </El>
      </div>
    </nav>
  )
}

function Layout() {
  return (
    <div style={{ display: 'flex', height: '100vh', fontFamily: "'IBM Plex Sans Thai',sans-serif", color: '#2B2420', background: '#F7F1E6', overflow: 'hidden' }}>
      <Sidebar />
      <Outlet />
    </div>
  )
}

function ComingSoon({ title }: { title: string }) {
  return (
    <div style={{ flex: 1, display: 'grid', placeItems: 'center', color: '#8A7A66', textAlign: 'center' }}>
      <div>
        <div style={{ fontSize: 20, fontWeight: 700, color: '#0F3B39', marginBottom: 6 }}>{title}</div>
        <div style={{ fontSize: 13 }}>กำลังพัฒนา — Phase 3b</div>
      </div>
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Cashier />} />
        <Route path="store" element={<Storefront />} />
        <Route path="inventory" element={<Inventory />} />
        <Route path="dashboard" element={<Dashboard />} />
      </Route>
    </Routes>
  )
}
