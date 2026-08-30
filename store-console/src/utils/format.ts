/**
 * 集中式格式化工具：金额（整数分）、手机号掩码、日期时间。
 * 全站禁止各页面重复实现相同格式化逻辑。
 */

/** 整数分 -> 人民币显示，如 8800 => ¥88.00。 */
export function formatCent(cent: number | null | undefined): string {
  if (cent == null || Number.isNaN(cent)) return '-'
  const yuan = cent / 100
  return `¥${yuan.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

/** 整数分 -> 图表紧凑金额，例如 1280000 => ¥1.3万。 */
export function formatCompactCent(cent: number): string {
  const yuan = cent / 100
  if (yuan >= 10_000) return `¥${(yuan / 10_000).toFixed(1)}万`
  return `¥${yuan.toFixed(yuan >= 100 ? 0 : 2)}`
}

/** 整数分 -> 纯数字元字符串（用于输入框回填）。 */
export function centToYuan(cent: number | null | undefined): number {
  if (cent == null || Number.isNaN(cent)) return 0
  return cent / 100
}

/** 元 -> 整数分（用于提交）。 */
export function yuanToCent(yuan: number | null | undefined): number {
  if (yuan == null || Number.isNaN(yuan)) return 0
  return Math.round(yuan * 100)
}

/**
 * 手机号掩码。若服务端已返回掩码则原样展示；否则本地掩码中间四位。
 * 门店后台默认展示掩码，保护会员隐私。
 */
export function maskPhone(phone: string | null | undefined): string {
  if (!phone) return '-'
  if (phone.includes('*')) return phone
  const prefix = phone.trim().startsWith('+') ? '+' : ''
  const digits = phone.replace(/\D/g, '')
  if (digits.length < 7) return phone
  return `${prefix}${digits.slice(0, 3)}****${digits.slice(-4)}`
}

/** ISO 时间 -> 本地日期时间展示。 */
export function formatDateTime(value: string | number | Date | null | undefined): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(
    d.getMinutes(),
  )}`
}

/** 剩余秒数 -> mm:ss 倒计时展示。 */
export function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const m = Math.floor(s / 60)
  const rest = s % 60
  return `${String(m).padStart(2, '0')}:${String(rest).padStart(2, '0')}`
}
