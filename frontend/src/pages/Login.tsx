import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Check, ChevronRight, Logo } from '../icons'
import { useAuth } from '../lib/auth'
import { useMeta } from '../lib/meta'
import { api, type DemoAccount } from '../lib/api'

/**
 * Sign-in, and — when the server says it is a demo — the product's front door.
 *
 * A showcase link that lands on a bare username box asks the visitor to guess
 * two things: what the software does, and how to get inside. So in demo mode
 * this screen answers both up front and offers one tap per permission level.
 * The accounts come from the server (/api/v1/meta), not from constants here, so
 * a real deployment of this same bundle shows nothing but the login form.
 */

const AVATAR_TINT: Record<string, string> = {
  superadmin: '#E89B2D',
  manager: '#8FCBB6',
  cashier: '#F0BE8A',
}

export default function Login() {
  const { login } = useAuth()
  const nav = useNavigate()
  const loc = useLocation() as { state?: { from?: { pathname: string } } }
  const meta = useMeta()

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState('')

  const enter = async (user: string, pass: string, tag: string) => {
    setErr('')
    setBusy(tag)
    try {
      await login(user.trim(), pass)
      nav(loc.state?.from?.pathname ?? '/', { replace: true })
    } catch {
      setErr('ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง')
      setBusy('')
    }
  }

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    void enter(username, password, 'form')
  }

  const demo = meta?.demo === true
  const accounts = meta?.accounts ?? []

  return (
    <div className="signin">
      <div className={`signin__panel${demo ? '' : ' signin__panel--solo'}`}>
        {demo && <Pitch />}

        <div className="signin__form">
          {demo ? (
            <>
              <span className="signin__badge">
                <span className="signin__dot" />
                ระบบสาธิต — ข้อมูลตัวอย่าง
              </span>
              <h1 className="signin__h">เลือกบทบาทเพื่อเข้าใช้งาน</h1>
              <p className="signin__sub">
                กดปุ่มเดียวเข้าได้เลย ไม่ต้องกรอกอะไร แต่ละบทบาทเห็นและทำได้ไม่เท่ากัน
                เพราะสิทธิ์ถูกตรวจที่ฝั่งเซิร์ฟเวอร์จริง
              </p>

              {accounts.map((a) => (
                <RoleButton
                  key={a.username}
                  account={a}
                  busy={busy}
                  onPick={() => void enter(a.username, a.password, a.username)}
                />
              ))}

              <details className="signin__manual">
                <summary>เข้าสู่ระบบด้วยชื่อผู้ใช้และรหัสผ่าน</summary>
                <Form
                  {...{ username, setUsername, password, setPassword, submit }}
                  busy={busy === 'form'}
                  err={err}
                />
              </details>

              <DemoFooter resetEvery={meta?.reset_every} version={meta?.version} />
            </>
          ) : (
            <>
              <div className="signin__brand" style={{ marginBottom: 6 }}>
                <div className="signin__mark">
                  <Logo w={26} h={26} stroke="#0F3B39" />
                </div>
                <div>
                  <div className="signin__wordmark" style={{ color: '#0F3B39' }}>
                    Sabai POS
                  </div>
                  <div className="signin__tagline" style={{ color: '#8A7A66' }}>
                    เข้าสู่ระบบเพื่อเริ่มขาย
                  </div>
                </div>
              </div>
              <Form
                {...{ username, setUsername, password, setPassword, submit }}
                busy={busy === 'form'}
                err={err}
                autoFocus
              />
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function RoleButton({
  account,
  busy,
  onPick,
}: {
  account: DemoAccount
  busy: string
  onPick: () => void
}) {
  const working = busy === account.username
  return (
    <button className="role" onClick={onPick} disabled={busy !== ''} type="button">
      <span
        className="role__avatar"
        style={{ background: AVATAR_TINT[account.role] ?? '#E0D3BD' }}
        aria-hidden="true"
      >
        {/* Latin initial from the username: a lone Thai vowel like "เ" is not a
            readable monogram, and the username is what gets typed anyway. */}
        {account.username.slice(0, 1).toUpperCase()}
      </span>
      <span className="role__body">
        <span className="role__label">
          {account.label}
          <span className="role__role">{account.role}</span>
        </span>
        <span className="role__desc">{account.description}</span>
      </span>
      <span className="role__go" aria-hidden="true">
        {working ? '…' : <ChevronRight w={18} h={18} />}
      </span>
    </button>
  )
}

function Form({
  username,
  setUsername,
  password,
  setPassword,
  submit,
  busy,
  err,
  autoFocus,
}: {
  username: string
  setUsername: (v: string) => void
  password: string
  setPassword: (v: string) => void
  submit: (e: React.FormEvent) => void
  busy: boolean
  err: string
  autoFocus?: boolean
}) {
  return (
    <form onSubmit={submit} className="signin__fields">
      <input
        className="signin__input"
        value={username}
        onChange={(e) => setUsername(e.target.value)}
        placeholder="ชื่อผู้ใช้"
        autoComplete="username"
        autoFocus={autoFocus}
      />
      <input
        className="signin__input"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        type="password"
        placeholder="รหัสผ่าน"
        autoComplete="current-password"
      />
      {err && <div className="signin__err">{err}</div>}
      <button className="signin__submit" type="submit" disabled={busy}>
        {busy ? 'กำลังเข้าสู่ระบบ…' : 'เข้าสู่ระบบ'}
      </button>
    </form>
  )
}

function Pitch() {
  return (
    <div className="signin__pitch">
      <div className="signin__brand">
        <div className="signin__mark">
          <Logo w={26} h={26} stroke="#0F3B39" />
        </div>
        <div>
          <div className="signin__wordmark">Sabai POS</div>
          <div className="signin__tagline">ระบบขายหน้าร้าน · offline-first</div>
        </div>
      </div>

      <p className="signin__lede">
        ระบบขายหน้าร้านสำหรับร้านโชห่วยและมินิมาร์ท ขายต่อได้แม้เน็ตหลุด
        ตัดสต็อกและปิดยอดให้ตรงเสมอ
      </p>

      <ul className="signin__features">
        <Feature title="ขายได้ตอนเน็ตหลุด">
          บิลถูกเก็บในเครื่องแล้วซิงก์ให้เองเมื่อเน็ตกลับมา ไม่มีบิลซ้ำ ไม่มีบิลหาย
        </Feature>
        <Feature title="สต็อกเป็นบัญชีเดินสะพัด">
          ทุกการเคลื่อนไหวถูกบันทึกและแก้ย้อนหลังไม่ได้ ยอดคงเหลือจึงตรวจสอบได้
        </Feature>
        <Feature title="รับเงินโอนแบบยืนยันเอง">
          สร้าง QR พร้อมเพย์ที่หน้าจอ และตรวจเงินเข้าอัตโนมัติได้ถ้าต่อ LINE ไว้
        </Feature>
        <Feature title="สิทธิ์แยกตามบทบาท">
          แคชเชียร์ขายได้แต่แก้ราคาและดูกำไรไม่ได้ ตรวจที่เซิร์ฟเวอร์ ไม่ใช่แค่ซ่อนปุ่ม
        </Feature>
        <Feature title="ใบเสร็จพิมพ์ได้จริง">
          ออกได้ทั้งแบบ HTML และคำสั่ง ESC/POS สำหรับเครื่องพิมพ์สลิป 58/80 มม.
        </Feature>
      </ul>

      <div className="signin__stack">
        {['Go', 'PostgreSQL', 'React', 'TypeScript', 'PWA', 'Docker'].map((s) => (
          <span className="signin__chip" key={s}>
            {s}
          </span>
        ))}
      </div>
    </div>
  )
}

function Feature({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <li>
      <Check w={15} h={15} className="signin__tick" style={{ flexShrink: 0, marginTop: 2 }} />
      <span>
        <b>{title}</b> — {children}
      </span>
    </li>
  )
}

/**
 * Visitors are meant to sell things, void bills and delete products — that is
 * the point of a demo — so the way back to a clean dataset has to be in reach.
 */
function DemoFooter({ resetEvery, version }: { resetEvery?: string; version?: string }) {
  const [state, setState] = useState<'idle' | 'confirm' | 'working' | 'done' | 'error'>('idle')

  const reset = async () => {
    setState('working')
    try {
      await api.resetDemo()
      setState('done')
    } catch {
      setState('error')
    }
  }

  return (
    <div className="signin__foot">
      ข้อมูลทั้งหมดเป็นชุดตัวอย่างที่สร้างขึ้น (ยอดขายย้อนหลัง 30 วัน)
      {resetEvery ? ' และถูกสร้างใหม่อัตโนมัติทุกวัน' : ''} · แก้อะไรก็ได้ตามสบาย
      <br />
      {state === 'idle' && (
        <button className="signin__link" onClick={() => setState('confirm')} type="button">
          รีเซ็ตข้อมูลตัวอย่างเดี๋ยวนี้
        </button>
      )}
      {state === 'confirm' && (
        <>
          ลบข้อมูลปัจจุบันทั้งหมดและสร้างใหม่?{' '}
          <button className="signin__link" onClick={() => void reset()} type="button">
            ยืนยัน
          </button>{' '}
          ·{' '}
          <button className="signin__link" onClick={() => setState('idle')} type="button">
            ยกเลิก
          </button>
        </>
      )}
      {state === 'working' && 'กำลังสร้างข้อมูลใหม่…'}
      {state === 'done' && 'รีเซ็ตเรียบร้อย — เข้าใช้งานได้เลย'}
      {state === 'error' && 'รีเซ็ตไม่สำเร็จ (อาจเพิ่งมีคนรีเซ็ตไป) ลองอีกครั้งในอีกครู่'}
      {version && version !== 'dev' && (
        <span style={{ float: 'right', fontFamily: "'Space Grotesk',sans-serif" }}>{version}</span>
      )}
    </div>
  )
}
