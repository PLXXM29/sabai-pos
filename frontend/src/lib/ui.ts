import { TILES } from '../data'

export function tileOf(cat: string): [string, string] {
  return TILES[cat] ?? TILES['บุหรี่/อื่นๆ']
}

export function statusOf(qty: number, reorder: number) {
  if (qty <= 0) return { t: 'หมด', bg: '#F8DBD6', col: '#A32617' }
  if (qty <= reorder) return { t: 'ใกล้หมด', bg: '#F9EBCB', col: '#8F6410' }
  return { t: 'มีของ', bg: '#DDEEE3', col: '#256B41' }
}
