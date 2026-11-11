import { useState } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import ConfirmDialog from 'components/molecules/ConfirmDialog'
import StatTile from 'components/molecules/StatTile'
import Toast from 'components/molecules/Toast'
import SectionHeader from 'components/molecules/SectionHeader'
import DeleteUserPermanentlyDialog from 'components/organisms/DeleteUserPermanentlyDialog'
import UserFormDialog from 'components/organisms/UserFormDialog'
import UserTable from 'components/organisms/UserTable'
import useAuth from 'hooks/useAuth'
import useUsers from 'hooks/useUsers'
import { AdminUser } from 'types/worklog'

import dashboard from 'styles/Dashboard.module.css'
import styles from 'styles/UsersPage.module.css'

// The admin's user management: add, edit and delete accounts. Deleting is
// soft (the account can't log in but can be restored); a deleted account
// can then be deleted permanently, with all its data.
export default function Users() {
  const { user: me } = useAuth()
  const { data, isLoading, error, notice, clearNotice, create, update, remove, restore, removePermanently } = useUsers()
  const [isAdding, setIsAdding] = useState(false)
  const [editing, setEditing] = useState<AdminUser | null>(null)
  const [deleting, setDeleting] = useState<AdminUser | null>(null)
  const [purging, setPurging] = useState<AdminUser | null>(null)
  const active = data?.active ?? []
  const deleted = data?.deleted ?? []
  const count = (pred: (u: AdminUser) => boolean) => String(active.filter(pred).length)

  return (
    <PageLayout
      title="Users"
      heading="Users"
      subheading="Add, edit and delete accounts. A deleted account can be restored, or deleted permanently."
      requireAdmin
      actions={
        <Button variant="primary" onClick={() => setIsAdding(true)}>
          <Icon name="plus" size={16} />
          Add user
        </Button>
      }
    >
      {error && <Message variant="error">{error}</Message>}

      {isLoading && !data ? (
        <Message>Loading…</Message>
      ) : (
        <>
          <div className={dashboard.stats}>
            <StatTile icon="users" color="var(--blue)" label="Accounts" value={String(active.length)} />
            <StatTile icon="account" color="var(--accent)" label="Admins" value={count((u) => u.role === 'admin')} />
            <StatTile icon="ticket" color="var(--text)" label="BlackBox" value={count((u) => u.team === 'BlackBox')} />
            <StatTile icon="ticket" color="var(--teal)" label="SPinno" value={count((u) => u.team === 'SPinno')} />
            <StatTile icon="trash" color="var(--pink)" label="Deleted" value={String(deleted.length)} />
          </div>

          <SectionHeader title="Accounts" count={active.length} countTone="accent" />
          <UserTable kind="active" users={active} currentUserId={me?.id} onEdit={setEditing} onDelete={setDeleting} />

          <SectionHeader title="Deleted accounts" count={deleted.length} countTone="red" />
          <p className={styles.hint}>
            They can&apos;t log in and are left out of member lists, reports and exports. Their data is kept until
            they&apos;re deleted permanently.
          </p>
          <UserTable
            kind="deleted"
            users={deleted}
            onRestore={(u) => restore(u).catch(() => {})}
            onDeletePermanently={setPurging}
          />
        </>
      )}

      {isAdding && <UserFormDialog onSave={create} onClose={() => setIsAdding(false)} />}
      {editing && (
        <UserFormDialog
          user={editing}
          isSelf={editing.id === me?.id}
          onSave={(input) => update(editing, input)}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting && (
        <ConfirmDialog
          title={`Delete ${deleting.name}?`}
          confirmLabel="Delete"
          onConfirm={() => remove(deleting)}
          onClose={() => setDeleting(null)}
        >
          <p>
            <strong>{deleting.username}</strong> will be logged out and won&apos;t be able to log in. They&apos;re left
            out of member lists, reports and exports.
          </p>
          <p>Their data is kept: you can restore the account from Deleted accounts, or delete it permanently there.</p>
        </ConfirmDialog>
      )}
      {purging && (
        <DeleteUserPermanentlyDialog
          user={purging}
          onConfirm={() => removePermanently(purging)}
          onClose={() => setPurging(null)}
        />
      )}

      <Toast message={notice} onDone={clearNotice} />
    </PageLayout>
  )
}
