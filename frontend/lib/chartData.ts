import { seriesColor } from 'lib/colors'
import { Option } from 'types/worklog'

export type Segment = {
  key: string
  label: string
  value: number
  color: string
}

// Totals value per key, e.g. hours per phase. Counts items by default.
export function sumBy<T>(items: T[], key: (item: T) => string, value: (item: T) => number = () => 1) {
  const totals: Record<string, number> = {}
  for (const item of items) {
    const k = key(item)
    totals[k] = (totals[k] ?? 0) + value(item)
  }
  return totals
}

// Lists the totals in the order of options (e.g. workflow order), each in
// its own colour.
export function toSegments(totals: Record<string, number>, options: Option[], colors: Record<string, string>): Segment[] {
  return options.map((o, i) => ({
    key: o.value,
    label: o.label,
    value: totals[o.value] ?? 0,
    color: colors[o.value] ?? seriesColor(i),
  }))
}

// For keys without a fixed order or colour, such as projects: largest first.
export function toSeriesSegments(totals: Record<string, number>): Segment[] {
  return Object.entries(totals)
    .sort((a, b) => b[1] - a[1])
    .map(([key, value], i) => ({ key, label: key, value, color: seriesColor(i) }))
}
