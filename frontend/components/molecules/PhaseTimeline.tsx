import { formatHours } from 'lib/dates'
import { Option } from 'types/worklog'

import styles from 'styles/PhaseTimeline.module.css'

// Short names so the timeline fits in a table cell
const SHORT: Record<string, string> = {
  investigation: 'Inv',
  confirmation: 'Conf',
  planning: 'Plan',
  coding: 'Code',
  testing: 'Test',
  review: 'Rev',
}

type Props = {
  phases: Option[] // in workflow order
  hours: Record<string, number>
}

// A stacked bar of the hours spent in each phase of a ticket.
export default function PhaseTimeline(props: Props) {
  const { phases, hours } = props

  const worked = phases.filter((p) => (hours[p.value] ?? 0) > 0)
  const total = worked.reduce((sum, p) => sum + hours[p.value], 0)
  if (total === 0) return null

  return (
    <div className={styles.timeline}>
      <div className={styles.bar} role="img" aria-label={worked.map((p) => `${p.label} ${hours[p.value]}h`).join(', ')}>
        {worked.map((p) => (
          <span
            key={p.value}
            className={`${styles.segment} ${styles[p.value]}`}
            style={{ width: `${(hours[p.value] / total) * 100}%` }}
            title={`${p.label}: ${formatHours(hours[p.value])}h`}
          />
        ))}
      </div>
      <div className={styles.legend}>
        {worked.map((p) => (
          <span key={p.value}>
            <i className={`${styles.dot} ${styles[p.value]}`} />
            {SHORT[p.value] ?? p.label} {formatHours(hours[p.value])}
          </span>
        ))}
      </div>
    </div>
  )
}
