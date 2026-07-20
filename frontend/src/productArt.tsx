// ─────────────────────────────────────────────────────────────────────────
// MiniMart POS — ภาพสินค้า (product art)
//
// ไทล์สินค้าเคยโชว์แค่ตัวอักษรแรกของชื่อ ไฟล์นี้แทนที่ด้วยภาพวาดจริงของสินค้า
// จับคู่จาก "ชื่อสินค้า" (ลูกอม → รูปลูกอม, น้ำดื่ม → รูปขวดน้ำ) ถ้าจับคู่ไม่ได้
// ก็ตกไปที่ภาพกลางของหมวดนั้นแทน — ไม่มีทางกลับไปเป็นตัวอักษรอีก
//
// ทำไมเป็น SVG ไม่ใช่รูปถ่าย: แอปนี้เป็น offline-first PWA รูปจากอินเทอร์เน็ต
// จะหายตอนออฟไลน์และคุมสไตล์ไม่ได้ ส่วน SVG ฝังมากับ bundle · คมทุกขนาด ·
// รับสีของหมวดสินค้าไปเลย ทำให้ทั้งหน้าจอดูเป็นชุดเดียวกัน
// ─────────────────────────────────────────────────────────────────────────

import type { ReactNode } from 'react'
import { TILES } from './data'

/** ตัวสินค้าเป็นสีขาว ตัดเส้น/แต้มสีด้วยสีประจำหมวด */
const W = '#fff'

type Glyph = (c: string) => ReactNode

// ── เครื่องดื่ม ──────────────────────────────────────────────────────────
const water: Glyph = (c) => (
  <>
    <rect x="13" y="2.6" width="6" height="3.6" rx="1.2" fill={c} stroke="none" />
    <path d="M13.4 6.2h5.2v2.2c0 1.3 2.6 2.1 2.6 4.3V26a2.4 2.4 0 0 1-2.4 2.4h-5.6A2.4 2.4 0 0 1 10.8 26V12.7c0-2.2 2.6-3 2.6-4.3z" fill={W} />
    <path d="M10.8 15.6h10.4v5.6H10.8z" fill={c} opacity=".85" stroke="none" />
    <path d="M13 17.6h6M13 19.4h4" stroke={W} strokeWidth="1.2" />
  </>
)

const soda: Glyph = (c) => (
  <>
    <rect x="13" y="2.6" width="6" height="3.6" rx="1.2" fill={c} stroke="none" />
    <path d="M13.4 6.2h5.2v2.2c0 1.3 2.6 2.1 2.6 4.3V26a2.4 2.4 0 0 1-2.4 2.4h-5.6A2.4 2.4 0 0 1 10.8 26V12.7c0-2.2 2.6-3 2.6-4.3z" fill={W} />
    <path d="M10.8 16.4h10.4v5.2H10.8z" fill={c} opacity=".28" stroke="none" />
    <circle cx="14" cy="18" r="1.1" fill={c} opacity=".9" stroke="none" />
    <circle cx="18" cy="17.2" r=".8" fill={c} opacity=".9" stroke="none" />
    <circle cx="17" cy="20.2" r="1" fill={c} opacity=".9" stroke="none" />
  </>
)

const cola: Glyph = (c) => (
  <>
    <path d="M11 7.5c0-1.4 2.2-2.5 5-2.5s5 1.1 5 2.5v17c0 1.4-2.2 2.5-5 2.5s-5-1.1-5-2.5z" fill={W} />
    <ellipse cx="16" cy="7.5" rx="5" ry="2.5" fill={W} />
    <path d="M14.4 7.3h3.2" strokeWidth="1.2" />
    <path d="M11 13.4h10v6.4H11z" fill={c} opacity=".9" stroke="none" />
    <path d="M11.8 18.4c2.6-2.8 6-2.8 8.6 0" stroke={W} strokeWidth="1.5" />
  </>
)

