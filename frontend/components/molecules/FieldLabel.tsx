import styles from 'styles/FormField.module.css'

type Props = {
  label: string
  required?: boolean
  // What's wrong with the field, shown on the label's line (so it takes no
  // extra room), in red; '' or left out when it's fine
  error?: string
  // Worth knowing but not wrong (e.g. the ticket already exists), in the same
  // place in orange; an error takes its place
  warning?: string
  errorId: string
}

// A field's label row: its name, and on the right what's wrong with it, if
// anything. Shared by the text, select and text-area fields.
export default function FieldLabel(props: Props) {
  const { label, required = false, error, warning, errorId } = props
  const message = error || warning

  return (
    <span className={styles.labelRow}>
      <span className={styles.labelText}>{required ? `${label} *` : label}</span>
      <span
        id={errorId}
        className={!error && warning ? `${styles.fieldError} ${styles.fieldWarning}` : styles.fieldError}
        title={message || undefined}
        aria-live="polite"
      >
        {message}
      </span>
    </span>
  )
}
