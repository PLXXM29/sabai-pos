import { useMemo, useState } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { El } from '../styled'
import * as I from '../icons'
import { CATS } from '../data'
import { db, type LocalProduct } from '../lib/db'
import { tileOf, statusOf } from '../lib/ui'
import { ProductArt } from '../productArt'
import { baht } from '../lib/format'
import { api, ApiError } from '../lib/api'
import { pullCatalog } from '../lib/sync'
import { useAuth, canManage } from '../lib/auth'

type FormState = {
  id?: string
  name: string
  barcode: string
  category: string
  costBaht: string
  priceBaht: string
  reorder: string
  initial: string
}
const emptyForm = (): FormState => ({ name: '', barcode: '', category: CATS[0], costBaht: '', priceBaht: '', reorder: '10', initial: '' })

export default function Inventory() {
  const { user } = useAuth()
  const manage = canManage(user?.role)
  const products = useLiveQuery(
    () => db.products.toArray().then((a) => a.sort((x, y) => x.name.localeCompare(y.name, 'th'))),
    [],
    [] as LocalProduct[],
  )
  const [q, setQ] = useState('')
  const [cat, setCat] = useState('ทั้งหมด')
  const [status, setStatus] = useState('ทั้งหมด')
  const [form, setForm] = useState<FormState | null>(null)
  const [formErr, setFormErr] = useState('')
  const [receiveP, setReceiveP] = useState<LocalProduct | null>(null)
  const [receiveQty, setReceiveQty] = useState('')
  const [busy, setBusy] = useState(false)
  const [toast, setToast] = useState('')

  const showToast = (m: string) => {
    setToast(m)
    setTimeout(() => setToast(''), 2600)
  }
  const statusKey = (p: LocalProduct) => (p.qty_on_hand <= 0 ? 'หมด' : p.qty_on_hand <= p.reorder_point ? 'ใกล้หมด' : 'มีของ')

  const rows = useMemo(() => {
    const ql = q.trim().toLowerCase()
    return (products ?? []).filter(
      (p) =>
        (!ql || p.name.toLowerCase().includes(ql) || (p.barcode ?? '').includes(ql)) &&
        (cat === 'ทั้งหมด' || p.category === cat) &&
        (status === 'ทั้งหมด' || statusKey(p) === status),
    )
  }, [products, q, cat, status])

  const lowCount = (products ?? []).filter((p) => p.qty_on_hand <= p.reorder_point).length

  const openAdd = () => {
    setFormErr('')
    setForm(emptyForm())
  }
  const openEdit = (p: LocalProduct) => {
    setFormErr('')
    setForm({
      id: p.id,
      name: p.name,
      barcode: p.barcode ?? '',
      category: p.category,
      costBaht: String(p.cost_price / 100),
      priceBaht: String(p.sell_price / 100),
      reorder: String(p.reorder_point),
      initial: '',
    })
  }

  const saveForm = async () => {
    if (!form) return
    if (!form.name.trim()) return setFormErr('กรุณากรอกชื่อสินค้า')
    const price = Math.round((parseFloat(form.priceBaht) || 0) * 100)
    if (price <= 0) return setFormErr('ราคาขายต้องมากกว่า 0')
    const input = {
      name: form.name.trim(),
      barcode: form.barcode.trim() || null,
      category: form.category,
      cost_price: Math.round((parseFloat(form.costBaht) || 0) * 100),
      sell_price: price,
      reorder_point: parseInt(form.reorder) || 0,
    }
    setBusy(true)
    try {
      if (form.id) {
        await api.updateProduct(form.id, input)
      } else {
        const created = await api.createProduct(input)
        const initial = parseInt(form.initial) || 0
        if (initial > 0) await api.receiveStock(created.id, initial, 'initial stock')
      }
      await pullCatalog()
      setForm(null)
      showToast('บันทึกสำเร็จ')
    } catch (e) {
      setFormErr(e instanceof ApiError ? e.message : 'บันทึกไม่สำเร็จ')
    } finally {
      setBusy(false)
    }
  }

  const doReceive = async () => {
    if (!receiveP) return
    const qty = parseInt(receiveQty) || 0
    if (qty <= 0) return
    setBusy(true)
    try {
      await api.receiveStock(receiveP.id, qty, 'manual receive')
      await pullCatalog()
      setReceiveP(null)
      setReceiveQty('')
      showToast('รับสินค้าเข้าแล้ว')
    } catch {
      showToast('รับเข้าไม่สำเร็จ')
    } finally {
      setBusy(false)
    }
  }

  const doDelete = async (p: LocalProduct) => {
    if (!confirm(`ลบ "${p.name}" ออกจากสต็อก?`)) return
    try {
      await api.deleteProduct(p.id)
      await pullCatalog()
      showToast('ลบสินค้าแล้ว')
    } catch {
      showToast('ลบไม่สำเร็จ')
    }
  }

  const inp = 'width:100%;border:1.5px solid #E0D3BD;border-radius:10px;padding:9px 12px;font-size:14px;outline:none;background:#fff;'
  const inpNum = inp + "font-family:'Space Grotesk',sans-serif;"

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, padding: '16px 18px', gap: 12, overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
        <div>
          <div style={{ fontSize: 20, fontWeight: 700 }}>จัดการสต็อกสินค้า</div>
          <div style={{ fontSize: 12.5, color: '#8A7A66' }}>{(products ?? []).length} รายการ · <span style={{ color: '#C23A2B', fontWeight: 600 }}>{lowCount} ใกล้หมด/หมด</span></div>
        </div>
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 999, padding: '8px 14px', width: 220 }}>
          <I.Search w={15} h={15} />
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="ค้นหาชื่อ / บาร์โค้ด…" style={{ border: 'none', outline: 'none', background: 'transparent', fontSize: 13.5, flex: 1, color: '#2B2420' }} />
        </div>
        {manage && (
          <El as="button" onClick={openAdd} s="display:flex;align-items:center;gap:6px;border:none;background:#C2571F;color:#fff;border-radius:999px;padding:9px 18px;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 0 #9C4517;" active="transform:translateY(2px);box-shadow:none;"><I.Plus w={14} h={14} /> เพิ่มสินค้า</El>
        )}
      </div>

      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {['ทั้งหมด', ...CATS].map((c) => {
          const on = cat === c
          return <El key={c} as="button" onClick={() => setCat(c)} s={`border:1.5px solid ${on ? '#C2571F' : '#E0D3BD'};background:${on ? '#C2571F' : '#fff'};color:${on ? '#fff' : '#2B2420'};border-radius:999px;padding:5px 13px;font-size:12.5px;font-weight:600;cursor:pointer;`} hover="border-color:#C2571F;">{c}</El>
        })}
        <div style={{ width: 12 }} />
        {['ทั้งหมด', 'มีของ', 'ใกล้หมด', 'หมด'].map((s) => {
          const on = status === s
          return <El key={s} as="button" onClick={() => setStatus(s)} s={`border:1.5px solid ${on ? '#17706A' : '#E0D3BD'};background:${on ? '#E4F0EE' : '#fff'};color:${on ? '#17706A' : '#8A7A66'};border-radius:999px;padding:5px 13px;font-size:12.5px;font-weight:600;cursor:pointer;`} hover="border-color:#17706A;">{s}</El>
        })}
      </div>

      <div style={{ flex: 1, overflow: 'auto', background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 16 }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            <tr style={{ position: 'sticky', top: 0, background: '#0F3B39', color: '#CBDEDB', zIndex: 2 }}>
              <th style={{ padding: '10px 8px 10px 14px', textAlign: 'left', fontWeight: 600 }}>รูป</th>
              <th style={{ padding: '10px 8px', textAlign: 'left', fontWeight: 600 }}>ชื่อสินค้า</th>
              <th style={{ padding: '10px 8px', textAlign: 'left', fontWeight: 600 }}>บาร์โค้ด</th>
              <th style={{ padding: '10px 8px', textAlign: 'left', fontWeight: 600 }}>หมวด</th>
              <th style={{ padding: '10px 8px', textAlign: 'right', fontWeight: 600 }}>ทุน</th>
              <th style={{ padding: '10px 8px', textAlign: 'right', fontWeight: 600 }}>ขาย</th>
              <th style={{ padding: '10px 8px', textAlign: 'right', fontWeight: 600 }}>คงเหลือ</th>
              <th style={{ padding: '10px 8px', textAlign: 'center', fontWeight: 600 }}>สถานะ</th>
              {manage && <th style={{ padding: '10px 14px 10px 8px', textAlign: 'right', fontWeight: 600 }}>จัดการ</th>}
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => {
              const [tile, tileCol] = tileOf(p.category)
              const st = statusOf(p.qty_on_hand, p.reorder_point)
              const stockCol = p.qty_on_hand <= 0 ? '#A32617' : p.qty_on_hand <= p.reorder_point ? '#8F6410' : '#2B2420'
              return (
                <El key={p.id} as="tr" s="border-bottom:1px solid #F0E7D6;" hover="background:#FBF7EF;">
                  <td style={{ padding: '6px 8px 6px 14px' }}>
                    <div style={{ width: 34, height: 34, borderRadius: 9, background: tile, color: tileCol, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}><ProductArt name={p.name} cat={p.category} size={26} color={tileCol} /></div>
                  </td>
                  <td style={{ padding: '6px 8px', fontWeight: 600 }}>{p.name}</td>
                  <td style={{ padding: '6px 8px', fontFamily: "'Space Grotesk',sans-serif", color: '#8A7A66', fontSize: 12 }}>{p.barcode}</td>
                  <td style={{ padding: '6px 8px', color: '#8A7A66' }}>{p.category}</td>
                  <td style={{ padding: '6px 8px', textAlign: 'right', fontFamily: "'Space Grotesk',sans-serif" }}>{baht(p.cost_price)}</td>
                  <td style={{ padding: '6px 8px', textAlign: 'right', fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700 }}>{baht(p.sell_price)}</td>
                  <td style={{ padding: '6px 8px', textAlign: 'right', fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, color: stockCol }}>{p.qty_on_hand}</td>
                  <td style={{ padding: '6px 8px', textAlign: 'center' }}>
                    <span style={{ background: st.bg, color: st.col, fontSize: 11.5, fontWeight: 600, borderRadius: 999, padding: '3px 11px', whiteSpace: 'nowrap' }}>{st.t}</span>
                  </td>
                  {manage && (
                    <td style={{ padding: '6px 14px 6px 8px', textAlign: 'right', whiteSpace: 'nowrap' }}>
                      <El as="button" onClick={() => { setReceiveP(p); setReceiveQty('') }} s="border:1.5px solid #17706A;background:#E4F0EE;color:#17706A;border-radius:999px;padding:4px 12px;font-size:12px;font-weight:600;cursor:pointer;margin-right:5px;" hover="background:#D3E7E4;">รับเข้า</El>
                      <El as="button" onClick={() => openEdit(p)} s="border:1.5px solid #E0D3BD;background:#fff;border-radius:999px;padding:4px 12px;font-size:12px;font-weight:600;cursor:pointer;color:#2B2420;margin-right:5px;" hover="border-color:#C2571F;color:#C2571F;">แก้ไข</El>
                      <El as="button" onClick={() => doDelete(p)} s="border:none;background:transparent;color:#B5A88F;cursor:pointer;padding:4px;vertical-align:middle;" hover="color:#C23A2B;"><I.Trash w={14} h={14} /></El>
                    </td>
                  )}
                </El>
              )
            })}
          </tbody>
        </table>
      </div>

      {/* add/edit modal */}
      {form && (
        <div onClick={() => setForm(null)} style={{ position: 'fixed', inset: 0, background: 'rgba(15,59,57,.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50 }}>
          <div onClick={(e) => e.stopPropagation()} style={{ background: '#fff', borderRadius: 20, width: 430, padding: '22px 24px', boxShadow: '0 20px 50px rgba(15,59,57,.3)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
              <div style={{ fontWeight: 700, fontSize: 18 }}>{form.id ? 'แก้ไขสินค้า' : 'เพิ่มสินค้าใหม่'}</div>
              <El as="button" onClick={() => setForm(null)} s="border:none;background:#F0E7D6;border-radius:50%;width:30px;height:30px;cursor:pointer;display:flex;align-items:center;justify-content:center;" hover="background:#E0D3BD;"><I.Close w={14} h={14} sw={2.5} /></El>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div>
                <div style={{ fontSize: 12.5, fontWeight: 600, color: '#8A7A66', marginBottom: 4 }}>ชื่อสินค้า *</div>
                <El as="input" value={form.name} onChange={(e: any) => setForm({ ...form, name: e.target.value })} s={inp} focus="border-color:#C2571F;" />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 600, color: '#8A7A66', marginBottom: 4 }}>บาร์โค้ด</div>
                  <El as="input" value={form.barcode} onChange={(e: any) => setForm({ ...form, barcode: e.target.value })} s={inpNum} focus="border-color:#C2571F;" />
                </div>
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 600, color: '#8A7A66', marginBottom: 4 }}>หมวดหมู่</div>
                  <El as="select" value={form.category} onChange={(e: any) => setForm({ ...form, category: e.target.value })} s={inp}>
                    {CATS.map((c) => <option key={c} value={c}>{c}</option>)}
                  </El>
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 600, color: '#8A7A66', marginBottom: 4 }}>ทุน (฿)</div>
                  <El as="input" value={form.costBaht} onChange={(e: any) => setForm({ ...form, costBaht: e.target.value })} type="number" s={inpNum} focus="border-color:#C2571F;" />
                </div>
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 600, color: '#8A7A66', marginBottom: 4 }}>ขาย (฿) *</div>
                  <El as="input" value={form.priceBaht} onChange={(e: any) => setForm({ ...form, priceBaht: e.target.value })} type="number" s={inpNum} focus="border-color:#C2571F;" />
                </div>
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 600, color: '#8A7A66', marginBottom: 4 }}>{form.id ? 'จุดสั่งซื้อ' : 'จำนวนเริ่มต้น'}</div>
                  <El as="input" value={form.id ? form.reorder : form.initial} onChange={(e: any) => setForm(form.id ? { ...form, reorder: e.target.value } : { ...form, initial: e.target.value })} type="number" s={inpNum} focus="border-color:#C2571F;" />
                </div>
              </div>
              {formErr && <div style={{ color: '#C23A2B', fontSize: 12.5, fontWeight: 600 }}>{formErr}</div>}
              <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                <El as="button" onClick={() => setForm(null)} s="flex:1;border:1.5px solid #E0D3BD;background:#fff;border-radius:999px;padding:11px 0;font-size:14px;font-weight:600;cursor:pointer;color:#2B2420;" hover="border-color:#8A7A66;">ยกเลิก</El>
                <El as="button" onClick={saveForm} s={`flex:1.5;border:none;background:${busy ? '#D8C9B0' : '#C2571F'};color:#fff;border-radius:999px;padding:11px 0;font-size:14px;font-weight:700;cursor:${busy ? 'default' : 'pointer'};box-shadow:0 2px 0 #9C4517;`} active="transform:translateY(2px);box-shadow:none;">{form.id ? 'บันทึกการแก้ไข' : 'เพิ่มสินค้า'}</El>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* receive modal */}
      {receiveP && (
        <div onClick={() => setReceiveP(null)} style={{ position: 'fixed', inset: 0, background: 'rgba(15,59,57,.45)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50 }}>
          <div onClick={(e) => e.stopPropagation()} style={{ background: '#fff', borderRadius: 20, width: 340, padding: '22px 24px', boxShadow: '0 20px 50px rgba(15,59,57,.3)' }}>
            <div style={{ fontWeight: 700, fontSize: 17, marginBottom: 2 }}>รับสินค้าเข้า</div>
            <div style={{ fontSize: 13, color: '#8A7A66', marginBottom: 12 }}>{receiveP.name} · คงเหลือ <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700 }}>{receiveP.qty_on_hand}</span></div>
            <El as="input" value={receiveQty} onChange={(e: any) => setReceiveQty(e.target.value)} type="number" placeholder="จำนวนที่รับเข้า" s="width:100%;border:1.5px solid #E0D3BD;border-radius:12px;padding:10px 14px;font-size:20px;font-family:'Space Grotesk',sans-serif;font-weight:700;outline:none;margin-bottom:8px;" focus="border-color:#17706A;" autoFocus />
            <div style={{ display: 'flex', gap: 6, marginBottom: 14 }}>
              {[10, 24, 50].map((n) => (
                <El key={n} as="button" onClick={() => setReceiveQty(String((parseInt(receiveQty) || 0) + n))} s="flex:1;border:1.5px solid #E0D3BD;background:#fff;border-radius:999px;padding:7px 0;font-size:12.5px;font-weight:600;cursor:pointer;font-family:'Space Grotesk',sans-serif;" hover="border-color:#17706A;color:#17706A;">+{n}</El>
              ))}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <El as="button" onClick={() => setReceiveP(null)} s="flex:1;border:1.5px solid #E0D3BD;background:#fff;border-radius:999px;padding:10px 0;font-size:13.5px;font-weight:600;cursor:pointer;color:#2B2420;">ยกเลิก</El>
              <El as="button" onClick={doReceive} s="flex:1.5;border:none;background:#17706A;color:#fff;border-radius:999px;padding:10px 0;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 0 #0A2C2A;" active="transform:translateY(2px);box-shadow:none;">ยืนยันรับเข้า</El>
            </div>
          </div>
        </div>
      )}

      {toast && (
        <div style={{ position: 'fixed', top: 20, left: '50%', transform: 'translateX(-50%)', zIndex: 70, background: '#0F3B39', color: '#F7F1E6', borderRadius: 999, padding: '12px 22px', fontSize: 13.5, fontWeight: 600, boxShadow: '0 8px 24px rgba(15,59,57,.35)', display: 'flex', alignItems: 'center', gap: 9 }}>
          <I.Check w={17} h={17} /> {toast}
        </div>
      )}
    </div>
  )
}
