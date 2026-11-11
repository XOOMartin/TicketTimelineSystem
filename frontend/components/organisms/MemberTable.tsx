import { ChangeEvent } from 'react'

import Badge from 'components/atoms/Badge'
import Icon from 'components/atoms/Icon'
import Select from 'components/atoms/Select'
import PersonCell from 'components/molecules/PersonCell'
import useAuth from 'hooks/useAuth'
import useOptions from 'hooks/useOptions'
import { ROLE_OPTIONS } from 'lib/accounts'
import { formatDateTime, formatDay, formatHours } from 'lib/dates'
import { MemberWeek } from 'types/worklog'

import styles from 'styles/Table.module.css'

const SUBMISSION_LABELS = {
  app: 'Logged in app',
  imported: 'File imported',
  missing: 'Missing',
}

type Props = {
  members: MemberWeek[]
  onAssign: (id: string, team: string, role: string) => void
  onChooseFile: (member: MemberWeek, file: File) => void
}

export default function MemberTable(props: Props) {
  const { members, onAssign, onChooseFile } = props
  const { user } = useAuth()
  const { options } = useOptions()

  if (members.length === 0) {
    return <div className={`${styles.wrap} ${styles.empty}`}>Nobody has signed up yet.</div>
  }

  const handleFile = (m: MemberWeek) => (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    // Clear it so choosing the same file again still fires onChange
    e.target.value = ''
    if (file) onChooseFile(m, file)
  }

  return (
    <div className={styles.wrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th>Member</th>
            <th>Team</th>
            <th>Role</th>
            <th className={styles.num}>Check-ins</th>
            <th className={styles.num} title="Hours logged / expected so far">
              Hours
            </th>
            <th title="Working days without check-ins">Missed days</th>
            <th>This week</th>
            <th aria-label="Import spreadsheet" />
          </tr>
        </thead>
        <tbody>
          {members.map((m) => (
            <tr key={m.user.id}>
              <td>
                <PersonCell
                  name={m.user.name}
                  seed={m.user.username}
                  detail={`@${m.user.username}`}
                  href={`/admin/members/${encodeURIComponent(m.user.id)}`}
                  hrefTitle="Open dashboard"
                />
              </td>
              <td>
                <Select
                  className={styles.select}
                  aria-label={`${m.user.name}'s team`}
                  value={m.user.team}
                  options={options?.projects ?? [m.user.team]}
                  onChange={(e) => onAssign(m.user.id, e.target.value, m.user.role)}
                />
              </td>
              <td>
                <Select
                  className={styles.select}
                  aria-label={`${m.user.name}'s role`}
                  value={m.user.role}
                  options={ROLE_OPTIONS}
                  // Admins can't demote themselves and lock everyone out
                  disabled={m.user.id === user?.id}
                  onChange={(e) => onAssign(m.user.id, m.user.team, e.target.value)}
                />
              </td>
              <td className={styles.num}>{m.activity.checkIns}</td>
              <td className={styles.num}>
                {formatHours(m.activity.hours)} / {formatHours(m.activity.expectedHours)}
              </td>
              <td className={styles.muted}>{m.activity.missedDays.map(formatDay).join(', ') || '—'}</td>
              <td>
                <Badge value={m.submission}>{SUBMISSION_LABELS[m.submission]}</Badge>
                {m.lastImport && (
                  <div className={styles.muted} title={formatDateTime(m.lastImport.importedAt)}>
                    {m.lastImport.filename}
                  </div>
                )}
              </td>
              <td>
                <label className={styles.upload}>
                  <Icon name="upload" size={14} />
                  Import .xlsx
                  <input
                    type="file"
                    accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                    aria-label={`Import a spreadsheet for ${m.user.name}`}
                    onChange={handleFile(m)}
                  />
                </label>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