const tea: Glyph = (c) => (
  <>
    <rect x="13.2" y="2.6" width="5.6" height="3.4" rx="1.1" fill={c} stroke="none" />
    <path d="M13.6 6h4.8v2c0 1.4 2.4 2.2 2.4 4.4V26a2.4 2.4 0 0 1-2.4 2.4h-4.8A2.4 2.4 0 0 1 11.2 26V12.4c0-2.2 2.4-3 2.4-4.4z" fill={W} />
    <path d="M11.2 14.6h9.6v7h-9.6z" fill={c} opacity=".85" stroke="none" />
    <path d="M13.6 20c0-2.4 1.9-4.3 4.3-4.3 0 2.4-1.9 4.3-4.3 4.3z" fill={W} stroke="none" />
    <path d="M13.9 19.8 18 15.9" stroke={c} strokeWidth="1" opacity=".7" />
  </>
)

const coffee: Glyph = (c) => (
  <>
    <path d="M11 8.5c0-1.3 2.2-2.3 5-2.3s5 1 5 2.3v15c0 1.3-2.2 2.3-5 2.3s-5-1-5-2.3z" fill={W} />
    <ellipse cx="16" cy="8.5" rx="5" ry="2.3" fill={W} />
    <path d="M11 13.8h10v6.4H11z" fill={c} opacity=".9" stroke="none" />
    <ellipse cx="16" cy="17" rx="2.8" ry="1.9" fill={W} stroke="none" transform="rotate(-20 16 17)" />
    <path d="M13.9 18c1.3-1.5 2.9-1.9 4.3-1.9" stroke={c} strokeWidth="1.1" />
  </>
)

const yogurt: Glyph = (c) => (
  <>
    <rect x="13" y="3.4" width="6" height="2.8" rx=".9" fill={c} stroke="none" />
    <path d="M13.6 6.2h4.8v2.6h-4.8z" fill={W} />
    <path d="M11.8 12.4c0-2 1.8-2.6 1.8-3.6h4.8c0 1 1.8 1.6 1.8 3.6V24a3 3 0 0 1-3 3h-2.4a3 3 0 0 1-3-3z" fill={W} />
    <path d="M11.8 15h8.4v5.6h-8.4z" fill={c} opacity=".85" stroke="none" />
    <path d="M14 17.8h4.2" stroke={W} strokeWidth="1.2" />
  </>
)

const energy: Glyph = (c) => (
  <>
    <rect x="13.2" y="3" width="5.6" height="3" rx="1" fill={c} stroke="none" />
    <path d="M13.4 6h5.2v2.6c0 1.4 2.4 2.4 2.4 4.6V24a3 3 0 0 1-3 3h-4a3 3 0 0 1-3-3V13.2c0-2.2 2.4-3.2 2.4-4.6z" fill={W} />
    <path d="M11 14h10v7H11z" fill={c} opacity=".9" stroke="none" />
    <path d="m17 15.2-3 3.9h2.1l-.9 2.6 3.2-3.9h-2.2z" fill={W} stroke="none" />
  </>
)

const juice: Glyph = (c) => (
  <>
    <path d="m19.4 3.4-1.7 3.6" strokeWidth="1.8" />
    <path d="M10 10h12v17a1.5 1.5 0 0 1-1.5 1.5h-9A1.5 1.5 0 0 1 10 27z" fill={W} />
    <path d="m10 10 3-3.5h6l3 3.5" fill={W} />
    <path d="M10 16.2h12v6.2H10z" fill={c} opacity=".85" stroke="none" />
    <circle cx="16" cy="19.3" r="2" fill={W} stroke="none" />
    <path d="M16 17.6v3.4" stroke={c} strokeWidth="1" opacity=".7" />
  </>
)

const drink: Glyph = (c) => (
  <>
    <path d="m20.6 4.6-1.4 5" strokeWidth="1.8" />
    <path d="M9.6 9.6h12.8l-1.5 16.6a2.2 2.2 0 0 1-2.2 2h-5.4a2.2 2.2 0 0 1-2.2-2z" fill={W} />
    <path d="M10.2 16h11.6l-.5 5.6H10.7z" fill={c} opacity=".85" stroke="none" />
  </>
)

