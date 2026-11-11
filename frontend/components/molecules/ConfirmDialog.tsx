import { ReactNode } from 'react'

import Button from 'components/atoms/Button'
import FormActions from 'components/molecules/FormActions'
import Modal from 'components/molecules/Modal'
import useAsyncAction from 'hooks/useAsyncAction'

import styles from 'styles/Card.module.css'

type Props = {
  title: string
  confirmLabel: string
  busyLabel?: string // while confirming; "Deleting…" by default
  cancelLabel?: string
  // Red for destroying something, the main colour for anything else
  tone?: 'danger' | 'primary'
  // False keeps the confirm button off, e.g. until a name is typed to confirm
  canConfirm?: boolean
  onConfirm: () => Promise<void>
  onClose: () => void
  children: ReactNode
}

// Asks before a destructive action, and stays open until it finishes.
export default function ConfirmDialog(props: Props) {
  const { title, confirmLabel, busyLabel = 'Deleting…', cancelLabel = 'Cancel', tone = 'danger' } = props
  const { canConfirm = true, onConfirm, onClose, children } = props
  // The page shows the error; the dialog stays open to try again
  const { run, isBusy } = useAsyncAction()
  const handleConfirm = async () => {
    if (await run(onConfirm, { stayBusy: true })) onClose()
  }

  return (
    <Modal title={title} onClose={onClose}>
      <div className={styles.text}>{children}</div>
      <FormActions onCancel={onClose} cancelLabel={cancelLabel} isBusy={isBusy}>
        <Button
          variant={tone === 'danger' ? 'destructive' : 'primary'}
          onClick={handleConfirm}
          disabled={isBusy || !canConfirm}
          autoFocus
        >
          {isBusy ? busyLabel : confirmLabel}
        </Button>
      </FormActions>
    </Modal>
  )
}
