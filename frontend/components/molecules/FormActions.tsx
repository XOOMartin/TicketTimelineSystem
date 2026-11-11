import { ReactNode } from 'react'

import Button from 'components/atoms/Button'

import styles from 'styles/Card.module.css'

type Props = {
  // A line on the left: what's missing, what saving will do, or (isError)
  // why it couldn't be saved
  hint?: ReactNode
  isError?: boolean
  // A short hint that gives way to three buttons
  compact?: boolean
  // Shows Cancel (or cancelLabel), off while the form is busy
  onCancel?: () => void
  cancelLabel?: string
  isBusy?: boolean
  // The form's own buttons, after Cancel
  children: ReactNode
}

// The row under a form or dialog: its hint or error, then Cancel and the
// form's buttons, on the right.
export default function FormActions(props: Props) {
  const { hint, isError = false, compact = false, onCancel, cancelLabel = 'Cancel', isBusy = false, children } = props
  const hintClass = [styles.hint, compact && styles.compact, isError && styles.error].filter(Boolean).join(' ')

  return (
    <div className={styles.actions}>
      {hint !== undefined && (
        <p className={hintClass} aria-live="polite">
          {hint}
        </p>
      )}
      {onCancel && (
        <Button variant="ghost" onClick={onCancel} disabled={isBusy}>
          {cancelLabel}
        </Button>
      )}
      {children}
    </div>
  )
}
