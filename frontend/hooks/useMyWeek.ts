import { useEffect } from 'react'

import useMutation from 'hooks/useMutation'
import useOptions from 'hooks/useOptions'
import useResource from 'hooks/useResource'
import useTimer from 'hooks/useTimer'
import useWeek from 'hooks/useWeek'
import * as api from 'lib/api'
import { formatShortHours, toDay } from 'lib/dates'
import { latestRound, roundLabel, roundPhaseHours } from 'lib/rounds'
import { ticketKeyOf } from 'lib/tickets'
import { CellChange, CheckInInput, OutcomeInput, TicketRow, TicketWorkInput } from 'types/worklog'

export default function useMyWeek() {
  const { typeLabel } = useOptions()
  const [week, setWeek] = useWeek()
  const { data, isLoading, error, setError, reload } = useResource(() => api.fetchMyWeek(week), [week])
  // Runs a change, confirms it (a notice for a toast), then reloads so the
  // ticket summary is recomputed. Re-throws so the form can keep its input.
  const { mutate, notice, clearNotice } = useMutation(reload, setError)

  // Time logged by ending a timer shows up straight away
  const { loggedCount } = useTimer()
  // (reload is stable, so this runs only when a timer logged time)
  useEffect(() => {
    if (loggedCount > 0) reload()
  }, [loggedCount, reload])

  const create = (input: TicketWorkInput) => {
    const hours = Object.values(input.phaseHours).reduce((sum, h) => sum + h, 0)
    return mutate(() => api.createCheckIns(input), `Logged ${formatShortHours(hours)} on ${ticketKeyOf(input)}`)
  }
  // A ticket put in the table with no time yet; a backend or web one can go
  // under a spec ticket (its number) straight away
  const addTicket = (id: { project: string; repo?: string; ticket: string }, title: string, parent = '', type = '') =>
    mutate(() => api.addTicket(id, title, parent, type), `Added ${ticketKeyOf(id)} to your timeline`)
  // Changes a SPinno ticket's type; its steps start over in the new type's
  const setType = (row: TicketRow, type: string) =>
    mutate(() => api.setType(row, type), `${row.ticketKey}: now ${typeLabel(row.repo, type)}`)
  const correct = (input: TicketWorkInput) =>
    mutate(() => api.setTicketTotals(input), `Saved changes to ${ticketKeyOf(input)}`)
  // Moves a child under a parent ticket, or none (a parent made by dragging
  // becomes a child again)
  const setParent = (row: TicketRow, parent: string) =>
    mutate(
      () => api.setParent(row, parent),
      parent ? `${row.ticketKey}: now under parent #${parent}` : `${row.ticketKey}: no parent`,
    )
  // Makes a child spec a parent ticket (dragged out of any parent)
  const makeParent = (row: TicketRow) => mutate(() => api.makeParent(row), `${row.ticketKey} is a parent ticket now`)
  // Makes a parent made by dragging a child again
  const makeChild = (row: TicketRow) => mutate(() => api.makeChild(row), `${row.ticketKey} is a child ticket again`)
  // Makes a child spec a parent by dropping another child on it: the spec
  // becomes a parent (made by dragging, so it can be a child again), then the
  // child goes under it. If the move fails, the spec is a child again.
  const makeParentOf = (parent: TicketRow, child: TicketRow) =>
    mutate(async () => {
      await api.makeParent(parent)
      try {
        await api.setParent(child, parent.ticket)
      } catch (e) {
        await api.makeChild(parent).catch(() => {})
        throw e
      }
    }, `${parent.ticketKey} is a parent ticket now, with ${child.ticketKey} under it`)
  // Saves one cell edited in the table, keeping everything else on the ticket
  // as it is: a phase's time in one round, or the title or status
  const saveCell = (row: TicketRow, change: CellChange) => {
    // Moving it under another spec (or none) touches nothing else
    if ('parent' in change) return setParent(row, change.parent)
    const round = 'round' in change ? change.round : latestRound(row)
    const phaseHours = { ...roundPhaseHours(row, round) }
    if ('phase' in change) {
      if (change.minutes > 0) phaseHours[change.phase] = change.minutes / 60
      else delete phaseHours[change.phase]
    }
    const status = 'status' in change ? change.status : row.status
    const what = 'phase' in change ? `${roundLabel(round)} time` : 'title' in change ? 'title' : 'status'
    return mutate(
      () =>
        api.setTicketTotals({
          project: row.project,
          repo: row.repo,
          ticket: row.ticket,
          title: 'title' in change ? change.title : row.title,
          round,
          status,
          note: row.note,
          workedAt: new Date().toISOString(),
          phaseHours,
          startDate: row.startDate,
          // Marking it done ends it today, as in the Edit form
          endDate: status === 'done' && !row.endDate ? toDay(new Date()) : row.endDate,
        }),
      `${row.ticketKey}: ${what} saved`,
    )
  }
  const update = (id: string, input: CheckInInput) =>
    mutate(() => api.updateCheckIn(id, input), `Check-in on ${ticketKeyOf(input)} updated`)
  const remove = (id: string) => mutate(() => api.deleteCheckIn(id), 'Check-in deleted')
  const setStep = (row: TicketRow, step: string, label: string) =>
    mutate(() => api.setStep(row, step, row.stepRound), `${row.ticketKey}: on to ${label}`)
  const recordOutcome = (row: TicketRow, input: OutcomeInput, message: string) =>
    mutate(() => api.recordOutcome(row, input), `${row.ticketKey}: ${message}`)
  const removeTicket = (row: TicketRow) => mutate(() => api.deleteTicket(row), `Deleted ${row.ticketKey}`)
  // Deletes one round (e.g. a review-back logged or started by mistake): its
  // time, or the review-back itself if it has none yet. The ticket's other
  // rounds and details stay as they are.
  const removeRound = (row: TicketRow, round: number) =>
    mutate(() => api.deleteRound(row, round), `Deleted ${roundLabel(round)} of ${row.ticketKey}`)

  return {
    week,
    setWeek,
    data,
    isLoading,
    error,
    notice,
    clearNotice,
    create,
    addTicket,
    setParent,
    makeParent,
    makeChild,
    makeParentOf,
    setType,
    correct,
    saveCell,
    update,
    remove,
    removeTicket,
    removeRound,
    setStep,
    recordOutcome,
    exportUrl: api.exportUrl(week),
  }
}
