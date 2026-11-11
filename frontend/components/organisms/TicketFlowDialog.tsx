import { useState } from 'react'

import Badge from 'components/atoms/Badge'
import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import DurationField, { parseDuration } from 'components/molecules/DurationField'
import FormField from 'components/molecules/FormField'
import Modal from 'components/molecules/Modal'
import Stepper, { Step } from 'components/molecules/Stepper'
import useAsyncAction from 'hooks/useAsyncAction'
import useOptions from 'hooks/useOptions'
import useTimer from 'hooks/useTimer'
import { formatClock, formatShortDay, formatShortHours } from 'lib/dates'
import { shortPhase } from 'lib/phases'
import { latestRound, roundLabel, roundPhaseHours } from 'lib/rounds'
import { sameTicket } from 'lib/tickets'
import { currentStepOf, reviewBackStart, workflowOf } from 'lib/workflows'
import { plural } from 'lib/text'
import { FlowStep, Outcome, OutcomeInput, TicketRow } from 'types/worklog'

import styles from 'styles/TicketProgress.module.css'

type Props = {
  row: TicketRow
  onSetStep: (step: string, label: string) => Promise<void>
  onOutcome: (input: OutcomeInput, message: string) => Promise<void>
  onClose: () => void
  // Opened from Input timeline's Continue: Previous on the first step goes
  // back to that form
  onBack?: () => void
}

