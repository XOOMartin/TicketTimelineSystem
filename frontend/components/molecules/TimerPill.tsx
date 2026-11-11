import Link from 'next/link'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import useOptions from 'hooks/useOptions'
import useTimer from 'hooks/useTimer'
import { formatClock, mondayOf, toDay } from 'lib/dates'
import { shortPhase } from 'lib/phases'

import styles from 'styles/TimerPill.module.css'

// The running timer, in the page header wherever you are, with a button to
// end it. Nothing shows when no timer runs.
export default function TimerPill() {
  const { timer, elapsed, isPaused, isBusy, togglePause, end } = useTimer()
  const { options, phaseLabel } = useOptions()
  if (!timer) return null
  // A SPinno ticket's step (e.g. "Code"), or the phase's short name
  const stepLabel = timer.step
    ? Object.values(options?.workflows ?? {})
        .flatMap((w) => [...w.first, ...w.again, ...w.redo])
        .find((s) => s.key === timer.step)?.label
    : undefined

  // Opens the ticket's progress steps in My timeline, in the week the timer
  // started
  const week = mondayOf(toDay(new Date(timer.startedAt)))
  const href = `/timeline?week=${week}&progress=${encodeURIComponent(timer.ticketKey)}`

  return (
    <div className={isPaused ? `${styles.pill} ${styles.paused}` : styles.pill} role="status">
      <span className={styles.dot} aria-hidden="true" />
      <Link
        href={href}
        className={styles.what}
        title={`${phaseLabel(timer.phase)} · ${timer.title} — show its progress`}
      >
        {stepLabel ?? shortPhase(timer.phase, phaseLabel(timer.phase))} · {timer.ticketKey}
      </Link>
      <span className={styles.clock} aria-label={isPaused ? 'Paused at' : 'Time so far'}>
        {isPaused && 'Paused '}
        {formatClock(elapsed)}
      </span>
      {/* Ending before a minute records nothing, so say so until it has run one */}
      {elapsed < 60_000 && (
        <span className={styles.minimum} title="Time under a minute isn’t recorded">
          Min. 1 min
        </span>
      )}
      <Button
        size="sm"
        variant="ghost"
        onClick={togglePause}
        disabled={isBusy}
        aria-label={isPaused ? 'Resume the timer' : 'Pause the timer'}
        title={isPaused ? 'Resume' : 'Pause'}
      >
        <Icon name={isPaused ? 'play' : 'pause'} size={14} />
      </Button>
      {/* Asks first, so a misclick can't end it */}
      <Button size="sm" variant="primary" onClick={end} disabled={isBusy}>
        <Icon name="stop" size={14} />
        End
      </Button>
    </div>
  )
}