// ── ขนม ─────────────────────────────────────────────────────────────────
const chips: Glyph = (c) => (
  <>
    <path d="M8.5 6.5h15l-1.4 20.4a1.6 1.6 0 0 1-1.6 1.5H11.5a1.6 1.6 0 0 1-1.6-1.5z" fill={W} />
    <path d="m8.5 6.5 1.6-2.6h11.8l1.6 2.6z" fill={c} opacity=".9" stroke="none" />
    <path d="M9.6 14.8h12.8l-.4 5.8H10z" fill={c} opacity=".85" stroke="none" />
    <path d="M13.2 18.6c1.1-1.8 4.5-1.8 5.6 0" stroke={W} strokeWidth="1.5" />
  </>
)

const wafer: Glyph = (c) => (
  <>
    <rect x="6.6" y="11" width="18.8" height="3.5" rx="1" fill={W} />
    <rect x="6.6" y="14.5" width="18.8" height="3.4" rx=".6" fill={c} opacity=".75" stroke="none" />
    <rect x="6.6" y="17.9" width="18.8" height="3.5" rx="1" fill={W} />
    <rect x="6.6" y="11" width="18.8" height="10.4" rx="1.4" />
    <path d="M12.9 11v10.4M19.1 11v10.4" strokeWidth="1" opacity=".55" />
  </>
)

const fish: Glyph = (c) => (
  <>
    <path d="M9.4 16c0-4 3.8-7.2 8.4-7.2S26.6 12 26.6 16s-4.2 7.2-8.8 7.2S9.4 20 9.4 16z" fill={W} />
    <path d="m9.4 16-4.8-4.4v8.8z" fill={c} opacity=".85" stroke="none" />
    <path d="M15.4 10.2c1.4-2 3.2-2 4.6-.4" strokeWidth="1.2" opacity=".6" />
    <path d="M13.6 12.4c1.4 2.2 1.4 5 0 7.2M17 11.8c1.5 2.6 1.5 5.8 0 8.4" strokeWidth="1.2" opacity=".45" />
    <circle cx="22.4" cy="14.2" r="1.1" fill={c} stroke="none" />
  </>
)

const seaweed: Glyph = (c) => (
  <>
    <rect x="6.5" y="6.5" width="19" height="19" rx="2.2" fill={W} />
    <path d="M6.5 10.8h19" strokeWidth="1.1" opacity=".45" />
    <path d="M10.6 14.4c1.8-1.6 3.6 1.6 5.4 0s3.6 1.6 5.4 0v6c-1.8 1.6-3.6-1.6-5.4 0s-3.6-1.6-5.4 0z" fill={c} opacity=".85" stroke="none" />
  </>
)

const candy: Glyph = (c) => (
  <>
    <path d="M10.5 13.6 5.6 10.8v10.4l4.9-2.8z" fill={c} opacity=".85" stroke="none" />
    <path d="M21.5 13.6l4.9-2.8v10.4l-4.9-2.8z" fill={c} opacity=".85" stroke="none" />
    <circle cx="16" cy="16" r="5.8" fill={W} />
    <path d="M13.3 17.9c-.3-2.8 2-5.1 4.5-4.2 1.7.6 2.1 2.9.6 3.8-1.1.7-2.3-.3-1.9-1.4" strokeWidth="1.4" />
  </>
)

const biscuit: Glyph = (c) => (
  <>
    <circle cx="16" cy="16" r="8.6" fill={W} />
    <path d="M7.7 14.3h16.6v3.4H7.7z" fill={c} opacity=".85" stroke="none" />
    <circle cx="16" cy="16" r="8.6" fill="none" />
    <circle cx="12.6" cy="11.2" r=".9" fill={c} stroke="none" />
    <circle cx="16" cy="9.8" r=".9" fill={c} stroke="none" />
    <circle cx="19.4" cy="11.2" r=".9" fill={c} stroke="none" />
    <circle cx="12.6" cy="20.8" r=".9" fill={c} stroke="none" />
    <circle cx="16" cy="22.2" r=".9" fill={c} stroke="none" />
    <circle cx="19.4" cy="20.8" r=".9" fill={c} stroke="none" />
  </>
)