// The progress form for every ticket: its steps one at a time, as a SPinno
// ticket's type lays them out or as BlackBox's phases (lib/workflows). Your
// own work is timed (or its time typed in); a review by others takes the time
// it took, typed in, and its result; a quick action is one click. A review
// asking for changes goes back a step, or starts a revision.
export default function TicketFlowDialog(props: Props) {
  const { row, onSetStep, onOutcome, onClose, onBack } = props
  const { options, maxHours, phaseLabel, statusLabel, typeLabel: typeLabelOf } = useOptions()
  const timers = useTimer()

  // A SPinno ticket has a type with its own steps; a BlackBox one steps
  // through the phases, its current step being a phase
  const isTyped = Boolean(row.type)
  const workflow = options ? workflowOf(row, options.workflows) : undefined
  const typeLabel = isTyped ? typeLabelOf(row.repo, row.type ?? '') : row.project
  const round = Math.max(1, row.stepRound || latestRound(row))
  const currentKey = currentStepOf(row)
  // A revision the leads started redoes the original steps (its steps'
  // keys are their own); one Cho-san started has the type's revision steps
  const isRedo = round > 1 && Boolean(workflow?.redo.some((s) => s.key === currentKey))
  const steps = (round > 1 ? (isRedo ? workflow?.redo : workflow?.again) : workflow?.first) ?? []
  const isFinished = currentKey === ''
  const current = steps.findIndex((s) => s.key === currentKey)
  const step: FlowStep | undefined = steps[current]
  const nextStep = steps[current + 1]
  const backToDetails = current <= 0 && Boolean(onBack)
  const hours = roundPhaseHours(row, round)

  // The step being looked at: the current one unless Previous, Next or the
  // stepper moved the view. Looking doesn't change anything; only the
  // current step can be filled in. A finished ticket's own view is "Done",
  // after its steps.
  const home = isFinished ? steps.length : current
  const [viewed, setViewed] = useState<number | null>(null)
  const viewIndex = viewed ?? home
  const isPreview = viewIndex !== home
  const view = (index: number) => setViewed(index === home ? null : index)

  const [outcome, setOutcome] = useState<Outcome | null>(null)
  const [spent, setSpent] = useState('')
  const [comment, setComment] = useState('')
  const { run, isBusy, error } = useAsyncAction()
  // Runs a change; once it's saved, the form starts fresh on the ticket's
  // (new) current step. On failure it says why, keeping the input.
  const act = async (action: () => Promise<unknown>) => {
    if (!(await run(action))) return
    setOutcome(null)
    setSpent('')
    setComment('')
    setViewed(null)
  }

  const goTo = (index: number) => {
    const s = steps[index]
    if (s) act(() => onSetStep(s.key, s.label))
  }

  // A phase shared by two steps of the round (e.g. both leads reviews) can't
  // say which its time was, so only a step with its own phase shows it
  const phaseCount = (phase: string) => steps.filter((s) => s.kind !== 'action' && s.phase === phase).length
  const loggedOn = (s: FlowStep) => (s.kind !== 'action' && phaseCount(s.phase) === 1 ? (hours[s.phase] ?? 0) : 0)
  const stateOf = (i: number): Step['state'] =>
    isFinished || i < current ? 'done' : i === current ? 'current' : 'upcoming'
  // Any step can be looked at, done or still to come; what's recorded on it
  // can't be changed from there
  const stepper: Step[] = steps.map((s, i) => {
    const logged = loggedOn(s)
    return {
      key: s.key,
      label: s.label,
      state: stateOf(i),
      viewing: i === viewIndex,
      // Where its time goes in the table ('' for an action, which takes
      // none), left out when the step is named after it (a BlackBox phase)
      column:
        s.kind === 'action'
          ? ''
          : shortPhase(s.phase, phaseLabel(s.phase)) === s.label
            ? undefined
            : shortPhase(s.phase, phaseLabel(s.phase)),
      detail: logged > 0 ? formatShortHours(logged) : undefined,
    }
  })

  // Timing this step
  const t = timers.timer
  // (a typed ticket's timer is on its step; a BlackBox one's on the phase)
  const isTimingHere =
    t !== null && sameTicket(t, row) && t.round === round && (isTyped ? t.step === step?.key : t.phase === step?.phase)
  const startTimer = () =>
    step &&
    timers
      .start({
        project: row.project,
        repo: row.repo,
        ticket: row.ticket,
        title: row.title,
        round,
        phase: step.phase,
        step: isTyped ? step.key : undefined,
      })
      .catch(() => {})

  const spentMinutes = parseDuration(spent)
  // Over the most one entry can hold is flagged beside the box and can't be saved
  const spentIsValid =
    spentMinutes === null || (!Number.isNaN(spentMinutes) && spentMinutes >= 0 && spentMinutes <= maxHours * 60)
  const spentHours = (spentMinutes ?? 0) / 60

  // Finishes the step with a result (and the time typed in, if any)
  const finish = (result: Outcome, message: string) =>
    step &&
    act(async () => {
      // A work step still being timed ends first, so its time isn't lost
      if (isTimingHere) await timers.stop()
      await onOutcome({ step: step.key, outcome: result, round, hours: spentHours, comment, title: row.title }, message)
    })

  // What a review's results lead to
  const changesDetail = (s: FlowStep) => {
    if (s.changesNewRound && workflow)
      return `Starts ${roundLabel(round + 1)}: ${reviewBackStart(s, workflow)?.label ?? ''}`
    if (s.changesTo) return `Back to ${steps.find((x) => x.key === s.changesTo)?.label ?? s.changesTo}`
    if (s.changesRestart)
      return `Starts ${roundLabel(round + 1)}: ${workflow?.redo[0]?.label ?? ''}, then the steps again`
    return ''
  }
  const approveDetail = nextStep ? `Next: ${nextStep.label}` : 'Finishes the ticket'
  // A work step done with no time: a typed ticket's workflow records it; a
  // BlackBox ticket just moves on to its next phase
  const doneNoTime = () => (isTyped ? finish('done', `${step?.label} done`) : goTo(current + 1))

  return (
    <Modal title={`${row.ticketKey} · ${row.title}`} size="large" onClose={onClose}>
      <div className={styles.meta}>
        <Badge value={row.status}>{statusLabel(row.status)}</Badge>
        <span className={styles.typeChip}>{typeLabel}</span>
        {round > 1 && <span>{roundLabel(round)}</span>}
        <span>{row.totalHours > 0 ? `Total ${formatShortHours(row.totalHours)}` : 'No time logged yet'}</span>
      </div>

      <Stepper steps={stepper} onSelect={view} disabled={isBusy} />

      {/* One fixed size whatever the step (a timer, a review, an action,
          done), so the pop-up doesn't change size */}
      <section className={styles.panel}>
        {isPreview && steps[viewIndex] ? (
          // Another step than the current one, to look at: what it is and
          // what's recorded on it, nothing to fill in
          <>
            <div className={styles.panelHeader}>
              <div>
                <span className={styles.stepNumber}>
                  Step {viewIndex + 1} of {steps.length} · {steps[viewIndex].who}
                  <span className={styles.previewTag}>
                    <Icon name="search" size={12} />
                    Preview · read only
                  </span>
                </span>
                <h3>{steps[viewIndex].label}</h3>
                <p className={styles.hint}>{steps[viewIndex].hint}</p>
              </div>
            </div>
            <div className={styles.previewBody}>
              {stateOf(viewIndex) === 'done' ? (
                <>
                  <span className={`${styles.previewIcon} ${styles.previewDone}`}>
                    <Icon name="check" size={18} />
                  </span>
                  <div>
                    <strong>Done</strong>
                    <p className={styles.hint}>
                      {steps[viewIndex].kind === 'action'
                        ? 'A quick step: no time'
                        : loggedOn(steps[viewIndex]) > 0
                          ? `${formatShortHours(loggedOn(steps[viewIndex]))} logged on it`
                          : phaseCount(steps[viewIndex].phase) > 1
                            ? `Its time is in the ${shortPhase(steps[viewIndex].phase, phaseLabel(steps[viewIndex].phase))} column, shared with another step`
                            : 'No time logged on it'}
                    </p>
                  </div>
                </>
              ) : (
                <>
                  <span className={styles.previewIcon}>
                    <Icon name="clock" size={18} />
                  </span>
                  <div>
                    <strong>Not reached yet</strong>
                    <p className={styles.hint}>
                      {step ? `It comes after ${step.label}, the step the ticket is on now.` : 'Still to come.'}
                    </p>
                  </div>
                </>
              )}
            </div>
            <Button variant="outline" onClick={() => setViewed(null)}>
              <Icon name="restore" size={16} />
              Back to {isFinished ? 'the result' : `current step (${step?.label ?? ''})`}
            </Button>
          </>
        ) : isFinished ? (
          <div className={styles.finished}>
            <span className={styles.finishedIcon}>
              <Icon name="check" size={22} />
            </span>
            <div>
              <h3>Done</h3>
              <p className={styles.hint}>
                {row.endDate ? `Finished ${formatShortDay(row.endDate)} · ` : ''}
                {formatShortHours(row.totalHours)}
                {row.reviewBacks > 0 ? ` · ${plural(row.reviewBacks, 'revision')}` : ''}
              </p>
            </div>
          </div>
        ) : step ? (
          <>
            <div className={styles.panelHeader}>
              <div>
                <span className={styles.stepNumber}>
                  Step {current + 1} of {steps.length} · {step.who}
                </span>
                <h3>{step.label}</h3>
                <p className={styles.hint}>{step.hint}</p>
              </div>
            </div>

            {/* Your own work: timed, or its time typed in, side by side */}
            {step.kind === 'work' && (
              <div className={styles.ways}>
                <div className={isTimingHere ? `${styles.way} ${styles.wayActive}` : styles.way}>
                  <span className={styles.wayLabel}>Time it</span>
                  <div className={styles.timer}>
                    <span
                      className={isTimingHere && !timers.isPaused ? `${styles.clock} ${styles.running}` : styles.clock}
                    >
                      {formatClock(isTimingHere ? timers.elapsed : 0)}
                    </span>
                    {isTimingHere ? (
                      <div className={styles.timerButtons}>
                        <Button variant="outline" onClick={timers.togglePause} disabled={timers.isBusy}>
                          <Icon name={timers.isPaused ? 'play' : 'pause'} size={16} />
                          {timers.isPaused ? 'Resume' : 'Pause'}
                        </Button>
                        <Button variant="primary" onClick={timers.end} disabled={timers.isBusy}>
                          <Icon name="stop" size={16} />
                          End
                        </Button>
                      </div>
                    ) : (
                      <Button variant="primary" onClick={startTimer} disabled={timers.isBusy || isBusy}>
                        <Icon name="play" size={16} />
                        Start
                      </Button>
                    )}
                  </div>
                  <p className={styles.hint}>
                    {isTimingHere
                      ? timers.isPaused
                        ? 'Paused · not counting'
                        : `End records the time · Next: ${nextStep?.label ?? 'done'}`
                      : t
                        ? `Starting ends the timer on ${t.ticketKey}`
                        : 'Counts while you work'}
                  </p>
                </div>
                {!isTimingHere && (
                  <div className={`${styles.way} ${styles.typed}`}>
                    <DurationField
                      maxHours={maxHours}
                      label="Or enter the time"
                      value={spent}
                      onChange={setSpent}
                      placeholder="e.g. 1h30m"
                    />
                    <Button
                      variant="outline"
                      onClick={() => (spentHours > 0 ? finish('done', `${step.label} done`) : doneNoTime())}
                      disabled={!spentIsValid || isBusy}
                    >
                      <Icon name="check" size={16} />
                      {spentHours > 0 ? 'Save and continue' : 'Done, no time'}
                    </Button>
                  </div>
                )}
              </div>
            )}

            {step.kind === 'review' && (
              <div className={styles.form}>
                <div className={styles.reviewRow}>
                  <DurationField
                    maxHours={maxHours}
                    label="Time it took (optional)"
                    value={spent}
                    onChange={setSpent}
                    placeholder="e.g. 30m"
                  />
                  <div className={styles.result}>
                    <span className={styles.resultLabel}>Result</span>
                    <div className={styles.choices} role="radiogroup" aria-label="Result">
                      {[
                        { value: 'approved' as Outcome, title: step.approveLabel || 'Approved' },
                        ...(changesDetail(step)
                          ? [{ value: 'changes_requested' as Outcome, title: 'Changes requested' }]
                          : []),
                        // A result that keeps the step open, e.g. still waiting for an answer
                        ...(step.waitLabel ? [{ value: 'waiting' as Outcome, title: step.waitLabel }] : []),
                      ].map((c) => (
                        <label
                          key={c.value}
                          className={outcome === c.value ? `${styles.choice} ${styles.chosen}` : styles.choice}
                        >
                          <input
                            type="radio"
                            name="outcome"
                            value={c.value}
                            checked={outcome === c.value}
                            onChange={() => setOutcome(c.value)}
                          />
                          {c.title}
                        </label>
                      ))}
                    </div>
                    <p className={styles.resultHint}>
                      {outcome === 'approved'
                        ? approveDetail
                        : outcome === 'waiting'
                          ? 'Stays on this step; the ticket shows as Waiting'
                          : outcome
                            ? changesDetail(step)
                            : ''}
                    </p>
                  </div>
                </div>
                <div className={styles.commentRow}>
                  {/* No comment in the Confirm phase: just its time and result
                      (the empty span keeps the button on the right) */}
                  {step.phase === 'confirmation' ? (
                    <span />
                  ) : (
                    <FormField
                      label="Comment (optional)"
                      value={comment}
                      onChange={(e) => setComment(e.target.value)}
                      placeholder={outcome === 'changes_requested' ? 'What needs changing?' : ''}
                      maxLength={300}
                    />
                  )}
                  <Button
                    variant="primary"
                    onClick={() =>
                      outcome &&
                      finish(
                        outcome,
                        `${step.label}: ${
                          outcome === 'approved'
                            ? (step.approveLabel || 'approved').toLowerCase()
                            : outcome === 'waiting'
                              ? (step.waitLabel ?? 'waiting').toLowerCase()
                              : 'changes requested'
                        }`,
                      )
                    }
                    disabled={!outcome || !spentIsValid || isBusy}
                  >
                    {isBusy ? 'Saving…' : 'Record result'}
                  </Button>
                </div>
              </div>
            )}

            {step.kind === 'action' && (
              <div className={styles.formActions}>
                <p className={styles.hint}>{nextStep ? `Next: ${nextStep.label}` : 'This finishes the ticket'}</p>
                <Button variant="primary" onClick={() => finish('done', `${step.label} done`)} disabled={isBusy}>
                  <Icon name="check" size={16} />
                  {isBusy ? 'Saving…' : 'Mark as done'}
                </Button>
              </div>
            )}
          </>
        ) : (
          <p className={styles.hint}>Loading…</p>
        )}
      </section>

      {/* Previous and Next only move the view; on the first step, Previous
          goes back to the ticket's details when it came from there */}
      <footer className={styles.footer}>
        <Button
          variant="ghost"
          onClick={() => (viewIndex > 0 ? view(viewIndex - 1) : onBack?.())}
          disabled={isBusy || (viewIndex <= 0 && !backToDetails)}
          title={viewIndex <= 0 && backToDetails ? 'Back to the ticket details' : undefined}
        >
          <Icon name="chevronLeft" size={16} />
          Previous
        </Button>
        {error && (
          <p className={styles.footerError} role="alert" title={error}>
            {error}
          </p>
        )}
        <Button
          variant="ghost"
          onClick={() => view(viewIndex + 1)}
          disabled={isBusy || viewIndex >= (isFinished ? steps.length : steps.length - 1)}
        >
          Next
          <Icon name="chevronRight" size={16} />
        </Button>
      </footer>
    </Modal>
  )
}
