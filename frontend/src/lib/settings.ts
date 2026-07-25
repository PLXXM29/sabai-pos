// Per-device POS settings (localStorage). The shop's PromptPay ID lives here so
// the transfer QR can be generated offline. For multi-device shops this can move
// to backend store.config later.
const PROMPTPAY_KEY = 'sabai-pos-promptpay-id'

export function getPromptPayId(): string {
  try {
    return localStorage.getItem(PROMPTPAY_KEY) ?? ''
  } catch {
    return ''
  }
}

export function setPromptPayId(id: string): void {
  try {
    localStorage.setItem(PROMPTPAY_KEY, id.trim())
  } catch {
    /* ignore */
  }
}

// Auto-confirm transfer via a payment webhook. Default OFF: the cashier confirms
// manually (fine for iPhone-only shops). Turn ON only if a forwarder is wired.
const AUTO_KEY = 'sabai-pos-auto-confirm'

export function getAutoConfirm(): boolean {
  try {
    return localStorage.getItem(AUTO_KEY) === '1'
  } catch {
    return false
  }
}

export function setAutoConfirm(on: boolean): void {
  try {
    localStorage.setItem(AUTO_KEY, on ? '1' : '0')
  } catch {
    /* ignore */
  }
}
