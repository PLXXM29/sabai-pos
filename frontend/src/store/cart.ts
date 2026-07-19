import { create } from 'zustand'
import type { LocalProduct } from '../lib/db'

export type CartItem = {
  id: string
  name: string
  price: number // satang
  category: string
  qty: number
  stock: number
}

type CartState = {
  items: CartItem[]
  add: (p: LocalProduct) => void
  inc: (id: string) => void
  dec: (id: string) => void
  remove: (id: string) => void
  clear: () => void
}

export const useCart = create<CartState>((set) => ({
  items: [],
  add: (p) =>
    set((s) => {
      const ex = s.items.find((i) => i.id === p.id)
      if (ex) {
        if (ex.qty >= p.qty_on_hand) return s // don't exceed stock
        return { items: s.items.map((i) => (i.id === p.id ? { ...i, qty: i.qty + 1 } : i)) }
      }
      if (p.qty_on_hand <= 0) return s
      return {
        items: [
          ...s.items,
          { id: p.id, name: p.name, price: p.sell_price, category: p.category, qty: 1, stock: p.qty_on_hand },
        ],
      }
    }),
  inc: (id) =>
    set((s) => ({
      items: s.items.map((i) => (i.id === id && i.qty < i.stock ? { ...i, qty: i.qty + 1 } : i)),
    })),
  dec: (id) =>
    set((s) => ({
      items: s.items.map((i) => (i.id === id ? { ...i, qty: i.qty - 1 } : i)).filter((i) => i.qty > 0),
    })),
  remove: (id) => set((s) => ({ items: s.items.filter((i) => i.id !== id) })),
  clear: () => set({ items: [] }),
}))

export const cartTotal = (items: CartItem[]) => items.reduce((s, i) => s + i.price * i.qty, 0)
export const cartCount = (items: CartItem[]) => items.reduce((s, i) => s + i.qty, 0)
