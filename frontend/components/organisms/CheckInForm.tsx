import { FormEvent, useState } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import DurationField, { parseDuration } from 'components/molecules/DurationField'
import FormField from 'components/molecules/FormField'
import SelectField from 'components/molecules/SelectField'
import TextAreaField from 'components/molecules/TextAreaField'
import FormActions from 'components/molecules/FormActions'
import ReadOnlyField from 'components/molecules/ReadOnlyField'
import useOptions from 'hooks/useOptions'
import useTimer from 'hooks/useTimer'
import useAsyncAction from 'hooks/useAsyncAction'
import useTouched from 'hooks/useTouched'
import { formatClock, formatHoursMinutes, toClock, toDay, toISO } from 'lib/dates'
import { latestRound, roundLabel, roundPhaseHours } from 'lib/rounds'
import { isParentTicket, parseTicketInput, reposOf, sameTicket } from 'lib/tickets'
import { plural } from 'lib/text'
import { CheckIn, CheckInInput, StoppedTimer, TicketRow, TicketWorkInput } from 'types/worklog'

import styles from 'styles/Card.module.css'

type Props = {
  defaultProject: string
  // Tickets the user worked on recently, offered as suggestions
  knownTickets: TicketRow[]
  // When set, the form edits this check-in, which covers one phase.
  checkIn?: CheckIn
  // When set, the form corrects this row of the timeline: each phase box
  // starts at the time logged so far and saving sets the new totals.
  ticketRow?: TicketRow
  // Otherwise it logs new time on any of the ticket's phases at once.
  onLog: (input: TicketWorkInput) => Promise<void>
  onCorrect?: (input: TicketWorkInput) => Promise<void>
  onUpdate?: (input: CheckInInput) => Promise<void>
  onCancel?: () => void
}

// Phase hours as the text the boxes show, e.g. "2 h 00 m"
function toPhaseText(phaseHours: Record<string, number>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(phaseHours)
      .filter(([, h]) => h > 0)
      .map(([phase, h]) => [phase, formatHoursMinutes(h)]),
  )
}

