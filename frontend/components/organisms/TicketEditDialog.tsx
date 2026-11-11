import { FormEvent, useEffect, useRef, useState } from 'react'

import Badge from 'components/atoms/Badge'
import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import DurationField, { parseDuration } from 'components/molecules/DurationField'
import FormField from 'components/molecules/FormField'
import Modal from 'components/molecules/Modal'
import SelectField from 'components/molecules/SelectField'
import ParentTicketOptions from 'components/molecules/ParentTicketOptions'
import FormActions from 'components/molecules/FormActions'
import useOptions from 'hooks/useOptions'
import useAsyncAction from 'hooks/useAsyncAction'
import useTouched from 'hooks/useTouched'
import { formatHoursMinutes, formatShortHours, toClock, toDay, toISO } from 'lib/dates'
import { latestRound, roundLabel } from 'lib/rounds'
import { hasParentRepo, isParentTicket, isTicketNumber, parentNumberOf, reposOf, ticketDigits } from 'lib/tickets'
import { TicketRow, TicketWorkInput } from 'types/worklog'

import styles from 'styles/TicketProgress.module.css'

type Props = {
  row: TicketRow
  // The round whose tab opens first (from that round's Edit in the
  // breakdown); otherwise the one the ticket is on
  initialRound?: number
  // The field to start in (double-clicked in the table): 'title', 'note',
  // 'status', 'start' or a phase
  initialField?: string
  // Saves one round's totals and the ticket's details
  onCorrect: (input: TicketWorkInput) => Promise<void>
  // Moves a SPinno backend or web ticket under another spec ticket (its
  // number, '' for none); specs are the user's spec tickets, as suggestions
  onSetParent?: (row: TicketRow, parent: string) => Promise<void>
  specs?: TicketRow[]
  // Changes a SPinno ticket's type (its steps change with it)
  onSetType?: (row: TicketRow, type: string) => Promise<void>
  onClose: () => void
}

// Phase hours as the text the box shows, e.g. "2 h 00 m"
const toText = (phaseHours: Record<string, number>): Record<string, string> =>
  Object.fromEntries(
    Object.entries(phaseHours)
      .filter(([, h]) => h > 0)
      .map(([phase, h]) => [phase, formatHoursMinutes(h)]),
  )

const minutesOf = (text: string | undefined) => parseDuration(text ?? '') ?? 0
const hoursToMinutes = (h: number | undefined) => Math.round((h ?? 0) * 60)

