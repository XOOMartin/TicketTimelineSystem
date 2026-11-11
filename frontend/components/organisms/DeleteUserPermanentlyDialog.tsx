import { useState } from 'react'

import ConfirmDialog from 'components/molecules/ConfirmDialog'
import FormField from 'components/molecules/FormField'
import { AdminUser } from 'types/worklog'

type Props = {
  user: AdminUser
  onConfirm: () => Promise<void>
  onClose: () => void
}

// Asks before deleting an account for good, which can't be undone: the
// usual confirm, which only works once the account's username is typed.
export default function DeleteUserPermanentlyDialog(props: Props) {
  const { user, onConfirm, onClose } = props
  const [typed, setTyped] = useState('')

  return (
    <ConfirmDialog
      title={`Delete ${user.name} permanently?`}
      confirmLabel="Delete permanently"
      canConfirm={typed.trim().toLowerCase() === user.username}
      onConfirm={onConfirm}
      onClose={onClose}
    >
      <p>
        This removes the account <strong>{user.username}</strong> and everything that is theirs: check-ins, tickets,
        progress, timers and imports. It can&apos;t be undone.
      </p>
      <p>Saved reports keep their numbers, with the name shown as “(deleted user)”.</p>
      <FormField
        label={`Type ${user.username} to confirm`}
        value={typed}
        onChange={(e) => setTyped(e.target.value)}
        autoComplete="off"
        autoCapitalize="none"
        spellCheck={false}
        data-autofocus
      />
    </ConfirmDialog>
  )
}
