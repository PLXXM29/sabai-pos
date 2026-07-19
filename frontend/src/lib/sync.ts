// Sync engine: pulls the catalog from the server into Dexie, and pushes locally
// queued sales up when a connection is available. Every sale carries a
// client_uuid so a retry (or an offline sale synced later) is idempotent server
// side and never duplicates.
import { api, ApiError, type CheckoutPayload } from './api'
import { db, setMeta, type LocalProduct, type PendingOp } from './db'

export type CartLine = { id: string; name: string; price: number; qty: number }

// Pull the product catalog (with live stock) into the local store.
export async function pullCatalog(): Promise<void> {
  const products = await api.listProducts()
  const rows: LocalProduct[] = products.map((p) => ({
    id: p.id,
    name: p.name,
    barcode: p.barcode,
    category: p.category,
    cost_price: p.cost_price,
    sell_price: p.sell_price,
    is_active: p.is_active,
    qty_on_hand: p.qty_on_hand,
    reorder_point: p.reorder_point,
  }))
  await db.transaction('rw', db.products, db.meta, async () => {
    await db.products.clear()
    await db.products.bulkPut(rows)
    await setMeta('catalogSyncedAt', Date.now())
  })
}

// Record a sale locally (optimistic) and queue it for the server.
export async function enqueueCheckout(
  lines: CartLine[],
  paymentMethod: 'cash' | 'transfer',
  paid: number,
  discount = 0,
): Promise<string> {
  const clientUUID = crypto.randomUUID()
  const subtotal = lines.reduce((s, l) => s + l.price * l.qty, 0)
  const total = subtotal - discount
  const effPaid = paymentMethod === 'transfer' ? total : paid
  const payload: CheckoutPayload = {
    lines: lines.map((l) => ({ product_id: l.id, qty: l.qty })),
    payment_method: paymentMethod,
    paid: effPaid,
    discount,
    client_uuid: clientUUID,
  }

  await db.transaction('rw', db.products, db.pending, db.bills, async () => {
    // Optimistic local stock decrement.
    for (const l of lines) {
      const p = await db.products.get(l.id)
      if (p) await db.products.update(l.id, { qty_on_hand: p.qty_on_hand - l.qty })
    }
    await db.pending.add({
      type: 'checkout',
      client_uuid: clientUUID,
      payload,
      status: 'pending',
      createdAt: Date.now(),
    } as PendingOp)
    await db.bills.put({
      client_uuid: clientUUID,
      total,
      paid: effPaid,
      change: effPaid - total,
      payment_method: paymentMethod,
      items: lines.map((l) => ({ name: l.name, qty: l.qty, price: l.price })),
      created_at: Date.now(),
      synced: false,
    })
  })

  void pushPending()
  return clientUUID
}

let pushing = false

// Drain the pending queue. Network errors leave ops pending (retried later);
// validation/conflict errors mark the op as errored and re-pull the catalog to
// reconcile optimistic local state.
export async function pushPending(): Promise<void> {
  if (pushing || !navigator.onLine) return
  pushing = true
  try {
    const ops = await db.pending.where('status').equals('pending').toArray()
    let needReconcile = false
    for (const op of ops) {
      try {
        const resp = await api.checkout(op.payload as CheckoutPayload)
        await db.transaction('rw', db.pending, db.bills, async () => {
          await db.bills.update(op.client_uuid, {
            server_id: resp.bill.id,
            bill_no: resp.bill.bill_no,
            change: resp.bill.change,
            synced: true,
          })
          if (op.localId != null) await db.pending.delete(op.localId)
        })
      } catch (e) {
        if (e instanceof ApiError && e.status >= 400 && e.status < 500) {
          // Server rejected it (e.g. oversold on another device) — don't retry.
          if (op.localId != null) await db.pending.update(op.localId, { status: 'error', error: e.message })
          needReconcile = true
        }
        // else: network/5xx → leave pending for the next cycle
      }
    }
    if (needReconcile && navigator.onLine) {
      try {
        await pullCatalog()
      } catch {
        /* ignore */
      }
    }
  } finally {
    pushing = false
  }
}

let started = false
export function startSync(): void {
  if (started) return
  started = true
  window.addEventListener('online', () => void pushPending())
  setInterval(() => void pushPending(), 15000)
}
