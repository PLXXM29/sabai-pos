import { useEffect, useMemo, useState } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { El } from '../styled'
import * as I from '../icons'
import { TILES, CATS } from '../data'
import { db, type LocalProduct } from '../lib/db'
import { pullCatalog, enqueueCheckout } from '../lib/sync'
import { api } from '../lib/api'
import { useCart, cartTotal, cartCount } from '../store/cart'
import { baht } from '../lib/format'

function tileOf(cat: string): [string, string] {
  return TILES[cat] ?? TILES['บุหรี่/อื่นๆ']
}

export default function Cashier() {
  const products = useLiveQuery(
    () => db.products.toArray().then((a) => a.sort((x, y) => x.name.localeCompare(y.name, 'th'))),
    [],
    [] as LocalProduct[],
  )
  const { items, add, inc, dec, remove, clear } = useCart()
  const [q, setQ] = useState('')
  const [cat, setCat] = useState('ทั้งหมด')
  const [payOpen, setPayOpen] = useState(false)
  const [method, setMethod] = useState<'cash' | 'transfer'>('cash')
  const [received, setReceived] = useState('')
  const [toast, setToast] = useState('')
  const [cust, setCust] = useState(false)

  // Refresh catalog from server on mount (works offline: keeps last cache).
  useEffect(() => {
    pullCatalog().catch(() => {})
  }, [])

  const showToast = (m: string) => {
    setToast(m)
    setTimeout(() => setToast(''), 2600)
  }

  const filtered = useMemo(() => {
    const ql = q.trim().toLowerCase()
    return (products ?? []).filter(
      (p) =>
        p.is_active &&
        (!ql || p.name.toLowerCase().includes(ql) || (p.barcode ?? '').includes(ql)) &&
        (cat === 'ทั้งหมด' || p.category === cat),
    )
  }, [products, q, cat])

  const total = cartTotal(items)
  const recvSatang = Math.round((parseFloat(received) || 0) * 100)
  const change = recvSatang - total
  const canPay = items.length > 0
  const enoughCash = method !== 'cash' || (received !== '' && change >= 0)

  const openPay = (m: 'cash' | 'transfer') => {
    if (!canPay) return
    setMethod(m)
    setReceived('')
    setPayOpen(true)
  }

  const confirm = async () => {
    if (!enoughCash) return
    const lines = items.map((i) => ({ id: i.id, name: i.name, price: i.price, qty: i.qty }))
    const clientUUID = await enqueueCheckout(lines, method, method === 'cash' ? recvSatang : total)
    clear()
    setPayOpen(false)
    showToast('บันทึกบิลแล้ว')
    // If it syncs quickly, open the printable receipt.
    for (let i = 0; i < 12; i++) {
      const b = await db.bills.get(clientUUID)
      if (b?.server_id) {
        try {
          const html = await api.receiptHTML(b.server_id)
          const url = URL.createObjectURL(new Blob([html], { type: 'text/html' }))
          window.open(url, '_blank')
          setTimeout(() => URL.revokeObjectURL(url), 60000)
        } catch {
          /* offline / print later */
        }
        return
      }
      await new Promise((r) => setTimeout(r, 250))
    }
  }

  const cats = ['ทั้งหมด', ...CATS]

  // Keyboard-wedge barcode: scanner types into the search box then hits Enter.
  const onSearchKey = (e: React.KeyboardEvent) => {
    if (e.key !== 'Enter') return
    const bc = q.trim()
    if (!bc) return
    const exact = (products ?? []).find((p) => p.is_active && p.barcode === bc)
    const match = exact ?? (filtered.length === 1 ? filtered[0] : null)
    if (match) {
      add(match)
      setQ('')
    }
  }

  return (
    <div style={{ flex: 1, display: 'flex', minWidth: 0 }}>
      {/* left: search + grid */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, padding: '14px 14px 14px 16px', gap: 10 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 8, background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 999, padding: '8px 14px' }}>
            <I.Search w={16} h={16} />
            <input value={q} onChange={(e) => setQ(e.target.value)} onKeyDown={onSearchKey} placeholder="ค้นหาสินค้า หรือยิงบาร์โค้ด…" style={{ border: 'none', outline: 'none', background: 'transparent', fontSize: 14, flex: 1, color: '#2B2420' }} autoFocus />
          </div>
          <El as="button" onClick={() => setCust(true)} title="เปิดจอสำหรับลูกค้า" s="display:flex;align-items:center;gap:6px;background:#0F3B39;color:#F7F1E6;border:none;border-radius:999px;padding:9px 16px;font-size:13px;font-weight:600;cursor:pointer;white-space:nowrap;" hover="background:#134E4A;"><I.Monitor w={15} h={15} /> จอลูกค้า</El>
        </div>
        <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
          {cats.map((c) => {
            const on = cat === c
            return (
              <El key={c} as="button" onClick={() => setCat(c)}
                s={`border:1.5px solid ${on ? '#C2571F' : '#E0D3BD'};background:${on ? '#C2571F' : '#fff'};color:${on ? '#fff' : '#2B2420'};border-radius:999px;padding:6px 15px;font-size:13px;font-weight:600;cursor:pointer;white-space:nowrap;`}
                hover="border-color:#C2571F;">
                {c}
              </El>
            )
          })}
        </div>
        <div style={{ flex: 1, overflowY: 'auto', paddingRight: 4 }}>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill,minmax(128px,1fr))', gap: 8 }}>
            {filtered.map((p) => {
              const [tile, tileCol] = tileOf(p.category)
              const out = p.qty_on_hand <= 0
              return (
                <El key={p.id} as="button" onClick={() => add(p)}
                  s={`background:#fff;border:1.5px solid #E0D3BD;border-radius:14px;padding:0;cursor:${out ? 'not-allowed' : 'pointer'};opacity:${out ? 0.45 : 1};text-align:left;overflow:hidden;transition:transform .1s, border-color .12s;display:flex;flex-direction:column;`}
                  hover="border-color:#C2571F;transform:translateY(-2px);" active="transform:scale(.97);">
                  <div style={{ height: 56, background: tile, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 24, color: tileCol, position: 'relative', width: '100%' }}>
                    {p.name.trim()[0]}
                    {!out && p.qty_on_hand <= p.reorder_point && (
                      <span style={{ position: 'absolute', top: 5, right: 6, background: '#C23A2B', color: '#fff', fontSize: 10, borderRadius: 999, padding: '1px 7px', fontWeight: 600 }}>เหลือ {p.qty_on_hand}</span>
                    )}
                  </div>
                  <div style={{ padding: '7px 9px 9px 9px', display: 'flex', flexDirection: 'column', gap: 2, width: '100%' }}>
                    <div style={{ fontSize: 12.5, fontWeight: 500, lineHeight: 1.25, height: 31, overflow: 'hidden' }}>{p.name}</div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
                      <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 15, color: '#C2571F' }}>{baht(p.sell_price)}</span>
                      <span style={{ fontSize: 10.5, color: '#8A7A66' }}>{out ? 'หมด' : 'เหลือ ' + p.qty_on_hand}</span>
                    </div>
                  </div>
                </El>
              )
            })}
          </div>
        </div>
      </div>

      {/* right: cart */}
      <div style={{ width: 330, flexShrink: 0, background: '#fff', borderLeft: '1.5px solid #E0D3BD', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '14px 16px 10px 16px', borderBottom: '1.5px dashed #E0D3BD', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ fontWeight: 700, fontSize: 16 }}>บิลปัจจุบัน <span style={{ color: '#8A7A66', fontWeight: 500, fontSize: 13 }}>({cartCount(items)} ชิ้น)</span></div>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '8px 12px', display: 'flex', flexDirection: 'column', gap: 6 }}>
          {items.length === 0 && (
            <div style={{ textAlign: 'center', color: '#B5A88F', padding: '40px 20px', fontSize: 13, lineHeight: 1.6 }}>ยังไม่มีสินค้าในบิล<br />แตะสินค้าด้านซ้ายเพื่อเพิ่ม</div>
          )}
          {items.map((r) => {
            const [tile, tileCol] = tileOf(r.category)
            return (
              <div key={r.id} style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#FBF7EF', border: '1px solid #EFE5D2', borderRadius: 12, padding: '7px 9px' }}>
                <div style={{ width: 34, height: 34, borderRadius: 9, background: tile, color: tileCol, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 15, flexShrink: 0 }}>{r.name.trim()[0]}</div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 12.5, fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{r.name}</div>
                  <div style={{ fontSize: 11, color: '#8A7A66', fontFamily: "'Space Grotesk',sans-serif" }}>{baht(r.price)}</div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <El as="button" onClick={() => dec(r.id)} s="width:24px;height:24px;border-radius:50%;border:1.5px solid #E0D3BD;background:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;color:#2B2420;" hover="border-color:#C2571F;color:#C2571F;"><I.Minus w={12} h={12} /></El>
                  <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 14, width: 22, textAlign: 'center' }}>{r.qty}</span>
                  <El as="button" onClick={() => inc(r.id)} s="width:24px;height:24px;border-radius:50%;border:1.5px solid #E0D3BD;background:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;color:#2B2420;" hover="border-color:#17706A;color:#17706A;"><I.Plus w={12} h={12} /></El>
                </div>
                <div style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 13.5, width: 52, textAlign: 'right' }}>{baht(r.price * r.qty)}</div>
                <El as="button" onClick={() => remove(r.id)} s="width:24px;height:24px;border:none;background:transparent;cursor:pointer;color:#B5A88F;display:flex;align-items:center;justify-content:center;" hover="color:#C23A2B;"><I.Trash w={13} h={13} /></El>
              </div>
            )
          })}
        </div>
        <div style={{ borderTop: '1.5px dashed #E0D3BD', padding: '12px 16px 14px 16px', display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div style={{ display: 'flex', gap: 6 }}>
            <El as="button" onClick={() => clear()} s="flex:1;border:1.5px solid #E0D3BD;background:#fff;border-radius:999px;padding:7px 0;font-size:12.5px;font-weight:600;cursor:pointer;color:#2B2420;" hover="border-color:#C23A2B;color:#C23A2B;">ยกเลิกบิล</El>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#8A7A66' }}>ยอดรวม</span>
            <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 32, color: '#0F3B39' }}>{baht(total)}</span>
          </div>
          <div style={{ display: 'flex', gap: 8 }}>
            <El as="button" onClick={() => openPay('cash')} s={`flex:1.4;display:flex;align-items:center;justify-content:center;gap:7px;border:none;background:${canPay ? '#C2571F' : '#D8C9B0'};color:#fff;border-radius:999px;padding:13px 0;font-size:15px;font-weight:700;cursor:${canPay ? 'pointer' : 'not-allowed'};box-shadow:0 3px 0 #9C4517;`} active="transform:translateY(2px);box-shadow:0 1px 0 #9C4517;"><I.CardIcon w={17} h={17} /> รับเงินสด</El>
            <El as="button" onClick={() => openPay('transfer')} s={`flex:1;display:flex;align-items:center;justify-content:center;gap:6px;border:none;background:${canPay ? '#0F3B39' : '#B7C6C4'};color:#fff;border-radius:999px;padding:13px 0;font-size:14px;font-weight:700;cursor:${canPay ? 'pointer' : 'not-allowed'};box-shadow:0 3px 0 #0A2C2A;`} active="transform:translateY(2px);box-shadow:0 1px 0 #0A2C2A;"><I.Phone w={15} h={15} /> โอนจ่าย</El>
          </div>
        </div>
      </div>

      {/* pay modal */}
      {payOpen && (
        <div onClick={() => setPayOpen(false)} style={{ position: 'fixed', inset: 0, background: 'rgba(15,59,57,.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50 }}>
          <div onClick={(e) => e.stopPropagation()} style={{ background: '#fff', borderRadius: 20, width: 400, padding: '22px 24px', boxShadow: '0 20px 50px rgba(15,59,57,.3)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <div style={{ fontWeight: 700, fontSize: 18 }}>{method === 'cash' ? 'รับเงินสด' : 'รับโอน / พร้อมเพย์'}</div>
              <El as="button" onClick={() => setPayOpen(false)} s="border:none;background:#F0E7D6;border-radius:50%;width:30px;height:30px;cursor:pointer;display:flex;align-items:center;justify-content:center;color:#2B2420;" hover="background:#E0D3BD;"><I.Close w={14} h={14} sw={2.5} /></El>
            </div>
            <div style={{ background: '#FBF7EF', border: '1px solid #EFE5D2', borderRadius: 14, padding: '12px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 12 }}>
              <span style={{ fontSize: 13.5, color: '#8A7A66', fontWeight: 600 }}>ยอดที่ต้องชำระ</span>
              <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 28, color: '#0F3B39' }}>{baht(total)}</span>
            </div>
            {method === 'cash' ? (
              <>
                <div style={{ fontSize: 13, fontWeight: 600, color: '#8A7A66', marginBottom: 6 }}>รับเงินมา (บาท)</div>
                <El as="input" value={received} onChange={(e: any) => setReceived(e.target.value)} type="number" placeholder="0" s="width:100%;border:1.5px solid #E0D3BD;border-radius:12px;padding:11px 14px;font-size:22px;font-family:'Space Grotesk',sans-serif;font-weight:700;outline:none;color:#2B2420;margin-bottom:8px;" focus="border-color:#C2571F;" autoFocus />
                <div style={{ display: 'flex', gap: 6, marginBottom: 12 }}>
                  {[['พอดี', total / 100], ['100', 100], ['500', 500], ['1,000', 1000]].map(([label, val]) => (
                    <El key={label as string} as="button" onClick={() => setReceived(String(val))} s="flex:1;border:1.5px solid #E0D3BD;background:#fff;border-radius:999px;padding:7px 0;font-size:12.5px;font-weight:600;cursor:pointer;font-family:'Space Grotesk',sans-serif;" hover="border-color:#C2571F;color:#C2571F;">{label}</El>
                  ))}
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', background: received === '' ? '#FBF7EF' : change >= 0 ? '#DDEEE3' : '#F8DBD6', borderRadius: 14, padding: '12px 16px', marginBottom: 14 }}>
                  <span style={{ fontSize: 13.5, fontWeight: 600, color: '#8A7A66' }}>เงินทอน</span>
                  <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 26, color: received === '' ? '#B5A88F' : change >= 0 ? '#256B41' : '#A32617' }}>{received === '' ? '—' : change >= 0 ? baht(change) : 'ขาด ' + baht(-change)}</span>
                </div>
              </>
            ) : (
              <div style={{ textAlign: 'center', padding: '6px 0 14px 0' }}>
                <div style={{ width: 130, height: 130, margin: '0 auto 10px auto', border: '1.5px dashed #B5A88F', borderRadius: 14, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8A7A66', fontSize: 12, background: 'repeating-linear-gradient(45deg,#FBF7EF,#FBF7EF 6px,#F5EDDE 6px,#F5EDDE 12px)' }}>QR พร้อมเพย์</div>
                <div style={{ fontSize: 13, color: '#8A7A66' }}>ให้ลูกค้าสแกนจ่าย แล้วกดยืนยัน</div>
              </div>
            )}
            <El as="button" onClick={confirm} s={`width:100%;border:none;background:${enoughCash ? '#17706A' : '#B7C6C4'};color:#fff;border-radius:999px;padding:13px 0;font-size:15.5px;font-weight:700;cursor:${enoughCash ? 'pointer' : 'not-allowed'};box-shadow:0 3px 0 rgba(0,0,0,.25);`} active="transform:translateY(2px);box-shadow:none;">{method === 'cash' ? 'ยืนยันรับเงิน · พิมพ์ใบเสร็จ' : 'เงินเข้าแล้ว · ยืนยัน'}</El>
          </div>
        </div>
      )}

      {toast && (
        <div style={{ position: 'fixed', top: 20, left: '50%', transform: 'translateX(-50%)', zIndex: 70, background: '#0F3B39', color: '#F7F1E6', borderRadius: 999, padding: '12px 22px', fontSize: 13.5, fontWeight: 600, boxShadow: '0 8px 24px rgba(15,59,57,.35)', display: 'flex', alignItems: 'center', gap: 9 }}>
          <I.Check w={17} h={17} /> {toast}
        </div>
      )}

      {/* customer display — synced to the cart */}
      {cust && (
        <div style={{ position: 'fixed', inset: 0, zIndex: 60, background: '#0F3B39', display: 'flex', flexDirection: 'column', color: '#F7F1E6' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '26px 40px', borderBottom: '1px solid #1B5F5A' }}>
            <div style={{ width: 52, height: 52, borderRadius: 16, background: '#E89B2D', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, boxShadow: 'inset 0 -2px 0 rgba(0,0,0,.12)' }}>
              <I.Logo w={29} h={29} stroke="#0F3B39" />
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 700, fontSize: 26, lineHeight: 1.1 }}>มาร์ทมงคลชัย</div>
              <div style={{ color: '#7FA9A4', fontSize: 15 }}>ยินดีต้อนรับจ้ะ 🙏 · รายการสินค้าของคุณ</div>
            </div>
            <El as="button" onClick={() => setCust(false)} title="กลับหน้าแคชเชียร์" s="display:flex;align-items:center;gap:7px;border:1px solid #2A6E68;background:#134E4A;color:#9FC3BF;border-radius:999px;padding:9px 16px;font-size:13px;font-weight:600;cursor:pointer;" hover="background:#1B5F5A;color:#fff;"><I.Close w={15} h={15} /> ปิดจอลูกค้า</El>
          </div>
          <div style={{ flex: 1, overflowY: 'auto', padding: '20px 40px' }}>
            {items.length === 0 ? (
              <div style={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 18, color: '#5C807C' }}>
                <I.Logo w={76} h={76} stroke="#2A6E68" sw={1.6} />
                <div style={{ fontSize: 22, color: '#7FA9A4' }}>รอสแกนสินค้าชิ้นแรก…</div>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, maxWidth: 920, margin: '0 auto' }}>
                {items.map((r) => (
                  <div key={r.id} style={{ display: 'flex', alignItems: 'center', gap: 14, background: '#134E4A', borderRadius: 12, padding: '9px 16px' }}>
                    <div style={{ flex: 1, fontSize: 18, fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{r.name}</div>
                    <div style={{ fontSize: 15, color: '#7FA9A4', fontFamily: "'Space Grotesk',sans-serif" }}>{baht(r.price)} × {r.qty}</div>
                    <div style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 20, width: 110, textAlign: 'right' }}>{baht(r.price * r.qty)}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div style={{ borderTop: '1px solid #1B5F5A', background: '#0C302E', padding: '24px 40px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ color: '#7FA9A4', fontSize: 16 }}>รวม {cartCount(items)} ชิ้น · ยอดที่ต้องชำระ</div>
              <div style={{ color: '#9FC3BF', fontSize: 14 }}>ชำระเงินสด / พร้อมเพย์</div>
            </div>
            <div style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 72, lineHeight: 1, color: '#E89B2D' }}>{baht(total)}</div>
          </div>
        </div>
      )}
    </div>
  )
}
