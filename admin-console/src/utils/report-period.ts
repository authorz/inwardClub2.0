export interface ReportPeriod {
  from: string
  to: string
}

export type ReportPreset = 'today' | 'yesterday' | 'lastMonth'

export function reportToday(now = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
  }).formatToParts(now)
  return ['year', 'month', 'day'].map((type) => parts.find((part) => part.type === type)!.value).join('-')
}

export function reportPreset(preset: ReportPreset, now = new Date()): ReportPeriod {
  const today = reportToday(now)
  const date = new Date(`${today}T00:00:00Z`)
  if (preset === 'yesterday') date.setUTCDate(date.getUTCDate() - 1)
  if (preset === 'lastMonth') {
    date.setUTCDate(0)
    const to = date.toISOString().slice(0, 10)
    date.setUTCDate(1)
    return { from: date.toISOString().slice(0, 10), to }
  }
  const day = date.toISOString().slice(0, 10)
  return { from: day, to: day }
}

export function periodDays(period: ReportPeriod): number {
  return Math.round((Date.parse(`${period.to}T00:00:00Z`) - Date.parse(`${period.from}T00:00:00Z`)) / 86_400_000) + 1
}