export default function CheckInForm(props: Props) {
  const { defaultProject, knownTickets, checkIn, ticketRow, onLog, onCorrect, onUpdate, onCancel } = props
  const { options, maxHours } = useOptions()
  const source = checkIn ?? ticketRow
  const isCorrecting = !checkIn && ticketRow !== undefined

  const workedAt = checkIn ? new Date(checkIn.workedAt) : new Date()
  const [project, setProject] = useState(source?.project ?? defaultProject)
  const [ticket, setTicket] = useState(source?.ticket ?? '')
  // SPinno's repo (spec, backend or web); each numbers its own tickets
  const [repo, setRepo] = useState(source?.repo ?? '')
  // Time is logged on backend and web tickets; a spec's is theirs in total
  const repos = reposOf(options, project)
  const [title, setTitle] = useState(source?.title ?? '')
  const [phase, setPhase] = useState(checkIn?.phase ?? '')
  const [status, setStatus] = useState(source?.status ?? 'ongoing')
  // Which pass through the phases: a timeline row opens on its latest round
  const [round, setRound] = useState(checkIn?.round ?? latestRound(ticketRow))
  const [spent, setSpent] = useState(checkIn ? formatHoursMinutes(checkIn.hours) : '')
  // Time typed per phase; blank means not done yet. When correcting, the
  // boxes hold the chosen round's totals.
  const [phaseTime, setPhaseTime] = useState(() => toPhaseText(roundPhaseHours(ticketRow, round)))
  const [note, setNote] = useState(source?.note ?? '')
  // The ticket's start and end, as on the progress sheet. A new ticket
  // starts on the day the work is logged for.
  const [startDate, setStartDate] = useState(ticketRow?.startDate || toDay(workedAt))
  const [endDate, setEndDate] = useState(ticketRow?.endDate ?? '')
  const [date, setDate] = useState(toDay(workedAt))
  const [time, setTime] = useState(toClock(workedAt))
  // Saving; a failure is shown beside the button, where the user is looking
  const { run: submit, isBusy: isSubmitting, error: submitError } = useAsyncAction()

  const isBlocked = status === 'blocked'
  // Blank or zero counts as nothing; NaN fails the range check
  const minutesOf = (text: string) => parseDuration(text) ?? 0
  // Totals have no cap, but each entry does, so only what is added counts
  const inRange = (minutes: number) => minutes >= 0 && (isCorrecting || minutes <= maxHours * 60)
  const loggedMinutes = (p: string) => Math.round((roundPhaseHours(ticketRow, round)[p] ?? 0) * 60)

  const phases = options?.phases ?? []
  const typed = phases.map((p) => ({ ...p, minutes: minutesOf(phaseTime[p.value] ?? '') }))
  const filled = typed.filter((t) => t.minutes > 0)
  const badPhase = typed.find((t) => !inRange(t.minutes))
  const overAdded = isCorrecting ? typed.find((t) => t.minutes - loggedMinutes(t.value) > maxHours * 60) : undefined
  const totalMinutes = filled.reduce((sum, t) => sum + t.minutes, 0)
  const spentMinutes = minutesOf(spent)

  // The ticket's phase so far, so the user can spot where to type
  const known = ticketRow ?? knownTickets.find((t) => sameTicket(t, { project, repo, ticket: ticket.trim() }))

  // Rounds to choose from: those so far, plus starting the next revision
  const lastRound = Math.max(latestRound(known), checkIn?.round ?? 1)
  const roundOptions = Array.from({ length: lastRound + 1 }, (_, i) => {
    const r = i + 1
    return { value: String(r), label: r > lastRound ? `New revision (${roundLabel(r)})` : roundLabel(r) }
  })
  const chooseRound = (r: number) => {
    setRound(r)
    // Show what that round has so far, to correct it
    if (isCorrecting) setPhaseTime(toPhaseText(roundPhaseHours(ticketRow, r)))
  }

  // The first thing still missing, shown beside the button instead of
  // leaving it disabled without a reason
  // A parent ticket (a spec with no type) takes no time of its own
  const isParent = Boolean(known && isParentTicket(known))
  let missing = ''
  if (!source && repos.length > 0 && !repo) missing = `Choose the ${project} repo`
  else if (ticket.trim() === '') missing = 'Enter the ticket number'
  else if (isParent) missing = `#${ticket.trim()} is a parent ticket: log the time on the tickets under it`
  else if (title.trim() === '') missing = 'Add a short title'
  else if (checkIn && phase === '') missing = 'Pick the task'
  else if (checkIn && !(spentMinutes > 0 && inRange(spentMinutes))) missing = 'Enter the time spent'
  else if (!checkIn && badPhase) missing = `Fix the time for ${badPhase.label}`
  else if (overAdded) missing = `Add at most ${maxHours} h to ${overAdded.label} at a time`
  else if (!checkIn && filled.length === 0) missing = 'Enter the time for at least one phase'
  else if (isBlocked && note.trim() === '') missing = 'Say what is blocking the ticket'
  else if (!checkIn && startDate && endDate && endDate < startDate)
    missing = 'The end date can’t be before the start date'
  else if (date === '' || time === '') missing = 'Pick the date and time'
  const isValid = missing === ''

  // Each field's problem, shown beside it once it's been filled in or left
  const { touch, shown } = useTouched()
  const fieldErrors = {
    repo: !source && repos.length > 0 && !repo ? 'Choose a repo' : '',
    ticket: ticket.trim() === '' ? 'Enter the number' : isParent ? 'A parent ticket takes no time' : '',
    title: title.trim() === '' ? 'Add a short title' : '',
    phase: checkIn && phase === '' ? 'Pick the task' : '',
    end: !checkIn && startDate && endDate && endDate < startDate ? 'Before the start date' : '',
    date: date === '' ? 'Pick a date' : '',
    time: time === '' ? 'Pick a time' : '',
    note: isBlocked && note.trim() === '' ? 'Say what’s blocking it' : '',
  }

  let summary = missing
  if (isValid && checkIn) {
    summary = `${formatHoursMinutes(spentMinutes / 60)} on this check-in`
  } else if (isValid && ticketRow) {
    const was = typed.reduce((sum, t) => sum + loggedMinutes(t.value), 0)
    const ticketTotal = Math.round(ticketRow.totalHours * 60) - was + totalMinutes
    summary =
      `${roundLabel(round)}: ${formatHoursMinutes(totalMinutes / 60)} (was ${formatHoursMinutes(was / 60)})` +
      ` · ticket total ${formatHoursMinutes(ticketTotal / 60)}`
  } else if (isValid) {
    const count = plural(filled.length, 'phase')
    summary = `Total ${formatHoursMinutes(totalMinutes / 60)} across ${count}`
  }

  // Timing a phase instead of typing its time in
  const timers = useTimer()
  const canTime = project !== '' && ticket.trim() !== '' && title.trim() !== ''
  const isTiming = (phaseValue: string) => {
    const t = timers.timer
    return (
      t !== null &&
      sameTicket(t, { project, repo, ticket: ticket.trim() }) &&
      t.round === round &&
      t.phase === phaseValue
    )
  }
  // When correcting totals, time a timer just logged on this ticket and
  // round goes into its box too, so saving the form doesn't take it back out
  const addTimed = (stopped: StoppedTimer | null) => {
    const t = stopped?.timer
    if (
      !isCorrecting ||
      !t ||
      !stopped.minutes ||
      !sameTicket(t, { project, repo, ticket: ticket.trim() }) ||
      t.round !== round
    )
      return
    setPhaseTime((prev) => ({
      ...prev,
      [t.phase]: formatHoursMinutes((minutesOf(prev[t.phase] ?? '') + stopped.minutes) / 60),
    }))
  }
  const startTimer = (phaseValue: string) =>
    timers
      .start({ project, repo, ticket: ticket.trim(), title: title.trim(), round, phase: phaseValue })
      .then(addTimed)
      .catch(() => {}) // shown in a toast
  // Asks first, so a misclick can't end it
  const stopTimer = () =>
    timers
      .confirmStop()
      .then(addTimed)
      .catch(() => {})

  // Accepts "3221" or a suggestion like "BlackBox#3221" or "SPinno/web#120",
  // and fills in the title of a ticket the user already worked on.
  const handleTicketChange = (value: string) => {
    const { project: proj, repo: rep, ticket: number } = parseTicketInput(value, { project, repo }, options)
    setTicket(number)
    setProject(proj)
    setRepo(rep)
    const seen = knownTickets.find((t) => sameTicket(t, { project: proj, repo: rep, ticket: number }))
    if (!source) {
      // Carry on in the ticket's latest round; a new ticket starts at the original
      setRound(latestRound(seen))
      if (seen) setTitle(seen.title)
      // A ticket worked on before keeps its dates; a new one starts today
      setStartDate(seen?.startDate || date)
      setEndDate(seen?.endDate ?? '')
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!isValid) return

    const common = {
      project,
      repo,
      ticket: ticket.trim(),
      title: title.trim(),
      round,
      status,
      note: note.trim(),
      workedAt: toISO(date, time),
    }
    // On failure the input stays, so the user can retry
    await submit(async () => {
      const phaseHours = Object.fromEntries(filled.map((t) => [t.value, t.minutes / 60]))
      if (checkIn) {
        await onUpdate?.({ ...common, phase, hours: spentMinutes / 60 })
      } else if (isCorrecting) {
        await onCorrect?.({ ...common, phaseHours, startDate, endDate })
      } else {
        await onLog({ ...common, phaseHours, startDate, endDate })
        // Keep the ticket: the next check-in is often on the same one
        const now = new Date()
        setNote('')
        setPhaseTime({})
        setDate(toDay(now))
        setTime(toClock(now))
      }
    })
  }

  // Logging or correcting lists every phase, beside the details; editing one
  // check-in has a single phase, so it stays one column
  const hasPhaseList = !checkIn

  return (
    // Shown in a Modal, which gives it the title and frame
    <form onSubmit={handleSubmit}>
      <div className={hasPhaseList ? styles.formColumns : undefined}>
        {/* Every field is required unless marked optional; the line by the
            button says what's missing, so no asterisks */}
        <div className={styles.pairs}>
          <h3 className={styles.groupTitle}>Ticket</h3>
          {/* When correcting, the ticket is in the pop-up's title and can't change */}
          {!isCorrecting && (
            <>
              {/* Fixed: the project the user chose at signup (or the ticket's
                  own, when it's one they already have) */}
              <ReadOnlyField label="Project" value={project} title="Your team’s project" />
              {repos.length > 0 && (
                <SelectField
                  label="Repo"
                  value={repo}
                  onChange={(e) => {
                    setRepo(e.target.value)
                    touch('repo')
                  }}
                  onBlur={() => touch('repo')}
                  error={shown('repo', fieldErrors.repo)}
                  options={repos}
                  placeholder="Choose a repo"
                  aria-required
                />
              )}
              <FormField
                label="Ticket #"
                value={ticket}
                onChange={(e) => {
                  handleTicketChange(e.target.value)
                  touch('ticket')
                }}
                onBlur={() => touch('ticket')}
                error={shown('ticket', fieldErrors.ticket)}
                // No suggestions of tickets you have: it only warns when it's one
                warning={!source && known && !isParent ? 'Already in your timeline: time adds to it' : undefined}
                inputMode="numeric"
                data-autofocus={source ? undefined : true}
                aria-required
              />
            </>
          )}
          <div className={styles.wide}>
            <FormField
              label="Title / short description"
              value={title}
              onChange={(e) => {
                setTitle(e.target.value)
                touch('title')
              }}
              onBlur={() => touch('title')}
              error={shown('title', fieldErrors.title)}
              maxLength={200}
              aria-required
            />
          </div>

          <h3 className={styles.groupTitle}>Progress</h3>
          <SelectField
            label="Status"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value)
              // Finishing the ticket: end it on the day of this work, unless
              // an end date is already there
              if (e.target.value === 'done' && !endDate) setEndDate(date)
              // Blocked needs a note: say so beside it straight away
              touch('note')
            }}
            options={options?.statuses ?? []}
            aria-required
          />
          <div>
            <SelectField
              label="Round"
              value={String(round)}
              onChange={(e) => chooseRound(Number(e.target.value))}
              options={roundOptions}
              aria-required
            />
            <p className={styles.fieldHint}>
              {isCorrecting ? 'Its times are shown on the right' : '“New revision” if it was sent back for changes'}
            </p>
          </div>
          {hasPhaseList && (
            <>
              <FormField
                label="Start date"
                type="date"
                value={startDate}
                max={endDate || undefined}
                onChange={(e) => {
                  setStartDate(e.target.value)
                  touch('end')
                }}
              />
              <FormField
                label="End date (when done)"
                type="date"
                value={endDate}
                min={startDate || undefined}
                onChange={(e) => {
                  setEndDate(e.target.value)
                  touch('end')
                }}
                error={shown('end', fieldErrors.end)}
              />
            </>
          )}
          {checkIn && (
            <>
              <SelectField
                label="Task"
                value={phase}
                onChange={(e) => {
                  setPhase(e.target.value)
                  touch('phase')
                }}
                onBlur={() => touch('phase')}
                error={shown('phase', fieldErrors.phase)}
                options={phases}
                placeholder="What did you work on?"
                aria-required
              />
              <DurationField
                label="Time spent"
                value={spent}
                onChange={setSpent}
                maxHours={maxHours}
                placeholder="e.g. 1h30m"
              />
            </>
          )}

          <h3 className={styles.groupTitle}>Time logged</h3>
          <FormField
            label="Date worked"
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
          <div className={styles.wide}>
            <TextAreaField
              label={isBlocked ? 'What is blocking it?' : 'Note (optional)'}
              value={note}
              onChange={(e) => {
                setNote(e.target.value)
                touch('note')
              }}
              onBlur={() => touch('note')}
              error={shown('note', fieldErrors.note)}
              rows={2}
              maxLength={1000}
              required={isBlocked}
            />
          </div>
        </div>

        {hasPhaseList && (
          <fieldset className={styles.phases}>
            <legend>{isCorrecting ? `${roundLabel(round)}: total time per phase` : 'Time spent per phase'}</legend>
            <p className={styles.hint}>
              {isCorrecting ? (
                'Change a total to correct it; clear it to remove that time.'
              ) : (
                <>
                  Type <kbd>1h30m</kbd>, <kbd>1:30</kbd> or <kbd>90m</kbd>. Blank = not done yet.
                </>
              )}
            </p>
            {phases.map((p) => (
              <DurationField
                key={p.value}
                label={p.label}
                tag={known?.phase === p.value ? 'Current' : undefined}
                value={phaseTime[p.value] ?? ''}
                onChange={(text) => setPhaseTime((prev) => ({ ...prev, [p.value]: text }))}
                maxHours={isCorrecting ? undefined : maxHours}
                placeholder="—"
                inline
                status={
                  isTiming(p.value) && (
                    <span className={styles.timing}>
                      {timers.isPaused ? 'Paused' : 'Timing'} · {formatClock(timers.elapsed)} ·{' '}
                      <button
                        type="button"
                        className={styles.linkButton}
                        onClick={timers.togglePause}
                        disabled={timers.isBusy}
                      >
                        {timers.isPaused ? 'Resume' : 'Pause'}
                      </button>
                    </span>
                  )
                }
                action={
                  isTiming(p.value) ? (
                    <Button size="sm" variant="primary" onClick={stopTimer} disabled={timers.isBusy}>
                      <Icon name="stop" size={14} />
                      End
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => startTimer(p.value)}
                      disabled={!canTime || isParent || timers.isBusy}
                      title={
                        isParent
                          ? 'A parent ticket takes no time of its own'
                          : !canTime
                            ? 'Enter the ticket and title first'
                            : timers.timer
                              ? 'Ends the running timer, logging its time, and starts this one'
                              : `Time ${p.label}`
                      }
                    >
                      <Icon name="play" size={14} />
                      Start
                    </Button>
                  )
                }
              />
            ))}
          </fieldset>
        )}
      </div>
      <FormActions
        hint={submitError ? `Couldn’t save: ${submitError}` : summary}
        isError={Boolean(submitError)}
        onCancel={onCancel}
        isBusy={isSubmitting}
      >
        <Button type="submit" variant="primary" disabled={!isValid || isSubmitting}>
          {isSubmitting ? 'Saving…' : source ? 'Save changes' : 'Log check-in'}
        </Button>
      </FormActions>
    </form>
  )
}
