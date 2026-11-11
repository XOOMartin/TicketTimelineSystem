import { FormEvent } from 'react'

import Button from 'components/atoms/Button'
import Message from 'components/atoms/Message'
import Modal from 'components/molecules/Modal'
import FormActions from 'components/molecules/FormActions'
import AccountFields from 'components/organisms/AccountFields'
import useAccountForm from 'hooks/useAccountForm'
import useAsyncAction from 'hooks/useAsyncAction'
import { AdminUser, UserInput } from 'types/worklog'

import styles from 'styles/Card.module.css'

type Props = {
  // The account to edit; none to add one
  user?: AdminUser
  // Whether it's the admin's own account: its role can't be changed here
  isSelf?: boolean
  onSave: (input: UserInput) => Promise<void>
  onClose: () => void
}

// An admin adding an account, or editing one (a new password left empty
// keeps the current one), with the account fields every account form shares.
export default function UserFormDialog(props: Props) {
  const { user, isSelf = false, onSave, onClose } = props
  const form = useAccountForm(user ? 'edit' : 'add', user)
  const { run, isBusy: isSaving, error } = useAsyncAction()

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (form.hasErrors) {
      form.showAll()
      return
    }
    if (await run(() => onSave(form.trimmed()), { stayBusy: true })) onClose()
  }

  return (
    <Modal title={user ? `Edit ${user.name}` : 'Add user'} onClose={onClose}>
      <form onSubmit={handleSubmit} noValidate>
        {error && <Message variant="error">{error}</Message>}
        {/* Two by two: first and last name, username and password, team and role */}
        <AccountFields form={form} className={styles.pairs} isSelf={isSelf} />
        <FormActions onCancel={onClose} isBusy={isSaving}>
          <Button type="submit" variant="primary" disabled={isSaving}>
            {isSaving ? 'Saving…' : user ? 'Save changes' : 'Add user'}
          </Button>
        </FormActions>
      </form>
    </Modal>
  )
}
