import Badge from 'components/atoms/Badge'
import Button from 'components/atoms/Button'
import FormActions from 'components/molecules/FormActions'
import useOptions from 'hooks/useOptions'
import { formatDateTime, formatDay, formatHours } from 'lib/dates'
import { plural } from 'lib/text'
import { SheetImport } from 'types/worklog'

import cardStyles from 'styles/Card.module.css'
import styles from 'styles/Table.module.css'

// Shown in a Modal titled with the member and file
type Props = {
  preview: SheetImport
  isBusy: boolean
  onConfirm: () => void
  onCancel: () => void
}

export default function ImportPreview(props: Props) {
  const { preview, isBusy, onConfirm, onCancel } = props
  const { phaseLabel, statusLabel } = useOptions()

  return (
    <>
      <p className={cardStyles.text}>
        Week of {formatDay(preview.period.first)} – {formatDay(preview.period.last)}: <strong>{preview.valid}</strong>{' '}
        new {preview.valid === 1 ? 'row' : 'rows'} will be added
        {preview.skipped > 0 && `, ${preview.skipped} skipped`}.
      </p>
      <div className={styles.wrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th className={styles.num}>Row</th>
              <th>When</th>
              <th>Ticket</th>
              <th>Title</th>
              <th>Task</th>
              <th>Status</th>
              <th className={styles.num}>Hours</th>
              <th>Result</th>
            </tr>
          </thead>
          <tbody>
            {preview.rows.map((row) => {
              const c = row.checkIn
              return (
                <tr key={row.line}>
                  <td className={styles.num}>{row.line}</td>
                  <td className={styles.nowrap}>{c ? formatDateTime(c.workedAt) : '—'}</td>
                  <td className={styles.key}>{c?.ticketKey ?? '—'}</td>
                  <td>{c?.title}</td>
                  <td>{c && phaseLabel(c.phase)}</td>
                  <td>{c && <Badge value={c.status}>{statusLabel(c.status)}</Badge>}</td>
                  <td className={styles.num}>{c && formatHours(c.hours)}</td>
                  <td>
                    {row.error ? (
                      <Badge value="blocked">{row.error}</Badge>
                    ) : row.duplicate ? (
                      <Badge value="not_started">Already imported</Badge>
                    ) : (
                      <Badge value="done">New</Badge>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <FormActions onCancel={onCancel} isBusy={isBusy}>
        <Button variant="primary" onClick={onConfirm} disabled={isBusy || preview.valid === 0}>
          {isBusy ? 'Importing…' : `Import ${plural(preview.valid, 'row')}`}
        </Button>
      </FormActions>
    </>
  )
}
