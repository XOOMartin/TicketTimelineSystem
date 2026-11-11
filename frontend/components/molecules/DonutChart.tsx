import { Segment } from 'lib/chartData'

import styles from 'styles/DonutChart.module.css'

const SIZE = 112
const STROKE = 14
const RADIUS = (SIZE - STROKE) / 2
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

type Props = {
  title: string
  segments: Segment[]
  // Shown under the total in the middle, e.g. "hours"
  unit: string
  format?: (value: number) => string
}

// A ring split by share of the total, with a legend beside it.
export default function DonutChart(props: Props) {
  const { title, segments, unit, format = String } = props

  const shown = segments.filter((s) => s.value > 0)
  const total = shown.reduce((sum, s) => sum + s.value, 0)

  let offset = 0
  const arcs = shown.map((s) => {
    const length = (s.value / total) * CIRCUMFERENCE
    const arc = (
      <circle
        key={s.key}
        cx={SIZE / 2}
        cy={SIZE / 2}
        r={RADIUS}
        stroke={s.color}
        strokeWidth={STROKE}
        strokeDasharray={`${length} ${CIRCUMFERENCE - length}`}
        strokeDashoffset={-offset}
        fill="none"
      >
        <title>{`${s.label}: ${format(s.value)}`}</title>
      </circle>
    )
    offset += length
    return arc
  })

  return (
    <figure className={styles.chart}>
      <figcaption className={styles.title}>{title}</figcaption>
      <div className={styles.body}>
        <div className={styles.ring}>
          <svg
            width={SIZE}
            height={SIZE}
            viewBox={`0 0 ${SIZE} ${SIZE}`}
            role="img"
            aria-label={shown.map((s) => `${s.label} ${format(s.value)}`).join(', ') || 'No data'}
          >
            <circle
              cx={SIZE / 2}
              cy={SIZE / 2}
              r={RADIUS}
              stroke="var(--surface-2)"
              strokeWidth={STROKE}
              fill="none"
            />
            <g transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}>{arcs}</g>
          </svg>
          <div className={styles.center}>
            <span className={styles.total}>{format(total)}</span>
            <span className={styles.unit}>{unit}</span>
          </div>
        </div>
        {shown.length === 0 ? (
          <p className={styles.empty}>No data yet</p>
        ) : (
          <ul className={styles.legend}>
            {shown.map((s) => (
              <li key={s.key}>
                <i style={{ background: s.color }} />
                <span className={styles.label}>{s.label}</span>
                <span className={styles.value}>{format(s.value)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </figure>
  )
}
