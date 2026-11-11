import Icon from 'components/atoms/Icon'

import styles from 'styles/Stepper.module.css'

export type StepState = 'done' | 'skipped' | 'current' | 'upcoming'

export type Step = {
  key: string
  label: string
  state: StepState
  detail?: string // under the label, e.g. the time logged
  // The table column the step's time goes in, shown before the detail;
  // "" for a step that takes no time
  column?: string
  // Why it can't be clicked, shown on hover, e.g. a finished step that
  // can't be reopened; left out, it can be
  locked?: string
  // The step being looked at, which may not be the current one: ringed
  viewing?: boolean
}

type Props = {
  steps: Step[]
  // Clicking a step moves to it; left out, the steps are display only
  onSelect?: (index: number) => void
  disabled?: boolean
}

// Numbered circles joined by a line that fills in up to the current step.
export default function Stepper(props: Props) {
  const { steps, onSelect, disabled = false } = props

  return (
    <ol className={styles.stepper}>
      {steps.map((step, i) => {
        const marker = (
          <>
            <span className={styles.circle} aria-hidden="true">
              {step.state === 'done' ? <Icon name="check" size={14} /> : i + 1}
            </span>
            <span className={styles.label}>{step.label}</span>
            {/* Always there (blank when there's nothing), so the steps keep
                one height whether or not time is logged */}
            <span className={styles.detail}>
              {step.column !== undefined && (
                <span
                  className={step.column ? styles.column : undefined}
                  title={step.column ? `Its time goes in the ${step.column} column` : 'Takes no time'}
                >
                  {step.column ? `→ ${step.column}` : 'No time'}
                </span>
              )}
              {step.column !== undefined && step.detail ? ' · ' : ''}
              {step.detail || (step.column === undefined ? '\u00a0' : '')}
            </span>
          </>
        )
        return (
          <li
            key={step.key}
            className={[styles.step, styles[step.state], step.viewing && styles.viewing].filter(Boolean).join(' ')}
          >
            {onSelect ? (
              <button
                type="button"
                className={styles.marker}
                onClick={() => onSelect(i)}
                // A step being viewed is where you are; when the steps say
                // which one is viewed, any other one can be looked at
                disabled={
                  disabled ||
                  Boolean(step.locked) ||
                  (steps.some((s) => s.viewing) ? Boolean(step.viewing) : step.state === 'current')
                }
                aria-current={step.state === 'current' ? 'step' : undefined}
                title={step.locked ? step.locked : step.viewing ? undefined : `View ${step.label}`}
              >
                {marker}
              </button>
            ) : (
              <span className={styles.marker} aria-current={step.state === 'current' ? 'step' : undefined}>
                {marker}
              </span>
            )}
          </li>
        )
      })}
    </ol>
  )
}