const bread: Glyph = (c) => (
  <>
    <path d="M5.6 20.4c0-6 4.7-10.4 10.4-10.4s10.4 4.4 10.4 10.4v.6h-20.8z" fill={W} />
    <path d="M5.6 19.8h20.8v2.6a2 2 0 0 1-2 2H7.6a2 2 0 0 1-2-2z" fill={c} opacity=".85" stroke="none" />
    <path d="M11.4 15.4c1-1.4 2.2-2.1 3.6-2.2M17 13.4c1.4.2 2.6.9 3.5 2.2" strokeWidth="1.2" opacity=".55" />
  </>
)

// ── อาหาร ────────────────────────────────────────────────────────────────
const noodle: Glyph = (c) => (
  <>
    <rect x="5.5" y="8" width="21" height="16" rx="2.4" fill={W} />
    <path d="M5.5 12.2h21" strokeWidth="1.1" opacity=".4" />
    <path d="M9.4 16.4c2-2.2 4-2.2 6 0s4 2.2 6 0M9.4 20.4c2-2.2 4-2.2 6 0s4 2.2 6 0" stroke={c} strokeWidth="1.6" opacity=".9" />
    <path d="M22.6 8v-2.2M26.5 10.4l2-1.4" strokeWidth="1.3" opacity=".5" />
  </>
)

const cupNoodle: Glyph = (c) => (
  <>
    <path d="M13.4 6.2c.9-1.1.9-2.2 0-3.3M18.6 6.2c.9-1.1.9-2.2 0-3.3" strokeWidth="1.2" opacity=".55" />
    <path d="M9.6 11h12.8l-1.6 15.2a2 2 0 0 1-2 1.8h-5.6a2 2 0 0 1-2-1.8z" fill={W} />
    <ellipse cx="16" cy="10.8" rx="7.4" ry="2.4" fill={c} opacity=".85" stroke="none" />
    <path d="M10.5 17h11l-.5 4.6H11z" fill={c} opacity=".28" stroke="none" />
  </>
)

const bento: Glyph = (c) => (
  <>
    <rect x="5.4" y="12.2" width="21.2" height="13.4" rx="2" fill={W} />
    <rect x="4" y="8.4" width="24" height="4.6" rx="1.6" fill={c} opacity=".9" stroke="none" />
    <path d="M16 13.4v12.2" strokeWidth="1.1" opacity=".45" />
    <path d="M7.8 21.8c0-2.2 1.7-3.8 3.7-3.8s3.7 1.6 3.7 3.8z" fill={c} opacity=".3" stroke="none" />
    <circle cx="21.4" cy="19.6" r="2.1" fill={c} opacity=".3" stroke="none" />
  </>
)

// ไม้เสียบทำให้อ่านออกทันทีว่าเป็นไส้กรอก ไม่งั้นแคปซูลเปล่าดูเหมือนพลาสเตอร์ปิดแผล
const sausage: Glyph = (c) => (
  <g transform="rotate(-30 16 16)">
    <path d="M2.4 16h6.2" strokeWidth="1.9" />
    <rect x="7.4" y="11.2" width="20" height="9.6" rx="4.8" fill={W} />
    <path d="m13 13.6-1.7 4.8M18 13.6l-1.7 4.8M23 13.6l-1.7 4.8" strokeWidth="1.2" opacity=".5" />
  </g>
)

const bao: Glyph = (c) => (
  <>
    <path d="M6.2 23c0-6.3 4.4-11 9.8-11s9.8 4.7 9.8 11a1.6 1.6 0 0 1-1.6 1.6H7.8A1.6 1.6 0 0 1 6.2 23z" fill={W} />
    <path d="M12.4 15.8c1.1 1.6 1.4 3.4 1 5.4M19.6 15.8c-1.1 1.6-1.4 3.4-1 5.4" strokeWidth="1.2" opacity=".5" />
    <circle cx="16" cy="13.6" r="1.8" fill={c} opacity=".8" stroke="none" />
  </>
)

