// Local-first store (IndexedDB via Dexie). The Cashier reads/writes here first
// so selling never waits on the network; the sync engine reconciles with the
// server in the background.
import Dexie, { type Table } from 'dexie'

export interface LocalProduct {
  id: string
  name: string
  barcode: string | null
  category: string
  cost_price: number
  sell_price: number
  is_active: boolean
  qty_on_hand: number
  reorder_point: number
}

export interface PendingOp {
  localId?: number
  type: 'checkout'
  client_uuid: string
  payload: unknown
  status: 'pending' | 'error'
  error?: string
  createdAt: number
}

export interface LocalBill {
  client_uuid: string
  server_id?: string
  bill_no?: string
  total: number
  paid: number
  change: number
  payment_method: string
  items: { name: string; qty: number; price: number }[]
  created_at: number
  synced: boolean
}

export interface Meta {
  key: string
  value: unknown
}

class MiniDB extends Dexie {
  products!: Table<LocalProduct, string>
  pending!: Table<PendingOp, number>
  bills!: Table<LocalBill, string>
  meta!: Table<Meta, string>

  constructor() {
    super('minimart-pos')
    this.version(1).stores({
      products: 'id, barcode, category, is_active',
      pending: '++localId, status, client_uuid',
      bills: 'client_uuid, server_id, synced, created_at',
      meta: 'key',
    })
  }
}

export const db = new MiniDB()

export async function setMeta(key: string, value: unknown) {
  await db.meta.put({ key, value })
}
export async function getMeta<T>(key: string): Promise<T | undefined> {
  return (await db.meta.get(key))?.value as T | undefined
}
