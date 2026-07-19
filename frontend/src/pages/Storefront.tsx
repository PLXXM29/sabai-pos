import { useMemo, useState } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { El } from '../styled'
import * as I from '../icons'
import { CATS } from '../data'
import { db, type LocalProduct } from '../lib/db'
import { tileOf, statusOf } from '../lib/ui'
import { baht } from '../lib/format'

export default function Storefront() {
  const products = useLiveQuery(
    () => db.products.toArray().then((a) => a.sort((x, y) => x.name.localeCompare(y.name, 'th'))),
    [],
    [] as LocalProduct[],
  )
  const [q, setQ] = useState('')
  const [cat, setCat] = useState('ทั้งหมด')

  const items = useMemo(() => {
    const ql = q.trim().toLowerCase()
    return (products ?? []).filter(
      (p) => p.is_active && (!ql || p.name.toLowerCase().includes(ql)) && (cat === 'ทั้งหมด' || p.category === cat),
    )
  }, [products, q, cat])

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, padding: '16px 18px', gap: 12, overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <div>
          <div style={{ fontSize: 20, fontWeight: 700 }}>หน้าร้าน — สินค้าทั้งหมด</div>
          <div style={{ fontSize: 12.5, color: '#8A7A66' }}>มุมมองลูกค้า · {items.length} รายการ</div>
        </div>
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 999, padding: '8px 14px', width: 260 }}>
          <I.Search w={15} h={15} />
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="ค้นหาสินค้า…" style={{ border: 'none', outline: 'none', background: 'transparent', fontSize: 13.5, flex: 1, color: '#2B2420' }} />
        </div>
      </div>
      <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {['ทั้งหมด', ...CATS].map((c) => {
          const on = cat === c
          return (
            <El key={c} as="button" onClick={() => setCat(c)} s={`border:1.5px solid ${on ? '#C2571F' : '#E0D3BD'};background:${on ? '#C2571F' : '#fff'};color:${on ? '#fff' : '#2B2420'};border-radius:999px;padding:6px 15px;font-size:13px;font-weight:600;cursor:pointer;white-space:nowrap;`} hover="border-color:#C2571F;">{c}</El>
          )
        })}
      </div>
      <div style={{ flex: 1, overflowY: 'auto', paddingRight: 4 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill,minmax(170px,1fr))', gap: 10 }}>
          {items.map((p) => {
            const [tile, tileCol] = tileOf(p.category)
            const st = statusOf(p.qty_on_hand, p.reorder_point)
            return (
              <El key={p.id} s="background:#fff;border:1.5px solid #E0D3BD;border-radius:16px;overflow:hidden;transition:transform .12s, border-color .12s;" hover="transform:translateY(-3px);border-color:#C2571F;">
                <div style={{ height: 92, background: tile, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 36, color: tileCol, position: 'relative' }}>
                  {p.name.trim()[0]}
                  <span style={{ position: 'absolute', top: 8, left: 8, background: st.bg, color: st.col, fontSize: 11, fontWeight: 600, borderRadius: 999, padding: '2px 9px' }}>{st.t}</span>
                </div>
                <div style={{ padding: '10px 12px 12px 12px' }}>
                  <div style={{ fontSize: 13.5, fontWeight: 600, lineHeight: 1.3, height: 35, overflow: 'hidden' }}>{p.name}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginTop: 4 }}>
                    <span style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 18, color: '#C2571F' }}>{baht(p.sell_price)}</span>
                    <span style={{ fontSize: 11, color: '#8A7A66' }}>{p.category}</span>
                  </div>
                </div>
              </El>
            )
          })}
        </div>
      </div>
    </div>
  )
}
