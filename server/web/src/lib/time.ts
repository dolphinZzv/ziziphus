import i18n from '@/i18n'

// Server timestamps may be seconds or milliseconds — normalize
function toDate(ts: number): Date {
  if (!ts || ts <= 0) return new Date(0)
  // Server uses UnixMilli (13 digits), but some legacy fields may use seconds (10 digits).
  // Treat anything ≤ 10 digits as seconds; anything 13+ as milliseconds.
  // Values with 11–12 digits are ambiguous — they're within a few decades of epoch in ms,
  // so treat them as milliseconds.
  if (ts < 100_000_000_000) return new Date(ts * 1000)       // ≤10 digits → seconds
  if (ts < 1_000_000_000_000) return new Date(ts)             // 11–12 digits → ms (near epoch)
  return new Date(ts)                                          // 13+ digits → ms (post-2001)
}

export function formatTime(ts: number): string {
  const d = toDate(ts)
  if (d.getTime() <= 0) return ''
  const h = d.getHours().toString().padStart(2, '0')
  const m = d.getMinutes().toString().padStart(2, '0')
  return `${h}:${m}`
}

export function formatMessageTime(ts: number): string {
  const d = toDate(ts)
  if (d.getTime() <= 0) return ''
  const now = Date.now()
  const diff = now - d.getTime()
  const secs = Math.floor(diff / 1000)
  if (secs < 60) return i18n.t('time.justNow')
  if (secs < 3600) return i18n.t('time.minuteAgo', { n: Math.floor(secs / 60) })
  if (secs < 86400) {
    const h = d.getHours().toString().padStart(2, '0')
    const m = d.getMinutes().toString().padStart(2, '0')
    return `${h}:${m}`
  }
  const today = new Date(now)
  today.setHours(0, 0, 0, 0)
  const yesterday = new Date(today.getTime() - 86400000)
  const msgDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  if (msgDay.getTime() >= yesterday.getTime() && msgDay.getTime() < today.getTime()) return i18n.t('common.yesterday')
  if (d.getFullYear() === today.getFullYear()) {
    return `${(d.getMonth() + 1).toString().padStart(2, '0')}/${d.getDate().toString().padStart(2, '0')}`
  }
  return `${d.getFullYear()}/${(d.getMonth() + 1).toString().padStart(2, '0')}/${d.getDate().toString().padStart(2, '0')}`
}

export function getDateLabel(ts: number): string {
  const d = toDate(ts)
  if (d.getTime() <= 0) return ''
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 86400000)
  const msgDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())

  if (msgDay.getTime() >= today.getTime()) return i18n.t('common.today')
  if (msgDay.getTime() >= yesterday.getTime()) return i18n.t('common.yesterday')
  return `${d.getFullYear()}/${(d.getMonth() + 1).toString().padStart(2, '0')}/${d.getDate().toString().padStart(2, '0')}`
}

export function isSameDay(ts1: number, ts2: number): boolean {
  if (!ts1 || !ts2) return false
  const d1 = toDate(ts1)
  const d2 = toDate(ts2)
  return d1.getFullYear() === d2.getFullYear() && d1.getMonth() === d2.getMonth() && d1.getDate() === d2.getDate()
}
