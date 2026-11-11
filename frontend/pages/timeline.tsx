import { useEffect, useState } from 'react'
import { useRouter } from 'next/router'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import ConfirmDialog from 'components/molecules/ConfirmDialog'
import AddRevisionDialog from 'components/organisms/AddRevisionDialog'
import Toast from 'components/molecules/Toast'
import WeekPicker from 'components/molecules/WeekPicker'
import DeleteRoundDialog from 'components/organisms/DeleteRoundDialog'
import TicketDetailsDialog, { TicketDetails } from 'components/organisms/TicketDetailsDialog'
import TicketEditDialog from 'components/organisms/TicketEditDialog'
import TicketFlowDialog from 'components/organisms/TicketFlowDialog'
import TicketTimelineSheet from 'components/organisms/TicketTimelineSheet'
import useAuth from 'hooks/useAuth'
import useMyWeek from 'hooks/useMyWeek'
import useTimer from 'hooks/useTimer'
import * as api from 'lib/api'
import { formatShortHours, mondayOf, today } from 'lib/dates'
import { parentCandidates, ticketKeyOf } from 'lib/tickets'
import { TicketRow } from 'types/worklog'

// A ticket just entered in Input timeline, shown in its progress steps until
// its first time is recorded and it comes back from the server
function draftRow({ project, repo, ticket, title, parent }: TicketDetails): TicketRow {
  return {
    project,
    repo,
    ticket,
    ticketKey: ticketKeyOf({ project, repo, ticket }),
    parent,
    title,
    assignees: [],
    phase: '',
    status: 'not_started',
    periodHours: 0,
    totalHours: 0,
    phaseHours: {},
    reviewBacks: 0,
    rounds: [],
    step: 'investigation',
    stepRound: 1,
    lastUpdate: '',
    note: '',
    startDate: '',
    endDate: '',
  }
}

