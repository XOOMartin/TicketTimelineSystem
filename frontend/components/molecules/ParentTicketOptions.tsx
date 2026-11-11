import { TicketRow } from 'types/worklog'

type Props = {
  // What a parent box's list attribute points at
  id: string
  tickets: TicketRow[]
}

// The parent tickets suggested in a parent box: each one's number, with its
// title beside it.
export default function ParentTicketOptions(props: Props) {
  const { id, tickets } = props

  return (
    <datalist id={id}>
      {tickets.map((t) => (
        <option key={t.ticketKey} value={t.ticket}>
          {t.title}
        </option>
      ))}
    </datalist>
  )
}