const egg: Glyph = (c) => (
  <>
    <path d="M16 4.6c4.4 0 8 6 8 11.6s-3.6 9.6-8 9.6-8-4-8-9.6S11.6 4.6 16 4.6z" fill={W} />
    <ellipse cx="16" cy="17.4" rx="3.5" ry="3.3" fill={c} opacity=".8" stroke="none" />
  </>
)

const sandwich: Glyph = (c) => (
  <>
    <path d="M16 7.2 29.4 25.3a1.5 1.5 0 0 1-1.2 2.1H3.8a1.5 1.5 0 0 1-1.2-2.1z" fill={W} />
    <path d="M5.4 21.6h21.2l2.8 3.7a1.5 1.5 0 0 1-1.2 2.1H3.8a1.5 1.5 0 0 1-1.2-2.1z" fill={c} opacity=".85" stroke="none" />
    <path d="M9.2 16.4h13.6" strokeWidth="1.3" opacity=".45" />
    <path d="M7 19.9c1.9-2.2 3.8 2.2 5.7 0s3.8 2.2 5.7 0 3.8 2.2 5.7 0" strokeWidth="1.4" />
  </>
)

const bowl: Glyph = (c) => (
  <>
    <path d="M13.4 9.4c.9-1.1.9-2.3 0-3.4M18.6 9.4c.9-1.1.9-2.3 0-3.4" strokeWidth="1.2" opacity=".55" />
    <path d="M4.6 13.4h22.8c0 6.2-5.1 11.2-11.4 11.2S4.6 19.6 4.6 13.4z" fill={W} />
    <path d="M4.6 13.4h22.8v2.8H4.6z" fill={c} opacity=".85" stroke="none" />
  </>
)

// ── ของใช้ ───────────────────────────────────────────────────────────────
const toothpaste: Glyph = (c) => (
  <>
    <rect x="14.4" y="3" width="3.2" height="3.4" rx="1" fill={c} stroke="none" />
    <path d="M13 6.4h6l1.6 3.2c.6 1.2.9 2.5.9 3.8V26a1.6 1.6 0 0 1-1.6 1.6h-7.8A1.6 1.6 0 0 1 10.5 26V13.4c0-1.3.3-2.6.9-3.8z" fill={W} />
    <path d="M10.5 14.8h11v4.8h-11z" fill={c} opacity=".85" stroke="none" />
    <path d="M10.5 24.6h11v1.4a1.6 1.6 0 0 1-1.6 1.6h-7.8A1.6 1.6 0 0 1 10.5 26z" fill={c} opacity=".45" stroke="none" />
  </>
)

const toothbrush: Glyph = (c) => (
  <g transform="rotate(-32 16 16)">
    <rect x="4.4" y="14.4" width="16.6" height="3.3" rx="1.6" fill={W} />
    <rect x="19.4" y="13.2" width="8" height="5.6" rx="2.6" fill={W} />
    <path d="M21.2 13.2v-3.1M23.4 13.2v-3.1M25.6 13.2v-3.1" strokeWidth="1.4" />
    <path d="M8.4 16h6" strokeWidth="1.1" opacity=".45" />
  </g>
)

const soap: Glyph = (c) => (
  <>
    <rect x="5" y="13.6" width="17.6" height="10.4" rx="4.6" fill={W} />
    <path d="M8.8 17.4c1.7-1.3 3.5-1.5 5.4-.7" strokeWidth="1.3" opacity=".55" />
    <circle cx="23" cy="8.8" r="3" fill={c} opacity=".3" stroke="none" />
    <circle cx="27.2" cy="13" r="1.9" fill={c} opacity=".3" stroke="none" />
    <circle cx="18.2" cy="6.4" r="1.6" fill={c} opacity=".3" stroke="none" />
  </>
)

