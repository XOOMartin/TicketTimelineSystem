import Badge from 'components/atoms/Badge'
import PhaseTimeline from 'components/molecules/PhaseTimeline'
import useOptions from 'hooks/useOptions'
import { formatDateTime, formatHours } from 'lib/dates'
import { TicketRow } from 'types/worklog'

import styles from 'styles/Table.module.css'

type Props = {
  rows: TicketRow[]
  periodLabel: string // e.g. "This week"
  showAssignees?: boolean
}

export default function TicketTable(props: Props) {
  const { rows, periodLabel, showAssignees = false } = props
  const { options, phaseLabel, statusLabel } = useOptions()

  if (rows.length === 0) {
    return <div className={`${styles.wrap} ${styles.empty}`}>No tickets were worked on in this period.</div>
  }

  return (
    <div className={styles.wrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>Ticket</th>
            <th>Title</th>
            {showAssignees && <th>Assignees</th>}
            <th>Task</th>
            <th>Status</th>
            <th className={styles.num}>{periodLabel} (h)</th>
            <th className={styles.num}>Total (h)</th>
            <th>Phases</th>
            <th className={styles.num} title="Times a review sent the ticket back">
              Revisions
            </th>
            <th>Last update</th>
            <th>Notes</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((t) => (
            <tr key={t.ticketKey}>
              <td className={styles.key}>{t.ticketKey}</td>
              <td>{t.title}</td>
              {showAssignees && <td className={styles.names}>{t.assignees.join(', ')}</td>}
              <td className={styles.task}>{phaseLabel(t.phase)}</td>
              <td>
                <Badge value={t.status}>{statusLabel(t.status)}</Badge>
              </td>
              <td className={styles.num}>{formatHours(t.periodHours)}</td>
              <td className={styles.num}>{formatHours(t.totalHours)}</td>
              <td>
                <PhaseTimeline phases={options?.phases ?? []} hours={t.phaseHours} />
              </td>
              <td className={styles.num}>{t.reviewBacks}</td>
              <td className={styles.nowrap}>{formatDateTime(t.lastUpdate)}</td>
              <td className={styles.muted}>{t.note}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
