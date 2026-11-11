import Message from 'components/atoms/Message'
import ConfirmDialog from 'components/molecules/ConfirmDialog'
import { formatShortHours } from 'lib/dates'
import { roundLabel } from 'lib/rounds'
import { TicketRow } from 'types/worklog'

type Props = {
  row: TicketRow
  round: number
  error: string | null
  onDeleteRound: (row: TicketRow, round: number) => Promise<void>
  onDeleteTicket: (row: TicketRow) => Promise<void>
  onClose: () => void
}

// Asks before deleting one round of a ticket (the original work or a
// revision) from its line in the breakdown. A ticket must keep some time,
// so when this is the only round with any, the whole ticket goes instead,
// and the dialog says so.
export default function DeleteRoundDialog(props: Props) {
  const { row, round, error, onDeleteRound, onDeleteTicket, onClose } = props
  const label = roundLabel(round)
  const hours = row.rounds?.find((r) => r.round === round)?.hours ?? 0
  const otherHours = row.totalHours - hours
  // In whole minutes, so float leftovers don't count as time
  const isLast = Math.round(otherHours * 60) <= 0

  return (
    <ConfirmDialog
      title={`Delete ${label} of ${row.ticketKey}?`}
      confirmLabel={isLast ? 'Delete ticket' : `Delete ${label}`}
      onConfirm={() => (isLast ? onDeleteTicket(row) : onDeleteRound(row, round))}
      onClose={onClose}
    >
      {error && <Message variant="error">{error}</Message>}
      {isLast ? (
        <>
          <strong>{label}</strong> is the only round of <strong>{row.title}</strong> with time logged, so deleting it
          removes the whole ticket: all <strong>{formatShortHours(row.totalHours)}</strong>, along with its start and
          end dates. This can’t be undone.
        </>
      ) : (
        <>
          {hours > 0 ? (
            <>
              All <strong>{formatShortHours(hours)}</strong> you logged in <strong>{label}</strong>
            </>
          ) : (
            <>
              <strong>{label}</strong>, which has no time yet,
            </>
          )}{' '}
          will be removed from <strong>{row.title}</strong>. Its other rounds ({formatShortHours(otherHours)}) stay as
          they are, and its progress steps are worked out again from the time left. This can’t be undone.
        </>
      )}
    </ConfirmDialog>
  )
}