const shampoo: Glyph = (c) => (
  <>
    <rect x="8" y="6.4" width="16" height="19.2" rx="1.6" fill={W} />
    <path d="M8 10.2h16M8 21.8h16" strokeWidth="1.1" opacity=".4" strokeDasharray="2 1.7" />
    <path d="M16 12.6c2.5 2.7 3.7 4.5 3.7 6.1a3.7 3.7 0 0 1-7.4 0c0-1.6 1.2-3.4 3.7-6.1z" fill={c} opacity=".85" stroke="none" />
  </>
)

const detergent: Glyph = (c) => (
  <>
    <path d="M6 9.4h20v17a1.6 1.6 0 0 1-1.6 1.6H7.6A1.6 1.6 0 0 1 6 26.4z" fill={W} />
    <path d="m6 9.4 2.4-3h15.2l2.4 3z" fill={c} opacity=".85" stroke="none" />
    <path d="M6 15.8h20v5.6H6z" fill={c} opacity=".2" stroke="none" />
    <circle cx="11.6" cy="19" r="2.7" fill={W} strokeWidth="1.2" />
    <circle cx="17.4" cy="20.2" r="1.8" fill={W} strokeWidth="1.1" />
    <circle cx="20.8" cy="17.4" r="1.2" fill={W} strokeWidth="1" />
  </>
)

const tissue: Glyph = (c) => (
  <>
    <path d="M24.8 15.6h3.6v9.2l-1.8-1.5-1.8 1.5z" fill={W} />
    <path d="M7 11.6c0-1.6 4-2.9 9-2.9s9 1.3 9 2.9v11c0 1.6-4 2.9-9 2.9s-9-1.3-9-2.9z" fill={W} />
    <ellipse cx="16" cy="11.6" rx="9" ry="2.9" fill={W} />
    <ellipse cx="16" cy="11.6" rx="2.7" ry="1" fill={c} opacity=".7" stroke="none" />
  </>
)

const wipes: Glyph = (c) => (
  <>
    <rect x="4.6" y="10" width="22.8" height="13.2" rx="3" fill={W} />
    <rect x="10" y="7.6" width="12" height="5.4" rx="2.6" fill={c} opacity=".85" stroke="none" />
    <path d="M8 17.4h6M8 20.2h9.4" strokeWidth="1.2" opacity=".45" />
  </>
)

const spray: Glyph = (c) => (
  <>
    <path d="M12.6 3.6h5.2v3.6h-5.2z" fill={W} />
    <path d="M18 4.4h4.4M18 6.6h3" strokeWidth="1.2" opacity=".5" />
    <path d="M10.8 12.4c0-2.6 1.8-3.6 1.8-5.2h5.2c0 1.6 1.8 2.6 1.8 5.2v13.6a2 2 0 0 1-2 2h-4.8a2 2 0 0 1-2-2z" fill={W} />
    <path d="M10.8 16.4h8.8v6.2h-8.8z" fill={c} opacity=".85" stroke="none" />
  </>
)

// ── บุหรี่ / อื่นๆ ───────────────────────────────────────────────────────
const battery: Glyph = (c) => (
  <>
    <rect x="13.5" y="2.6" width="5" height="2.8" rx="1" fill={c} stroke="none" />
    <rect x="9" y="5.2" width="14" height="22.2" rx="2.2" fill={W} />
    <path d="M9 16.8h14v8.4a2.2 2.2 0 0 1-2.2 2.2h-9.6A2.2 2.2 0 0 1 9 25.2z" fill={c} opacity=".85" stroke="none" />
    <path d="m17.2 8-3.6 4.9h2.4l-.9 2.6 3.5-4.8h-2.4z" fill={c} opacity=".8" stroke="none" />
  </>
)

