import { useState } from 'react'

import Button from 'components/atoms/Button'
import DurationField, { parseDuration } from 'components/molecules/DurationField'
import FormActions from 'components/molecules/FormActions'
import FormField from 'components/molecules/FormField'
import Modal from 'components/molecules/Modal'
import useOptions from 'hooks/useOptions'
import { formatShortHours } from 'lib/dates'
import { errorMessage } from 'lib/errors'
import { latestRound, roundLabel, roundPhaseHours } from 'lib/rounds'
import { reviewBackStart, workflowOf } from 'lib/workflows'
import { FlowStep, OutcomeInput, TicketRow } from 'types/worklog'

import styles from 'styles/AddRevisionDialog.module.css'

type Props = {
  row: TicketRow
  // Records the review's "Changes requested", which starts the revision
  onRecord: (input: OutcomeInput, message: string) => Promise<void>
  onClose: () => void
}

// A revision only starts because a review asked for changes. So adding one
// records that: which review (the Confirm phase or the final Review) asked,
// and the time it took, asked for when that review has no time yet in this
// round. The ticket then moves into the revision, where the review sends it.
export default function AddRevisionDialog(props: Props) {
  const { row, onRecord, onClose } = props
  const { options, maxHours } = useOptions()
  const round = latestRound(row)
  const next = roundLabel(round + 1)

  const workflow = options ? workflowOf(row, options.workflows) : undefined
  // The reviews of this round whose "Changes requested" starts the next round
  const reviews: FlowStep[] = ((round === 1 ? workflow?.first : workflow?.again) ?? []).filter(
    (s) => s.kind === 'review' && (s.changesNewRound || s.changesRestart),
  )
  // The final review to start with: it's the one that usually sends work back
  const [stepKey, setStepKey] = useState(() => reviews[reviews.length - 1]?.key ?? '')
  const step = reviews.find((s) => s.key === stepKey) ?? reviews[reviews.length - 1]

  const [spent, setSpent] = useState('')
  const [comment, setComment] = useState('')
  const [error, setError] = useState('')
  const [isBusy, setIsBusy] = useState(false)

  // Time this review already has in this round; without any, it's asked for
  const logged = step ? (roundPhaseHours(row, round)[step.phase] ?? 0) : 0
  const needsTime = logged <= 0
  const spentMinutes = parseDuration(spent)
  const spentIsValid =
    spentMinutes === null || (!Number.isNaN(spentMinutes) && spentMinutes >= 0 && spentMinutes <= maxHours * 60)
  const spentHours = (spentMinutes ?? 0) / 60
  const canSave = Boolean(step) && spentIsValid && (!needsTime || spentHours > 0) && !isBusy

  // Where the revision starts, as the review sends it
  const startsAt =
    step && workflow
      ? step.changesNewRound
        ? reviewBackStart(step, workflow)?.label
        : workflow.redo[0]?.label
      : undefined

  const save = async () => {
    if (!step || !canSave) return
    setIsBusy(true)
    setError('')
    try {
      await onRecord(
        {
          step: step.key,
          outcome: 'changes_requested',
          round,
          hours: spentHours,
          comment: comment.trim(),
          title: row.title,
        },
        `${step.label}: changes requested, ${next} added`,
      )
      onClose()
    } catch (e) {
      setError(errorMessage(e))
      setIsBusy(false)
    }
  }

  return (
    <Modal title={`Add ${next} to ${row.ticketKey}`} onClose={onClose}>
      <p className={styles.intro}>
        A revision starts when a review asks for changes. Which review was it, and how long did it take?
      </p>

      {reviews.length === 0 ? (
        <p className={styles.intro}>This ticket’s reviews can’t start a revision.</p>
      ) : (
        <div className={styles.form}>
          {reviews.length > 1 && (
            <div className={styles.field}>
              <span className={styles.label}>Changes requested by</span>
              <div className={styles.choices} role="radiogroup" aria-label="Changes requested by">
                {reviews.map((s) => (
                  <label key={s.key} className={s.key === step?.key ? `${styles.choice} ${styles.chosen}` : styles.choice}>
                    <input
                      type="radio"
                      name="review"
                      value={s.key}
                      checked={s.key === step?.key}
                      onChange={() => setStepKey(s.key)}
                    />
                    {s.label}
                  </label>
                ))}
              </div>
            </div>
          )}

          <DurationField
            maxHours={maxHours}
            label={needsTime ? `How long did ${step?.label ?? 'the review'} take?` : 'More time on it (optional)'}
            tag={needsTime ? undefined : `${formatShortHours(logged)} logged`}
            value={spent}
            onChange={setSpent}
            placeholder="e.g. 30m"
            required={needsTime}
            autoFocus
          />

          <FormField
            label="What needs changing? (optional)"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            maxLength={300}
          />

          {startsAt && (
            <p className={styles.next}>
              {next} starts at <strong>{startsAt}</strong>.
            </p>
          )}
        </div>
      )}

      <FormActions
        onCancel={onClose}
        isBusy={isBusy}
        isError={Boolean(error)}
        hint={error || (needsTime && step && spentHours <= 0 ? `Enter the time ${step.label} took` : undefined)}
      >
        <Button variant="primary" onClick={save} disabled={!canSave}>
          {isBusy ? 'Adding…' : `Add ${next}`}
        </Button>
      </FormActions>
    </Modal>
  )
}
