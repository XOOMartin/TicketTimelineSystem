import { useState } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import Select from 'components/atoms/Select'
import PageLayout from 'components/layouts/PageLayout'
import Toast from 'components/molecules/Toast'
import ConfirmDialog from 'components/molecules/ConfirmDialog'
import WeekPicker from 'components/molecules/WeekPicker'
import SearchBox from 'components/molecules/SearchBox'
import CheckInDialog from 'components/organisms/CheckInDialog'
import CheckInLog from 'components/organisms/CheckInLog'
import useAuth from 'hooks/useAuth'
import useMyWeek from 'hooks/useMyWeek'
import useOptions from 'hooks/useOptions'
import { formatDateTime, formatHours } from 'lib/dates'
import { plural } from 'lib/text'
import { CheckIn } from 'types/worklog'

import styles from 'styles/Dashboard.module.css'

// The open dialog: a new check-in, one being edited, or none
type Editing = CheckIn | 'new' | null

// Create, read, update and delete the member's check-ins for a week.
export default function CheckIns() {
  const { user } = useAuth()
  const { options } = useOptions()
  const { week, setWeek, data, isLoading, error, create, update, remove, exportUrl, notice, clearNotice } = useMyWeek()
  const [editing, setEditing] = useState<Editing>(null)
  const [deleting, setDeleting] = useState<CheckIn | null>(null)
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState('')

  const checkIns = data?.checkIns ?? []
  const q = query.trim().toLowerCase()
  const shown = checkIns.filter(
    (c) =>
      (status === '' || c.status === status) &&
      (q === '' || [c.ticketKey, c.title, c.note].some((s) => s.toLowerCase().includes(q))),
  )
  const shownHours = shown.reduce((sum, c) => sum + c.hours, 0)
  const isFiltered = q !== '' || status !== ''

  return (
    <PageLayout
      title="My check-ins"
      heading="My check-ins"
      subheading="Log, edit and remove the work you did this week"
      actions={
        <>
          <LinkButton href={exportUrl} download>
            <Icon name="download" size={16} />
            Export .xlsx
          </LinkButton>
          <Button variant="primary" onClick={() => setEditing('new')}>
            <Icon name="plus" size={16} />
            New check-in
          </Button>
        </>
      }
    >
      {user && (
        <>
          {error && !editing && !deleting && <Message variant="error">{error}</Message>}

          <div className={styles.toolbar}>
            <WeekPicker week={week} onChange={setWeek} />
            <SearchBox
              className={styles.search}
              placeholder="Search ticket, title or note"
              value={query}
              onChange={setQuery}
              label="Search check-ins"
            />
            <Select
              className={styles.filter}
              aria-label="Filter by status"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              options={[{ value: '', label: 'All statuses' }, ...(options?.statuses ?? [])]}
            />
            <span className={styles.count}>
              {plural(shown.length, 'check-in')} · {formatHours(shownHours)} h
            </span>
          </div>

          {isLoading && !data ? (
            <Message>Loading…</Message>
          ) : (
            <CheckInLog
              checkIns={shown}
              editingId={editing && editing !== 'new' ? editing.id : null}
              emptyText={isFiltered ? 'No check-ins match your search.' : 'No check-ins this week yet.'}
              onEdit={setEditing}
              onDelete={setDeleting}
            />
          )}

          {editing && (
            <CheckInDialog
              // key resets the form when switching between check-ins
              key={editing === 'new' ? 'new' : editing.id}
              checkIn={editing === 'new' ? null : editing}
              defaultProject={user.team}
              knownTickets={data?.tickets ?? []}
              onLog={create}
              onUpdate={editing === 'new' ? undefined : (input) => update(editing.id, input)}
              onClose={() => setEditing(null)}
            />
          )}

          {deleting && (
            <ConfirmDialog
              title="Delete check-in?"
              confirmLabel="Delete"
              onConfirm={() => remove(deleting.id)}
              onClose={() => setDeleting(null)}
            >
              {error && <Message variant="error">{error}</Message>}
              The <strong>{deleting.ticketKey}</strong> check-in from{' '}
              <strong>{formatDateTime(deleting.workedAt)}</strong> ({formatHours(deleting.hours)} h) will be removed.
              This can’t be undone.
            </ConfirmDialog>
          )}
        </>
      )}
      <Toast message={notice} onDone={clearNotice} />
    </PageLayout>
  )
}
