import Badge, { BadgeTone } from 'components/atoms/Badge'
import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import PersonCell from 'components/molecules/PersonCell'
import { roleLabel } from 'lib/accounts'
import { formatDateTime } from 'lib/dates'
import { AdminUser } from 'types/worklog'

import tableStyles from 'styles/Table.module.css'
import styles from 'styles/UsersPage.module.css'

type Props = {
  users: AdminUser[]
  // The active accounts, or the deleted ones (which can be restored or
  // deleted permanently)
  kind: 'active' | 'deleted'
  currentUserId?: string
  onEdit?: (user: AdminUser) => void
  onDelete?: (user: AdminUser) => void
  onRestore?: (user: AdminUser) => void
  onDeletePermanently?: (user: AdminUser) => void
}

// Each team's tag colour, as in the team sheet: BlackBox dark, SPinno teal
const TEAM_TONES: Record<string, BadgeTone> = { BlackBox: 'ink', SPinno: 'teal' }

// The accounts: who (name, username), team, role, when, then what can be
// done. Every row has the same two buttons, so they line up.
export default function UserTable(props: Props) {
  const { users, kind, currentUserId, onEdit, onDelete, onRestore, onDeletePermanently } = props
  const isDeleted = kind === 'deleted'

  if (users.length === 0) {
    return (
      <div className={styles.empty}>
        <Icon name={isDeleted ? 'check' : 'account'} size={16} />
        {isDeleted ? 'No deleted accounts.' : 'No accounts yet.'}
      </div>
    )
  }

  return (
    <div className={tableStyles.wrap}>
      <table className={`${tableStyles.table} ${styles.table}`}>
        <colgroup>
          <col className={styles.colUser} />
          <col className={styles.colTeam} />
          <col className={styles.colRole} />
          <col className={styles.colDate} />
          <col className={styles.colActions} />
        </colgroup>
        <thead>
          <tr>
            <th>User</th>
            <th>Team</th>
            <th>Role</th>
            <th>{isDeleted ? 'Deleted' : 'Created'}</th>
            <th className={styles.actionsCell} aria-label="Actions" />
          </tr>
        </thead>
        <tbody>
          {users.map((u) => {
            const isSelf = u.id === currentUserId
            return (
              <tr key={u.id} className={isDeleted ? styles.deletedRow : undefined}>
                <td>
                  <PersonCell
                    name={u.name}
                    seed={u.username}
                    detail={`@${u.username}`}
                    tag={isSelf && <span className={styles.you}>You</span>}
                    avatarSize={36}
                    muted={isDeleted}
                  />
                </td>
                <td>
                  <Badge tone={TEAM_TONES[u.team] ?? 'gray'}>{u.team}</Badge>
                </td>
                <td>
                  <Badge tone={u.role === 'admin' ? 'accent' : 'gray'}>{roleLabel(u.role)}</Badge>
                </td>
                <td className={styles.date}>
                  {isDeleted ? (
                    <Badge tone="red">{formatDateTime(u.deletedAt ?? '')}</Badge>
                  ) : (
                    formatDateTime(u.createdAt)
                  )}
                </td>
                <td className={styles.actionsCell}>
                  <div className={styles.actions}>
                    {isDeleted ? (
                      <>
                        <Button size="sm" variant="outline" onClick={() => onRestore?.(u)}>
                          <Icon name="restore" size={14} />
                          Restore
                        </Button>
                        <Button size="sm" variant="danger" onClick={() => onDeletePermanently?.(u)}>
                          <Icon name="trash" size={14} />
                          Delete permanently
                        </Button>
                      </>
                    ) : (
                      <>
                        <Button size="sm" variant="outline" onClick={() => onEdit?.(u)}>
                          <Icon name="edit" size={14} />
                          Edit
                        </Button>
                        {/* Shown but off for the admin's own account, so the
                            buttons line up on every row */}
                        <Button
                          size="sm"
                          variant="danger"
                          onClick={() => onDelete?.(u)}
                          disabled={isSelf}
                          title={isSelf ? "You can't delete your own account" : undefined}
                        >
                          <Icon name="trash" size={14} />
                          Delete
                        </Button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