// Corrects a ticket's times: a tab per round (the original work and each
// revision), with every phase's total for that round in view at once, and
// the ticket's details underneath.
export default function TicketEditDialog(props: Props) {
  const { row, initialRound, initialField, onCorrect, onSetParent, specs = [], onSetType, onClose } = props
  const { options, maxHours, statusLabel } = useOptions()
  const phases = options?.phases ?? []

  // A double-clicked phase: its box appears once the phases have loaded, after
  // the pop-up's first focus, so it is focused (contents selected, ready to
  // type over) when it does
  const timesRef = useRef<HTMLElement>(null)
  const startsInPhase = phases.some((p) => p.value === initialField)
  useEffect(() => {
    if (!startsInPhase) return
    const input = timesRef.current?.querySelector<HTMLInputElement>('[data-autofocus]')
    input?.focus()
    input?.select()
  }, [startsInPhase])

  const saved = Object.fromEntries((row.rounds ?? []).map((r) => [r.round, r.phaseHours]))
  const lastSaved = latestRound(row)
  // The rounds being edited: those saved, plus a new one if added
  const [rounds, setRounds] = useState(() => (row.rounds?.length ? row.rounds.map((r) => r.round) : [1]))
  const [round, setRound] = useState(() =>
    initialRound && rounds.includes(initialRound) ? initialRound : Math.min(row.stepRound || lastSaved, lastSaved),
  )
  const [times, setTimes] = useState<Record<number, Record<string, string>>>(() =>
    Object.fromEntries((row.rounds ?? []).map((r) => [r.round, toText(r.phaseHours)])),
  )

  const now = new Date()
  const [title, setTitle] = useState(row.title)
  const [status, setStatus] = useState(row.status)
  const [startDate, setStartDate] = useState(row.startDate)
  const [endDate, setEndDate] = useState(row.endDate)
  const [note, setNote] = useState(row.note)
  // The parent ticket a child sits under; a parent ticket (a spec with no
  // type) goes under nothing
  const isChild = hasParentRepo(row) && Boolean(onSetParent) && !isParentTicket(row)
  const [parent, setParent] = useState(row.parent ?? '')
  const parentNumber = parentNumberOf(parent)
  const parentChanged = isChild && parentNumber !== (row.parent ?? '')
  // A SPinno ticket's type: changing it starts its steps over in the new
  // type. A parent ticket (a spec with no type, put under no spec) has none.
  const isParentLevel = isParentTicket(row) && parentNumber === ''
  // A parent ticket (a spec with no type) takes no time of its own
  const isParentRow = isParentTicket(row)
  const types = isParentLevel ? [] : (options?.ticketTypes[row.repo] ?? [])
  const [type, setType] = useState(row.type ?? '')
  const typeChanged = Boolean(onSetType) && types.length > 0 && type !== '' && type !== (row.type ?? '')
  const repoLabel = reposOf(options, row.project).find((r) => r.value === row.repo)?.label
  const [date, setDate] = useState(toDay(now))
  const [time, setTime] = useState(toClock(now))
  // Busy until the dialog closes on a save
  const { run, isBusy: isSaving, error } = useAsyncAction()

  const typed = (r: number, p: string) => minutesOf(times[r]?.[p])
  const was = (r: number, p: string) => hoursToMinutes(saved[r]?.[p])
  const roundTotal = (r: number, read: (r: number, p: string) => number) =>
    phases.reduce((sum, p) => sum + read(r, p.value), 0)

  const changedRounds = rounds.filter((r) => phases.some((p) => typed(r, p.value) !== was(r, p.value)))
  // A done ticket has passed its review, so it can't be sent back again
  const isDone = status === 'done'
  const addsRound = changedRounds.some((r) => r > lastSaved)
  const ticketWas = hoursToMinutes(row.totalHours)
  const ticketNow = ticketWas + changedRounds.reduce((sum, r) => sum + roundTotal(r, typed) - roundTotal(r, was), 0)
  const detailsChanged =
    title.trim() !== row.title ||
    status !== row.status ||
    startDate !== row.startDate ||
    endDate !== row.endDate ||
    note.trim() !== row.note

  // The first thing standing in the way of saving
  let missing = ''
  const bad = rounds.flatMap((r) =>
    phases.filter((p) => Number.isNaN(parseDuration(times[r]?.[p.value] ?? '') ?? 0)).map((p) => ({ r, p })),
  )[0]
  const tooMuch = rounds.flatMap((r) =>
    phases.filter((p) => typed(r, p.value) - was(r, p.value) > maxHours * 60).map((p) => ({ r, p })),
  )[0]
  if (title.trim() === '') missing = 'Add a short title'
  else if (isDone && addsRound) missing = 'A done ticket can’t get a new revision: change its status first'
  else if (bad) missing = `Fix the time for ${bad.p.label} in ${roundLabel(bad.r)}`
  else if (tooMuch) missing = `Add at most ${maxHours} h to ${tooMuch.p.label} at a time`
  // A ticket added without time can stay at none; once it has time, it keeps some
  else if (ticketNow <= 0 && ticketWas > 0) missing = 'The ticket needs some time on at least one phase'
  else if (status === 'blocked' && note.trim() === '') missing = 'Say what is blocking the ticket in the note'
  else if (startDate && endDate && endDate < startDate) missing = 'The end date can’t be before the start date'
  else if (parentChanged && parentNumber !== '' && !isTicketNumber(parentNumber))
    missing = 'The parent spec is a ticket number, like 120'
  else if (!changedRounds.length && !detailsChanged && !parentChanged && !typeChanged) missing = 'No changes yet'

  // Each detail's problem, shown beside it as it's changed
  const { touch, shown } = useTouched()
  const fieldErrors = {
    title: title.trim() === '' ? 'Add a short title' : '',
    note: status === 'blocked' && note.trim() === '' ? 'Say what’s blocking it' : '',
    end: startDate && endDate && endDate < startDate ? 'Before the start date' : '',
    date: date === '' ? 'Pick a date' : '',
    time: time === '' ? 'Pick a time' : '',
  }

  let summary = missing
  if (!missing) {
    const hours = (minutes: number) => formatShortHours(minutes / 60)
    const parts = changedRounds.map(
      (r) => `${roundLabel(r)} ${hours(roundTotal(r, was))} → ${hours(roundTotal(r, typed))}`,
    )
    parts.push(`ticket total ${hours(ticketNow)}`)
    if (typeChanged) parts.push(`type → ${types.find((t) => t.value === type)?.label ?? type} (its steps start over)`)
    summary = parts.join(' · ')
  }

  const addRound = () => {
    const next = Math.max(...rounds) + 1
    setRounds((prev) => [...prev, next])
    setTimes((prev) => ({ ...prev, [next]: {} }))
    setRound(next)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (missing) return
    const saved = await run(
      async () => {
        // One save per round that changed; details go with each. With only
        // details changed, they go with the round on screen. A new parent
        // spec is saved on its own.
        const rounds = changedRounds.length ? changedRounds : detailsChanged ? [round] : []
        for (const r of rounds) {
          await onCorrect({
            project: row.project,
            repo: row.repo,
            ticket: row.ticket,
            title: title.trim(),
            round: r,
            status,
            note: note.trim(),
            workedAt: toISO(date, time),
            phaseHours: Object.fromEntries(
              phases.filter((p) => typed(r, p.value) > 0).map((p) => [p.value, typed(r, p.value) / 60]),
            ),
            startDate,
            endDate,
          })
        }
        if (parentChanged && onSetParent) await onSetParent(row, parentNumber)
        if (typeChanged && onSetType) await onSetType(row, type)
      },
      { stayBusy: true },
    )
    if (saved) onClose()
  }

  return (
    <Modal title={`Edit ${row.ticketKey} · ${row.title}`} size="large" onClose={onClose}>
      <form onSubmit={handleSubmit}>
        <div className={styles.meta}>
          <Badge value={row.status}>{statusLabel(row.status)}</Badge>
          {/* A parent's total is its tickets' */}
          <span>
            Total {formatShortHours(row.totalHours + (isParentRow ? (row.childHours ?? 0) : 0))}
            {isParentRow ? ' from its tickets' : ''}
          </span>
          {repoLabel && <span>{repoLabel}</span>}
          {row.parent && <span>Under parent #{row.parent}</span>}
        </div>

        {isParentRow ? (
          // A parent takes no time of its own, so there's none to edit
          <p className={styles.hint}>
            A parent ticket takes no time of its own: its total is the time on the tickets under it.
          </p>
        ) : (
          <>
            {/* The revision timeline: one tab per round */}
            <div className={styles.roundTabs} role="tablist" aria-label="Round">
              {rounds.map((r) => (
                <button
                  key={r}
                  type="button"
                  role="tab"
                  aria-selected={r === round}
                  className={r === round ? `${styles.roundTab} ${styles.activeTab}` : styles.roundTab}
                  onClick={() => setRound(r)}
                >
                  {roundLabel(r)}
                  <span className={styles.tabTotal}>{formatShortHours(roundTotal(r, typed) / 60)}</span>
                </button>
              ))}
              {Math.max(...rounds) <= lastSaved && (
                <button
                  type="button"
                  className={`${styles.roundTab} ${styles.addTab}`}
                  onClick={addRound}
                  disabled={isDone}
                  title={isDone ? 'This ticket is done. Change its status to add a revision.' : undefined}
                >
                  <Icon name="plus" size={14} />
                  New revision
                </button>
              )}
            </div>

            <section ref={timesRef} className={styles.times} aria-label={`${roundLabel(round)}: total time per phase`}>
              {phases.map((p) => {
                const isChanged = typed(round, p.value) !== was(round, p.value)
                return (
                  <DurationField
                    // A fresh box per round, so switching tabs shows that round's times
                    key={`${round}-${p.value}`}
                    label={p.label}
                    value={times[round]?.[p.value] ?? ''}
                    onChange={(text) => setTimes((prev) => ({ ...prev, [round]: { ...prev[round], [p.value]: text } }))}
                    placeholder="—"
                    status={isChanged ? `was ${formatShortHours(was(round, p.value) / 60)}` : undefined}
                    autoFocus={initialField === p.value}
                  />
                )
              })}
            </section>
          </>
        )}

        <section className={styles.details}>
          <h3 className={styles.detailsTitle}>Ticket details</h3>
          <div className={styles.detailsGrid}>
            <FormField
              label="Title / short description"
              value={title}
              data-autofocus={initialField === 'title' || undefined}
              onChange={(e) => {
                setTitle(e.target.value)
                touch('title')
              }}
              onBlur={() => touch('title')}
              error={shown('title', fieldErrors.title)}
              maxLength={200}
              aria-required
            />
            <SelectField
              label="Status"
              value={status}
              data-autofocus={initialField === 'status' || undefined}
              onChange={(e) => {
                setStatus(e.target.value)
                if (e.target.value === 'done' && !endDate) setEndDate(date)
                // Only a done ticket has ended: set back to e.g. Ongoing, the end goes
                if (e.target.value !== 'done') setEndDate('')
                // Blocked needs a note: say so beside it straight away
                touch('note')
              }}
              options={options?.statuses ?? []}
              aria-required
            />
            <FormField
              label="Start date"
              type="date"
              value={startDate}
              data-autofocus={initialField === 'start' || undefined}
              max={endDate || undefined}
              onChange={(e) => {
                setStartDate(e.target.value)
                touch('end')
              }}
            />
            <FormField
              label="End date"
              type="date"
              value={endDate}
              data-autofocus={initialField === 'end' || undefined}
              min={startDate || undefined}
              onChange={(e) => {
                setEndDate(e.target.value)
                touch('end')
              }}
              error={shown('end', fieldErrors.end)}
            />
            <FormField
              label={status === 'blocked' ? 'What is blocking it?' : 'Note'}
              value={note}
              data-autofocus={initialField === 'note' || undefined}
              onChange={(e) => {
                setNote(e.target.value)
                touch('note')
              }}
              onBlur={() => touch('note')}
              error={shown('note', fieldErrors.note)}
              maxLength={1000}
            />
            {onSetType && types.length > 0 && (
              <SelectField
                label="Type"
                value={type}
                onChange={(e) => setType(e.target.value)}
                options={types}
                title="Changing it starts the ticket's steps over in the new type's"
              />
            )}
            {isChild && (
              <>
                <FormField
                  label="Parent ticket #"
                  value={parent}
                  data-autofocus={initialField === 'parent' || undefined}
                  onChange={(e) => setParent(ticketDigits(e.target.value))}
                  inputMode="numeric"
                  list="edit-spec-tickets"
                  placeholder="None"
                />
                <ParentTicketOptions id="edit-spec-tickets" tickets={specs} />
              </>
            )}
            {/* When added time is logged: a parent has none to add */}
            {!isParentRow && (
              <>
                <FormField
                  label="Date for added time"
                  type="date"
                  value={date}
                  onChange={(e) => {
                    setDate(e.target.value)
                    touch('date')
                  }}
                  error={shown('date', fieldErrors.date)}
                  aria-required
                />
                <FormField
                  label="Time"
                  type="time"
                  value={time}
                  onChange={(e) => {
                    setTime(e.target.value)
                    touch('time')
                  }}
                  error={shown('time', fieldErrors.time)}
                  aria-required
                />
              </>
            )}
          </div>
        </section>

        <FormActions
          hint={error ? `Couldn’t save: ${error}` : summary}
          isError={Boolean(error)}
          onCancel={onClose}
          isBusy={isSaving}
        >
          <Button type="submit" variant="primary" disabled={Boolean(missing) || isSaving}>
            {isSaving ? 'Saving…' : 'Save changes'}
          </Button>
        </FormActions>
      </form>
    </Modal>
  )
}