const cigarette: Glyph = (c) => (
  <>
    <g transform="rotate(14 19 8)">
      <rect x="17.3" y="3.2" width="3.4" height="7.6" rx="1.2" fill={W} />
      <path d="M17.3 7.8h3.4v1.8a1.2 1.2 0 0 1-1.2 1.2h-1a1.2 1.2 0 0 1-1.2-1.2z" fill={c} opacity=".8" stroke="none" />
    </g>
    <rect x="8" y="9.2" width="16" height="18.6" rx="1.8" fill={W} />
    <path d="M8 14h16" strokeWidth="1.1" opacity=".45" />
    <path d="M8 19.4h16v4.4H8z" fill={c} opacity=".85" stroke="none" />
  </>
)

const lighter: Glyph = (c) => (
  <>
    <path d="M16 1.6c2.4 2.4 3.2 3.8 3.2 5.1a3.2 3.2 0 0 1-6.4 0c0-1.3.8-2.7 3.2-5.1z" fill={c} opacity=".9" stroke="none" />
    <path d="M11.6 8.8h8.8v2.6h-8.8z" fill={c} opacity=".85" stroke="none" />
    <rect x="10" y="11.2" width="12" height="16.6" rx="2.4" fill={W} />
    <rect x="13" y="16.6" width="6" height="7.2" rx="1" fill={c} opacity=".25" stroke="none" />
  </>
)

const trash: Glyph = (c) => (
  <>
    <path d="M12.8 12.6c-1-2.4-.4-4.7 1.5-5.8M19.2 12.6c1-2.4.4-4.7-1.5-5.8" strokeWidth="1.4" />
    <path d="M9 12.4h14l1.5 12.4a2.4 2.4 0 0 1-2.4 2.7H9.9a2.4 2.4 0 0 1-2.4-2.7z" fill={W} />
    <path d="M8.2 18.2h15.6l.6 5.4H7.6z" fill={c} opacity=".22" stroke="none" />
  </>
)

const box: Glyph = (c) => (
  <>
    <path d="M5.6 10.4h20.8v15.2a2 2 0 0 1-2 2H7.6a2 2 0 0 1-2-2z" fill={W} />
    <path d="M4.4 6.4h23.2v4H4.4z" fill={c} opacity=".85" stroke="none" />
    <path d="M13 16h6" strokeWidth="1.4" opacity=".5" />
  </>
)

const ART = {
  water, soda, cola, tea, coffee, yogurt, energy, juice, drink,
  chips, wafer, fish, seaweed, candy, biscuit, bread,
  noodle, cupNoodle, bento, sausage, bao, egg, sandwich, bowl,
  toothpaste, toothbrush, soap, shampoo, detergent, tissue, wipes, spray,
  battery, cigarette, lighter, trash, box,
} satisfies Record<string, Glyph>

export type ArtKey = keyof typeof ART

/**
 * จับคู่ชื่อสินค้า → ภาพ เรียงจาก "เฉพาะเจาะจงที่สุด" ลงมา
 *
 * ⚠️ ภาษาไทยไม่เว้นวรรค คำสั้นจึงไปโผล่ซ้อนในคำอื่นได้ เช่น "ขนมปัง" มี "นม"
 * อยู่ข้างใน (ข-นม-ปัง) กฎที่ใช้คำสั้นจึงต้องยึดหัวคำด้วย \b หรือ (^|\s)
 * และลำดับก็สำคัญ — "บิสกิตแซนวิชครีม" ต้องเจอ biscuit ก่อน sandwich,
 * "ผ้าเปียก" ต้องเจอ wipes ก่อน tissue
 */
