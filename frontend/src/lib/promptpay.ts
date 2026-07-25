// Build a PromptPay (Thai QR Payment / EMVCo) payload string from a PromptPay
// target (mobile no. / national ID / e-wallet) and an amount in baht.
// The result is what a banking app scans to pay.

function tlv(id: string, value: string): string {
  return id + String(value.length).padStart(2, '0') + value
}

// CRC-16/CCITT-FALSE (poly 0x1021, init 0xFFFF) over the payload incl. "6304".
function crc16(data: string): string {
  let crc = 0xffff
  for (let i = 0; i < data.length; i++) {
    crc ^= data.charCodeAt(i) << 8
    for (let j = 0; j < 8; j++) {
      crc = crc & 0x8000 ? (crc << 1) ^ 0x1021 : crc << 1
      crc &= 0xffff
    }
  }
  return crc.toString(16).toUpperCase().padStart(4, '0')
}

function formatTarget(id: string): { sub: string; value: string } {
  const s = id.replace(/[^0-9]/g, '')
  if (s.length >= 15) return { sub: '03', value: s } // e-wallet
  if (s.length >= 13) return { sub: '02', value: s } // national / tax id
  // mobile: 0812345678 -> 0066812345678 (13 digits)
  const phone = '66' + s.replace(/^0/, '')
  return { sub: '01', value: ('0000000000000' + phone).slice(-13) }
}

export function promptPayPayload(promptPayId: string, amountBaht: number): string {
  const { sub, value } = formatTarget(promptPayId)
  const merchant = tlv('00', 'A000000677010111') + tlv(sub, value) // AID + target
  const hasAmount = amountBaht > 0
  let payload =
    tlv('00', '01') + // payload format indicator
    tlv('01', hasAmount ? '12' : '11') + // dynamic (with amount) vs static
    tlv('29', merchant) + // PromptPay merchant account info
    tlv('53', '764') + // currency THB
    (hasAmount ? tlv('54', amountBaht.toFixed(2)) : '') +
    tlv('58', 'TH') // country
  payload += '6304' // CRC tag + length; CRC is appended over this prefix
  return payload + crc16(payload)
}

export function isValidPromptPayId(id: string): boolean {
  const s = id.replace(/[^0-9]/g, '')
  return s.length === 10 || s.length === 13 || s.length === 15
}
