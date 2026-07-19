import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { baht } from '../lib/format'

const DOW = ['อา.', 'จ.', 'อ.', 'พ.', 'พฤ.', 'ศ.', 'ส.']

export default function Dashboard() {
  const summary = useQuery({ queryKey: ['summary'], queryFn: api.summary })
  const top = useQuery({ queryKey: ['top'], queryFn: () => api.topProducts(5) })
  const daily = useQuery({ queryKey: ['daily'], queryFn: () => api.salesDaily(7) })

  if (summary.isError) {
    return (
      <div style={{ flex: 1, display: 'grid', placeItems: 'center', color: '#8A7A66', textAlign: 'center' }}>
        <div>
          <div style={{ fontSize: 18, fontWeight: 700, color: '#0F3B39', marginBottom: 6 }}>ดูแดชบอร์ดไม่ได้</div>
          <div style={{ fontSize: 13 }}>ต้องเป็นผู้จัดการขึ้นไป</div>
        </div>
      </div>
    )
  }

  const s = summary.data
  const days = daily.data ?? []
  const maxV = Math.max(1, ...days.map((d) => d.sales))
  const top5 = top.data ?? []
  const maxSold = Math.max(1, ...top5.map((t) => t.qty_sold))

  const card = (label: string, value: string, sub: string, opts?: { dark?: boolean; valueCol?: string; border?: string }) => (
    <div style={{ background: opts?.dark ? '#0F3B39' : '#fff', border: opts?.dark ? 'none' : `1.5px solid ${opts?.border ?? '#E0D3BD'}`, borderRadius: 16, padding: '16px 18px', color: opts?.dark ? '#F7F1E6' : '#2B2420' }}>
      <div style={{ fontSize: 12.5, color: opts?.dark ? '#7FA9A4' : '#8A7A66' }}>{label}</div>
      <div style={{ fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 30, margin: '2px 0', color: opts?.valueCol ?? (opts?.dark ? '#F7F1E6' : '#0F3B39') }}>{value}</div>
      <div style={{ fontSize: 12, color: opts?.dark ? '#7FA9A4' : '#8A7A66' }}>{sub}</div>
    </div>
  )

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, padding: '16px 18px', gap: 12, overflowY: 'auto' }}>
      <div>
        <div style={{ fontSize: 20, fontWeight: 700 }}>แดชบอร์ดสรุปยอด</div>
        <div style={{ fontSize: 12.5, color: '#8A7A66' }}>อัปเดตสดจากหน้าแคชเชียร์ (สุทธิหลังหักบิลที่ยกเลิก)</div>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 10 }}>
        {card('ยอดขายวันนี้', s ? baht(s.sales_today) : '—', '', { dark: true })}
        {card('จำนวนบิล', s ? String(s.bills) : '—', s ? `เฉลี่ย ${baht(s.avg_per_bill)}/บิล` : '')}
        {card('กำไรโดยประมาณ', s ? baht(s.profit) : '—', s ? `มาร์จิ้น ${s.margin_pct}%` : '', { valueCol: '#2F7D4F' })}
        {card('สินค้าใกล้หมด/หมด', s ? String(s.low_stock) : '—', 'ไปหน้าสต็อกเพื่อเติม', { valueCol: '#C23A2B', border: s && s.low_stock > 0 ? '#E5B7AE' : '#E0D3BD' })}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr', gap: 10, flex: 1, minHeight: 300 }}>
        <div style={{ background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 16, padding: '16px 18px', display: 'flex', flexDirection: 'column' }}>
          <div style={{ fontWeight: 700, fontSize: 15, marginBottom: 4 }}>ยอดขาย 7 วันล่าสุด</div>
          <div style={{ flex: 1, display: 'flex', alignItems: 'stretch', gap: 10, paddingTop: 8 }}>
            {days.map((d, i) => {
              const today = i === days.length - 1
              const label = DOW[new Date(d.day + 'T00:00:00').getDay()]
              return (
                <div key={d.day} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6 }}>
                  <div style={{ flex: 1, width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end', gap: 6 }}>
                    <div style={{ fontFamily: "'Space Grotesk',sans-serif", fontSize: 13, fontWeight: 600 }}>{(d.sales / 100 / 1000).toFixed(1)}k</div>
                    <div style={{ width: '100%', maxWidth: 62, height: `${Math.max(4, Math.round((d.sales / maxV) * 82))}%`, minHeight: 6, background: today ? '#C2571F' : '#E4D3B4', borderRadius: '8px 8px 4px 4px', transition: 'height .3s' }} />
                  </div>
                  <div style={{ fontSize: 12.5, color: today ? '#C2571F' : '#8A7A66', fontWeight: today ? 700 : 400 }}>{today ? 'วันนี้' : label}</div>
                </div>
              )
            })}
          </div>
        </div>
        <div style={{ background: '#fff', border: '1.5px solid #E0D3BD', borderRadius: 16, padding: '16px 18px', display: 'flex', flexDirection: 'column', gap: 9 }}>
          <div style={{ fontWeight: 700, fontSize: 15 }}>สินค้าขายดี Top 5</div>
          {top5.length === 0 && <div style={{ fontSize: 13, color: '#B5A88F' }}>ยังไม่มีข้อมูลการขาย</div>}
          {top5.map((t, i) => (
            <div key={t.id} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <div style={{ width: 24, height: 24, borderRadius: '50%', background: i === 0 ? '#E89B2D' : '#F0E7D6', color: i === 0 ? '#0F3B39' : '#8A7A66', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: "'Space Grotesk',sans-serif", fontWeight: 700, fontSize: 12, flexShrink: 0 }}>{i + 1}</div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, marginBottom: 3 }}>
                  <span style={{ fontWeight: 600, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{t.name}</span>
                  <span style={{ fontFamily: "'Space Grotesk',sans-serif", color: '#8A7A66', flexShrink: 0 }}>{t.qty_sold} ชิ้น</span>
                </div>
                <div style={{ height: 7, background: '#F0E7D6', borderRadius: 999, overflow: 'hidden' }}>
                  <div style={{ height: '100%', width: `${Math.round((t.qty_sold / maxSold) * 100)}%`, background: '#E89B2D', borderRadius: 999 }} />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
