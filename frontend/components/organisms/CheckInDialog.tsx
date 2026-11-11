import Modal from 'components/molecules/Modal'
import CheckInForm from 'components/organisms/CheckInForm'
import { CheckIn, CheckInInput, TicketRow, TicketWorkInput } from 'types/worklog'

type Props = {
  // When set, edits this check-in; otherwise logs a new one
  checkIn: CheckIn | null
  // When set (and no checkIn), corrects this timeline row's totals
  ticketRow?: TicketRow
  // For a new entry; defaults to "Log a check-in"
  title?: string
  defaultProject: string
  knownTickets: TicketRow[]
  onLog: (input: TicketWorkInput) => Promise<void>
  // Only needed when editing a check-in or correcting a row
  onUpdate?: (input: CheckInInput) => Promise<void>
  onCorrect?: (input: TicketWorkInput) => Promise<void>
  onClose: () => void
}

export default function CheckInDialog(props: Props) {
  const { checkIn, ticketRow, title, defaultProject, knownTickets, onLog, onUpdate, onCorrect, onClose } = props

  let heading = title ?? 'Log a check-in'
  if (checkIn) heading = `Edit check-in · ${checkIn.ticketKey}`
  else if (ticketRow) heading = `Edit ${ticketRow.ticketKey}`

  // Closes only once the save worked, so a failed save keeps the input
  const thenClose =
    <T,>(save?: (input: T) => Promise<void>) =>
    async (input: T) => {
      await save?.(input)
      onClose()
    }

  return (
    // Logging or correcting puts the phases beside the details, so it gets
    // the room for two columns; editing one check-in is a short form
    <Modal title={heading} size={checkIn ? 'normal' : 'large'} onClose={onClose}>
      <CheckInForm
        checkIn={checkIn ?? undefined}
        ticketRow={ticketRow}
        defaultProject={defaultProject}
        knownTickets={knownTickets}
        onLog={thenClose(onLog)}
        onUpdate={thenClose(onUpdate)}
        onCorrect={thenClose(onCorrect)}
        onCancel={onClose}
      />
    </Modal>
  )
}
