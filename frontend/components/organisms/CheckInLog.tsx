import Badge from 'components/atoms/Badge'
import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import useOptions from 'hooks/useOptions'
import { formatDateTime, formatHours } from 'lib/dates'
import { roundLabel } from 'lib/rounds'
import { CheckIn } from 'types/worklog'

import styles from 'styles/Table.module.css'

type Props = {
  checkIns: CheckIn[]
  editingId: string | null
  emptyText: string
  onEdit: (checkIn: CheckIn) => void
  onDelete: (checkIn: CheckIn) => void
}

// The user's check-ins for the week, newest first.
export default function CheckInLog(props: Props) {
  const { checkIns, editingId, emptyText, onEdit, onDelete } = props
  const { phaseLabel, statusLabel } = useOptions()

  if (checkIns.length === 0) {
    return <div className={`${styles.wrap} ${styles.empty}`}>{emptyText}</div>
  }

  return (
    <div className={styles.wrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>When</th>
            <th>Ticket</th>
            <th>Title</th>
            <th>Task</th>
            <th>Status</th>
            <th className={styles.num}>Hours</th>
            <th>Note</th>
            <th aria-label="Actions" />
          </tr>
        </thead>
        <tbody>
          {checkIns.map((c) => (
            <tr key={c.id} className={c.id === editingId ? styles.highlight : undefined}>
              <td className={styles.nowrap}>
                {formatDateTime(c.workedAt)}
                {c.source === 'import' && <div className={styles.muted}>imported</div>}
              </td>
              <td className={styles.key}>{c.ticketKey}</td>
              <td>{c.title}</td>
              <td className={styles.task}>
                {phaseLabel(c.phase)}
                {c.round > 1 && <div className={styles.muted}>{roundLabel(c.round)}</div>}
              </td>
              <td>
                <Badge value={c.status}>{statusLabel(c.status)}</Badge>
              </td>
              <td className={styles.num}>{formatHours(c.hours)}</td>
              <td className={styles.muted}>{c.note}</td>
              <td>
                <div className={styles.actions}>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onEdit(c)}
                    aria-label={`Edit ${c.ticketKey} check-in`}
                    title="Edit"
                  >
                    <Icon name="edit" size={16} />
                  </Button>
                  <Button
                    variant="danger"
                    size="icon"
                    onClick={() => onDelete(c)}
                    aria-label={`Delete ${c.ticketKey} check-in`}
                    title="Delete"
                  >
                    <Icon name="trash" size={16} />
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
