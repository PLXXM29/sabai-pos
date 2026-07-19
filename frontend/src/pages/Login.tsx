import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { El } from '../styled'
import { Logo } from '../icons'
import { useAuth } from '../lib/auth'

export default function Login() {
  const { login } = useAuth()
  const nav = useNavigate()
  const loc = useLocation() as { state?: { from?: { pathname: string } } }
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr('')
    setBusy(true)
    try {
      await login(username.trim(), password)
      nav(loc.state?.from?.pathname ?? '/', { replace: true })
    } catch {
      setErr('ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง')
    } finally {
      setBusy(false)
    }
  }

  const inp =
    'width:100%;border:1.5px solid #E0D3BD;border-radius:12px;padding:11px 14px;font-size:15px;outline:none;background:#fff;color:#2B2420;'

  return (
    <div style={{ height: '100vh', display: 'grid', placeItems: 'center', background: '#F7F1E6', fontFamily: "'IBM Plex Sans Thai',sans-serif" }}>
      <form onSubmit={submit} style={{ width: 340, background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 20, padding: '28px 26px', boxShadow: '0 20px 50px rgba(15,59,57,.12)' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10, marginBottom: 18 }}>
          <div style={{ width: 52, height: 52, borderRadius: 16, background: '#E89B2D', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: 'inset 0 -2px 0 rgba(0,0,0,.12)' }}>
            <Logo w={28} h={28} stroke="#0F3B39" />
          </div>
          <div style={{ fontWeight: 700, fontSize: 20, color: '#0F3B39' }}>MiniMart POS</div>
          <div style={{ fontSize: 13, color: '#8A7A66' }}>เข้าสู่ระบบเพื่อเริ่มขาย</div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <El as="input" value={username} onChange={(e: any) => setUsername(e.target.value)} placeholder="ชื่อผู้ใช้" s={inp} focus="border-color:#C2571F;" autoFocus />
          <El as="input" value={password} onChange={(e: any) => setPassword(e.target.value)} type="password" placeholder="รหัสผ่าน" s={inp} focus="border-color:#C2571F;" />
          {err && <div style={{ color: '#C23A2B', fontSize: 13, fontWeight: 600 }}>{err}</div>}
          <button type="submit" disabled={busy} style={{ marginTop: 4, border: 'none', background: busy ? '#D8C9B0' : '#C2571F', color: '#fff', borderRadius: 999, padding: '12px 0', fontSize: 15, fontWeight: 700, cursor: busy ? 'default' : 'pointer', boxShadow: '0 3px 0 #9C4517', fontFamily: 'inherit' }}>
            {busy ? 'กำลังเข้าสู่ระบบ…' : 'เข้าสู่ระบบ'}
          </button>
        </div>
        <div style={{ marginTop: 16, fontSize: 11.5, color: '#B5A88F', textAlign: 'center', lineHeight: 1.6 }}>
          ทดลอง: owner / manager / cashier<br />รหัส: [role]1234
        </div>
      </form>
    </div>
  )
}
