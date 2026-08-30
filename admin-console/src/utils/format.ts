/**
 * 展示格式化工具（集中定义）。
 *
 * 约定（依据实现规格 5.1）：
 *  - 金额字段一律为整数分（*Cent），展示时转元。
 *  - 时间为 RFC3339 UTC，展示时转本地可读格式。
 * 页面不得各自手写金额/时间格式化逻辑。
 */

/** 整数分 -> 人民币展示字符串，例：12345 -> "¥123.45" */
export function formatCent(cent: number | null | undefined, withSymbol = true): string {
  if (cent == null || Number.isNaN(cent)) return '-'
  const yuan = (cent / 100).toFixed(2)
  return withSymbol ? `¥${yuan}` : yuan
}

/** 元 -> 整数分（表单提交时使用），非法输入返回 null */
export function toCent(yuan: number | string | null | undefined): number | null {
  if (yuan == null || yuan === '') return null
  const n = typeof yuan === 'string' ? Number(yuan) : yuan
  if (Number.isNaN(n)) return null
  return Math.round(n * 100)
}

/** RFC3339 / ISO 时间 -> 本地 "YYYY-MM-DD HH:mm:ss" */
export function formatDateTime(value: string | number | Date | null | undefined): string {
  if (value == null || value === '') return '-'
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  )
}

/** RFC3339 / ISO 时间 -> 本地 "YYYY-MM-DD" */
export function formatDate(value: string | number | Date | null | undefined): string {
  if (value == null || value === '') return '-'
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 手机号掩码展示，例：138****8000 */
export function maskPhone(phone: string | null | undefined): string {
  if (!phone) return '-'
  if (phone.includes('*')) return phone
  const prefix = phone.trim().startsWith('+') ? '+' : ''
  const digits = phone.replace(/\D/g, '')
  if (digits.length < 7) return phone
  return `${prefix}${digits.slice(0, 3)}****${digits.slice(-4)}`
}