const RULES: [RegExp, ArtKey][] = [
  [/น้ำอัดลม|โคล่า|โค้ก|เป๊ปซี่|cola|coke/i, 'cola'],
  [/โซดา|soda/i, 'soda'],
  [/(^|\s)ชา|ชาเขียว|ชาดำ|ชาเย็น|ชานม|ชามะนาว|ชาไทย|\btea\b/i, 'tea'],
  [/กาแฟ|coffee|latte/i, 'coffee'],
  [/นมเปรี้ยว|โยเกิร์ต|ยาคูลท์|yogurt/i, 'yogurt'],
  [/ชูกำลัง|กระทิงแดง|เอ็ม-?150|energy/i, 'energy'],
  [/น้ำผลไม้|น้ำส้ม|น้ำองุ่น|(^|\s)นม|นมสด|นมจืด|นมกล่อง|juice|milk/i, 'juice'],
  [/น้ำดื่ม|น้ำเปล่า|น้ำแร่|water/i, 'water'],

  [/มันฝรั่ง|ข้าวเกรียบ|ขนมถุง|สแน็ค|chips|snack/i, 'chips'],
  [/เวเฟอร์|ช็อกโกแลต|ช็อคโกแลต|wafer|chocolate/i, 'wafer'],
  [/ปลาเส้น|ปลาหมึก|ปลาสวรรค์|ปลาแผ่น/, 'fish'],
  [/สาหร่าย|seaweed/i, 'seaweed'],
  [/ลูกอม|ท็อฟฟี่|หมากฝรั่ง|มินต์|candy|mint/i, 'candy'],
  [/บิสกิต|คุกกี้|แครกเกอร์|biscuit|cookie/i, 'biscuit'],
  [/ขนมปัง|สังขยา|เบเกอรี่|bread/i, 'bread'],

  [/บะหมี่|มาม่า|ก๋วยเตี๋ยว|noodle/i, 'noodle'],
  [/โจ๊ก|คัพ|ถ้วย|\bcup\b/i, 'cupNoodle'],
  [/ข้าวกล่อง|อาหารกล่อง|ข้าว|bento/i, 'bento'],
  [/ไส้กรอก|ฮอทดอก|sausage/i, 'sausage'],
  [/ซาลาเปา|เปา|bao/i, 'bao'],
  [/ไข่|egg/i, 'egg'],
  [/แซนด์วิช|แซนวิช|sandwich|burger/i, 'sandwich'],

  [/ยาสีฟัน|toothpaste/i, 'toothpaste'],
  [/แปรงสีฟัน|แปรง|toothbrush/i, 'toothbrush'],
  [/สบู่|soap/i, 'soap'],
  [/แชมพู|ครีมนวด|shampoo/i, 'shampoo'],
  [/ผงซักฟอก|น้ำยาซักผ้า|ซักผ้า|detergent/i, 'detergent'],
  [/ผ้าเปียก|ทิชชู่เปียก|ทิชชูเปียก|wipe/i, 'wipes'],
  [/ทิชชู่|ทิชชู|กระดาษชำระ|กระดาษ|tissue/i, 'tissue'],
  [/สเปรย์|น้ำยา|spray|cleaner/i, 'spray'],

  [/ถ่าน|แบตเตอรี่|battery/i, 'battery'],
  [/บุหรี่|ยาเส้น|cigarette/i, 'cigarette'],
  [/ไฟแช็ก|ไฟแช็ค|lighter/i, 'lighter'],
  [/ถุงขยะ|ถุงดำ|trash|garbage/i, 'trash'],
]

/** ภาพประจำหมวด ใช้เมื่อชื่อสินค้าไม่ตรงกฎไหนเลย (เช่น สินค้าใหม่ที่เพิ่งเพิ่ม) */
const BY_CAT: Record<string, ArtKey> = {
  เครื่องดื่ม: 'drink',
  ขนม: 'chips',
  อาหาร: 'bowl',
  ของใช้: 'spray',
  'บุหรี่/อื่นๆ': 'box',
}

export function artKeyFor(name: string, cat = ''): ArtKey {
  for (const [re, key] of RULES) if (re.test(name)) return key
  return BY_CAT[cat] ?? 'box'
}

export function ProductArt({
  name,
  cat,
  size = 40,
  color,
}: {
  name: string
  cat: string
  size?: number
  color?: string
}) {
  const c = color ?? TILES[cat]?.[1] ?? TILES['บุหรี่/อื่นๆ'][1]
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      stroke={c}
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      style={{ display: 'block', flexShrink: 0 }}
    >
      {ART[artKeyFor(name, cat)](c)}
    </svg>
  )
}
