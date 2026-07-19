// Money is integer satang everywhere. Convert to baht for DISPLAY ONLY.
export function baht(satang: number): string {
  return (
    '฿' +
    (satang / 100).toLocaleString('th-TH', { minimumFractionDigits: 0, maximumFractionDigits: 2 })
  )
}

export function bahtPlain(satang: number): string {
  return (satang / 100).toLocaleString('th-TH', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
