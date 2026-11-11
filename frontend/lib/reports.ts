import { addDays, formatDay, formatShortDay, mondayOf, today } from 'lib/dates'
import { ReportKind, ReportSummary } from 'types/worklog'

// How each kind of report is named: on its chip, its view and the dashboard
export const REPORT_KIND_LABELS: Record<ReportKind, string> = {
  daily: 'Daily',
  weekly: 'Weekly',
}

// The days a report covers: one day by itself, a range as first – last
export function reportDays(report: Pick<ReportSummary, 'period'>, long = false): string {
  const format = long ? formatDay : formatShortDay
  const { first, last } = report.period
  return first === last ? format(first) : `${format(first)} – ${format(last)}`
}

// A report being picked: its kind and its first and last day
export type ReportRange = { kind: ReportKind; start: string; end: string }

// The days a kind starts on: this working week (Monday to Friday, no
// weekend), or today
export function defaultReportRange(kind: ReportKind): ReportRange {
  if (kind === 'weekly') {
    const monday = mondayOf(today())
    return { kind, start: monday, end: addDays(monday, 4) }
  }
  return { kind, start: today(), end: today() }
}

// Whether the days can make a report: both picked, in order, not starting
// in the future
export function isValidRange({ start, end }: ReportRange): boolean {
  return Boolean(start && end && start <= end && start <= today())
}