// The member's tickets laid out like their tab in the progress sheet. Input
// timeline asks which ticket, then opens its progress steps; Edit corrects
// the times in a form.
export default function Timeline() {
  const { user } = useAuth()
  const myWeek = useMyWeek()
  const { timer } = useTimer()
  const { week, setWeek, data, isLoading, error, notice, clearNotice } = myWeek
  const {
    addTicket,
    setParent,
    makeParent,
    makeChild,
    makeParentOf,
    setType,
    correct,
    saveCell,
    removeTicket,
    removeRound,
    setStep,
    recordOutcome,
  } = myWeek
  // The row whose times are being corrected, and the round whose tab opens
  // first when its own Edit in the breakdown was used
  const [editing, setEditing] = useState<TicketRow | null>(null)
  const [editRound, setEditRound] = useState<number | null>(null)
  // The field to start in, when a cell was double-clicked
  const [editField, setEditField] = useState<string | undefined>(undefined)
  const edit = (row: TicketRow, round: number | null = null, field?: string) => {
    setEditing(row)
    setEditRound(round)
    setEditField(field)
  }
  // The round waiting for the user to confirm deleting it
  const [deletingRound, setDeletingRound] = useState<{ row: TicketRow; round: number } | null>(null)
  // Input timeline's first part, the ticket details; from a spec ticket's +,
  // for a ticket under that spec
  const [isEntering, setIsEntering] = useState(false)
  const [underSpec, setUnderSpec] = useState<TicketRow | null>(null)
  const enterUnder = (spec: TicketRow) => {
    setUnderSpec(spec)
    setIsEntering(true)
  }
  const stopEntering = () => {
    setIsEntering(false)
    setUnderSpec(null)
    setInitialDetails(undefined)
  }
  // What was entered before Continue, so the steps' Previous can go back to
  // the form with it filled in
  const [initialDetails, setInitialDetails] = useState<TicketDetails | undefined>(undefined)
  const [entered, setEntered] = useState<{ details: TicketDetails; underSpec: TicketRow | null } | null>(null)
  const backToDetails = () => {
    if (!entered) return
    setProgressKey(null)
    setUnderSpec(entered.underSpec)
    setInitialDetails(entered.details)
    setIsEntering(true)
  }
  const [draft, setDraft] = useState<TicketRow | null>(null)
  // The row waiting for the user to confirm deleting it
  const [deleting, setDeleting] = useState<TicketRow | null>(null)
  // The ticket whose + asks which review requested changes, to add a revision
  const [revisionFor, setRevisionFor] = useState<TicketRow | null>(null)
  // The ticket whose progress steps are open; its row follows the data, so
  // the steps move on as time is recorded
  const [progressKey, setProgressKey] = useState<string | null>(null)
  const progressRow =
    data?.tickets.find((t) => t.ticketKey === progressKey) ?? (draft?.ticketKey === progressKey ? draft : null)

  // Once the new ticket's first time is recorded, the real row takes over
  useEffect(() => {
    if (draft && data?.tickets.some((t) => t.ticketKey === draft.ticketKey)) setDraft(null)
  }, [draft, data])

  const startProgress = async (details: TicketDetails) => {
    const key = ticketKeyOf(details)
    const isKnown = data?.tickets.some((t) => t.ticketKey === key)
    // A SPinno ticket's type decides its steps, so a new one goes in the table
    // straight away (with its type and parent), and its steps are saved as
    // it goes; if that fails (e.g. its spec isn't there), the form says why
    if (details.type && !isKnown) {
      await addTicket(details, details.title, details.parent, details.type)
      setEntered({ details, underSpec })
      stopEntering()
      setWeek(mondayOf(today()))
      setProgressKey(key)
      return
    }
    // A backend or web ticket's spec goes on straight away, before any time;
    // if that spec isn't in the timeline, the form stays open and says so
    if (details.parent) await api.setParent(details, details.parent)
    setEntered({ details, underSpec })
    stopEntering()
    if (!data?.tickets.some((t) => t.ticketKey === key)) {
      setDraft(draftRow(details))
      // Its time is recorded today, so show this week to see it arrive
      setWeek(mondayOf(today()))
    }
    setProgressKey(key)
  }

  // Opened from a ticket on the dashboard, as
  // /timeline?week=2026-11-02&edit=BlackBox%233287: show that week, then open
  // the ticket in the form
  const { query, isReady, replace } = useRouter()
  const editKey = typeof query.edit === 'string' ? query.edit : null
  const openKey = typeof query.progress === 'string' ? query.progress : null
  const queryWeek = typeof query.week === 'string' ? query.week : null
  const queryRound = typeof query.round === 'string' ? Number(query.round) || null : null
  const queryField = typeof query.field === 'string' ? query.field : undefined

  useEffect(() => {
    if (isReady && queryWeek) setWeek(queryWeek)
  }, [isReady, queryWeek, setWeek])

  // Opened from the dashboard's Input timeline, as
  // /timeline?week=2026-11-02&input=1: open Input timeline here
  const wantsInput = query.input === '1'
  useEffect(() => {
    if (!isReady || !wantsInput) return
    setIsEntering(true)
    // Drop the query so a reload doesn't reopen the form
    replace('/timeline', undefined, { shallow: true })
  }, [isReady, wantsInput, replace])

  useEffect(() => {
    // Wait for the requested week's data, not the default week's
    if (!(editKey || openKey) || !data || (queryWeek && data.period.first !== queryWeek)) return
    const row = data.tickets.find((t) => t.ticketKey === (editKey ?? openKey))
    if (row && editKey) edit(row, queryRound, queryField)
    if (row && openKey) setProgressKey(row.ticketKey)
    // A new ticket with its first timer running has no row yet: rebuild it
    // from the timer, on the step being timed
    if (!row && openKey && timer?.ticketKey === openKey) {
      setDraft({
        ...draftRow({ ...timer, repo: timer.repo ?? '', parent: '', type: '' }),
        step: timer.phase,
        stepRound: timer.round,
      })
      setProgressKey(openKey)
    }
    // Drop the query so a reload doesn't reopen the ticket
    replace('/timeline', undefined, { shallow: true })
  }, [editKey, openKey, queryWeek, queryRound, queryField, data, timer, replace])

  return (
    <PageLayout
      title="My timeline"
      heading="My timeline"
      subheading="Your tickets and the time spent on each phase, like your tab in the progress sheet"
      actions={
        <>
          <WeekPicker week={week} onChange={setWeek} />
          <Button variant="primary" onClick={() => setIsEntering(true)}>
            <Icon name="plus" size={16} />
            Input timeline
          </Button>
        </>
      }
    >
      {user && (
        <>
          {/* Save errors show beside the form's button; this is for loading */}
          {error && !data && <Message variant="error">{error}</Message>}

          {isLoading && !data ? (
            <Message>Loading…</Message>
          ) : (
            <TicketTimelineSheet
              name={user.name}
              rows={data?.tickets ?? []}
              onEdit={(row, field) => edit(row, null, field)}
              onDelete={setDeleting}
              onEditRound={(row, round, field) => edit(row, round, field)}
              onSaveCell={saveCell}
              onAddRound={setRevisionFor}
              onAddChild={enterUnder}
              onMakeParent={makeParent}
              onMakeChild={makeChild}
              onMakeParentOf={makeParentOf}
              onDeleteRound={(row, round) => setDeletingRound({ row, round })}
              onOpen={(row) => {
                // Opened from the table: there's no form to go back to
                setEntered(null)
                setProgressKey(row.ticketKey)
              }}
              editingKey={editing?.ticketKey ?? deleting?.ticketKey ?? deletingRound?.row.ticketKey}
            />
          )}

          {editing && (
            <TicketEditDialog
              // key resets the form when switching rows or rounds
              key={`${editing.ticketKey}-${editRound ?? ''}-${editField ?? ''}`}
              row={editing}
              initialRound={editRound ?? undefined}
              initialField={editField}
              onSetParent={setParent}
              onSetType={setType}
              specs={parentCandidates(data?.tickets ?? [], editing.project, editing.ticketKey)}
              onCorrect={correct}
              onClose={() => setEditing(null)}
            />
          )}

          {isEntering && (
            <TicketDetailsDialog
              underSpec={underSpec ?? undefined}
              initial={initialDetails}
              defaultProject={user.team}
              knownTickets={data?.tickets ?? []}
              onContinue={startProgress}
              onAdd={async ({ project, repo, ticket, title, parent, type }) => {
                await addTicket({ project, repo, ticket }, title, parent, type)
                stopEntering()
                // It shows from this week on, so show this week to see it
                setWeek(mondayOf(today()))
              }}
              onClose={stopEntering}
            />
          )}

          {/* One progress form for every ticket: a SPinno ticket steps through
              its type's own steps, a BlackBox one through the phases */}
          {progressRow && (
            <TicketFlowDialog
              // A fresh dialog per ticket, so a draft's state doesn't carry over
              key={progressRow.ticketKey}
              row={progressRow}
              onSetStep={async (step, label) => {
                // A new ticket has nothing saved yet: just move its step
                if (progressRow === draft) setDraft(draft.type ? { ...draft, flowStep: step } : { ...draft, step })
                else await setStep(progressRow, step, label)
              }}
              onOutcome={(input, message) => recordOutcome(progressRow, input, message)}
              onBack={entered && ticketKeyOf(entered.details) === progressRow.ticketKey ? backToDetails : undefined}
              onClose={() => {
                setProgressKey(null)
                setEntered(null)
              }}
            />
          )}

          {revisionFor && (
            <AddRevisionDialog
              row={revisionFor}
              onRecord={(input, message) => recordOutcome(revisionFor, input, message)}
              onClose={() => setRevisionFor(null)}
            />
          )}

          {deleting && (
            <ConfirmDialog
              title={`Delete ${deleting.ticketKey}?`}
              confirmLabel="Delete ticket"
              onConfirm={() => removeTicket(deleting)}
              onClose={() => setDeleting(null)}
            >
              {error && <Message variant="error">{error}</Message>}
              {deleting.totalHours > 0 ? (
                <>
                  <strong>{deleting.title}</strong> and all <strong>{formatShortHours(deleting.totalHours)}</strong> you
                  logged on it
                  {deleting.rounds?.length > 1 ? `, in all ${deleting.rounds.length} rounds,` : ''} will be removed,
                  along with its start and end dates. This can’t be undone.
                </>
              ) : (
                <>
                  <strong>{deleting.title}</strong> has no time logged yet. It will be removed from your timeline. This
                  can’t be undone.
                </>
              )}
            </ConfirmDialog>
          )}

          {deletingRound && (
            <DeleteRoundDialog
              {...deletingRound}
              error={error}
              onDeleteRound={removeRound}
              onDeleteTicket={removeTicket}
              onClose={() => setDeletingRound(null)}
            />
          )}
        </>
      )}
      <Toast message={notice} onDone={clearNotice} />
    </PageLayout>
  )
}
