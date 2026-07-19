// Typed API client. Access token lives in memory; the refresh token is an
// httpOnly cookie the browser sends automatically. On a 401 we transparently
// refresh once and retry.
const API_BASE = '/api/v1'

let accessToken: string | null = null
export function setAccessToken(t: string | null) {
  accessToken = t
}
export function getAccessToken() {
  return accessToken
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function refresh(): Promise<boolean> {
  const res = await fetch(`${API_BASE}/auth/refresh`, { method: 'POST', credentials: 'include' })
  if (!res.ok) return false
  const data = await res.json()
  accessToken = data.access_token
  return true
}

async function request<T>(path: string, opts: RequestInit = {}, retry = true): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (opts.headers) Object.assign(headers, opts.headers)
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  const res = await fetch(`${API_BASE}${path}`, { ...opts, headers, credentials: 'include' })

  if (res.status === 401 && retry && !path.startsWith('/auth/')) {
    if (await refresh()) return request<T>(path, opts, false)
  }
  if (!res.ok) {
    let msg = res.statusText
    try {
      msg = (await res.json()).error ?? msg
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export type User = { id: string; username: string; role: string; store_id: string }

export type ServerProduct = {
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

export type CheckoutPayload = {
  lines: { product_id: string; qty: number }[]
  payment_method: 'cash' | 'transfer'
  paid: number
  discount: number
  client_uuid: string
}

export type BillResponse = {
  bill: { id: string; bill_no: string; total: number; paid: number; change: number; status: string }
  items: unknown[]
}

export const api = {
  base: API_BASE,
  async login(username: string, password: string) {
    const d = await request<{ access_token: string; user: User }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
    accessToken = d.access_token
    return d.user
  },
  async tryRestore(): Promise<User | null> {
    if (!(await refresh())) return null
    try {
      return (await request<{ user: User }>('/auth/me')).user
    } catch {
      return null
    }
  },
  async logout() {
    try {
      await request('/auth/logout', { method: 'POST' })
    } finally {
      accessToken = null
    }
  },
  async me() {
    return (await request<{ user: User }>('/auth/me')).user
  },
  async listProducts() {
    return (await request<{ products: ServerProduct[] }>('/products')).products
  },
  async createProduct(input: ProductInput) {
    return (await request<{ product: ServerProduct }>('/products', {
      method: 'POST',
      body: JSON.stringify(input),
    })).product
  },
  async updateProduct(id: string, input: ProductInput) {
    return (await request<{ product: ServerProduct }>(`/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    })).product
  },
  async deleteProduct(id: string) {
    await request(`/products/${id}`, { method: 'DELETE' })
  },
  async receiveStock(id: string, qty: number, reason: string) {
    return request(`/products/${id}/receive`, {
      method: 'POST',
      body: JSON.stringify({ qty, reason, client_uuid: crypto.randomUUID() }),
    })
  },
  async checkout(payload: CheckoutPayload) {
    return request<BillResponse>('/bills', { method: 'POST', body: JSON.stringify(payload) })
  },
  // Fetch the receipt HTML WITH auth (a plain tab navigation can't send the
  // Bearer token, so we fetch then open it as a blob).
  async receiptHTML(billID: string, width = 58): Promise<string> {
    const url = `${API_BASE}/bills/${billID}/receipt?format=html&width=${width}`
    const doFetch = () =>
      fetch(url, {
        headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
        credentials: 'include',
      })
    let res = await doFetch()
    if (res.status === 401 && (await refresh())) res = await doFetch()
    return res.text()
  },
  async summary() {
    return request<Summary>('/reports/summary')
  },
  async topProducts(limit = 5) {
    return (await request<{ products: TopProduct[] }>(`/reports/top-products?limit=${limit}`)).products
  },
  async salesDaily(days = 7) {
    return (await request<{ days: DaySales[] }>(`/reports/sales-daily?days=${days}`)).days
  },
}

export type ProductInput = {
  name: string
  barcode: string | null
  category: string
  cost_price: number
  sell_price: number
  reorder_point: number
}

export type Summary = {
  sales_today: number
  bills: number
  profit: number
  margin_pct: number
  avg_per_bill: number
  low_stock: number
}
export type TopProduct = { id: string; name: string; qty_sold: number }
export type DaySales = { day: string; sales: number }

export { request as apiRequest }
