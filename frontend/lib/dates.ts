// Dates travel as YYYY-MM-DD strings in the user's local time zone, matching
// calendar.Day on the server.

const pad = (n: number) => String(n).padStart(2, '0')

export function toDay(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function toClock(d: Date): string {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export const today = () => toDay(new Date())

// Parses YYYY-MM-DD as local midnight (new Date('2026-10-30') would be UTC).
export function fromDay(day: string): Date {
  const [y, m, d] = day.split('-').map(Number)
  return new Date(y, m - 1, d)
}

export function addDays(day: string, n: number): string {
  const d = fromDay(day)
  d.setDate(d.getDate() + n)
  return toDay(d)
}

// The Monday of the week the day falls in.
export function mondayOf(day: string): string {
  const offset = (fromDay(day).getDay() + 6) % 7
  return addDays(day, -offset)
}

export function formatDay(day: string): string {
  return fromDay(day).toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString([], {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// Combines a date and time input into an ISO timestamp.
export function toISO(day: string, clock: string): string {
  const [h, min] = clock.split(':').map(Number)
  const d = fromDay(day)
  d.setHours(h, min)
  return d.toISOString()
}

export function formatHours(h: number): string {
  return Number.isInteger(h) ? `${h}` : h.toFixed(2).replace(/0$/, '')
}

// e.g. "2 h 30 m", as written in the team's progress sheets
export function formatHoursMinutes(h: number): string {
  const minutes = Math.round(h * 60)
  return `${Math.floor(minutes / 60)} h ${pad(minutes % 60)} m`
}

// e.g. "2h", "30m", "1h 16m", for tight table cells
export function formatShortHours(h: number): string {
  const minutes = Math.round(h * 60)
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  if (hours === 0) return `${rest}m`
  return rest === 0 ? `${hours}h` : `${hours}h ${rest}m`
}

// A running clock, e.g. "0:05:09" or "1:23:45"
export function formatClock(ms: number): string {
  const seconds = Math.max(0, Math.floor(ms / 1000))
  return `${Math.floor(seconds / 3600)}:${pad(Math.floor(seconds / 60) % 60)}:${pad(seconds % 60)}`
}

// The time of day of a timestamp, e.g. "09:32 AM", in the user's locale
export function formatTimeOfDay(iso: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// e.g. "Nov 3"
export function formatShortDay(day: string): string {
  return fromDay(day).toLocaleDateString([], { month: 'short', day: 'numeric' })
}

// How long ago a moment was, briefly: "just now", "5m ago", "3h ago",
// "yesterday", "4d ago", then the day itself
export function formatTimeAgo(iso: string, now = Date.now()): string {
  const minutes = Math.floor((now - new Date(iso).getTime()) / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days}d ago`
  return formatShortDay(toDay(new Date(iso)))
}
