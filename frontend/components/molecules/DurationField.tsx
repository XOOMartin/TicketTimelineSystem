import { ReactNode, useId } from 'react'

import TextInput from 'components/atoms/TextInput'
import { formatHoursMinutes } from 'lib/dates'

import styles from 'styles/DurationField.module.css'

const HOURS = '(\\d+(?:\\.\\d+)?)'
const PATTERNS: [RegExp, (m: RegExpMatchArray) => number][] = [
  // 3:20
  [/^(\d+):(\d{1,2})$/, (m) => (Number(m[2]) < 60 ? Number(m[1]) * 60 + Number(m[2]) : NaN)],
  // 3h, 3h20, 3h20m, 1.5h
  [new RegExp(`^${HOURS}h(?:(\\d+)m?)?$`), (m) => Number(m[1]) * 60 + Number(m[2] ?? 0)],
  // 200m
  [/^(\d+)m$/, (m) => Number(m[1])],
  // 3.5, read as hours
  [new RegExp(`^${HOURS}$`), (m) => Number(m[1]) * 60],
]

// Minutes in text such as "3h20m", "3h 20m", "3:20", "3.5" (hours) or
// "200m". null when blank, NaN when it can't be read.
export function parseDuration(text: string): number | null {
  const s = text
    .toLowerCase()
    .replace(/\s+/g, '')
    .replace(/hours?|hrs?/g, 'h')
    .replace(/minutes?|mins?/g, 'm')
  if (s === '') return null
  for (const [pattern, toMinutes] of PATTERNS) {
    const match = s.match(pattern)
    if (match) return Math.round(toMinutes(match))
  }
  return NaN
}

type Props = {
  label: string
  value: string
  onChange: (value: string) => void
  // No cap when left out, e.g. for a ticket's total time
  maxHours?: number
  // A short tag beside the label, e.g. the ticket's current phase
  tag?: string
  placeholder?: string
  required?: boolean
  // Label beside the box, for a list of phases; otherwise label above
  inline?: boolean
  // A button after the box, e.g. to start a timer on this phase
  action?: ReactNode
  // Shown under the box in place of the usual echo, e.g. a running clock
  status?: ReactNode
  // The box to start in when its pop-up opens
  autoFocus?: boolean
}

// A time-spent box that reads what people naturally type and echoes it back
// as "3 h 20 m", or says what's wrong.
export default function DurationField(props: Props) {
  const { label, value, onChange, maxHours, tag, placeholder, required = false, inline = false } = props
  const { action, status, autoFocus = false } = props
  const inputId = useId()
  const messageId = useId()

  const minutes = parseDuration(value)
  let message = ''
  let isError = false
  if (minutes !== null && Number.isNaN(minutes)) {
    message = 'Try 1h30m, 1:30 or 90m'
    isError = true
  } else if (minutes !== null && maxHours !== undefined && minutes > maxHours * 60) {
    // A bare "30" was most likely meant as minutes
    message = /^\s*\d+\s*$/.test(value)
      ? `Over ${maxHours} h — for minutes, type ${value.trim()}m`
      : `At most ${maxHours} h`
    isError = true
  } else if (minutes && value.trim() !== formatHoursMinutes(minutes / 60)) {
    // Only when it adds something: "1:30" echoes, "1 h 30 m" already says it
    message = `= ${formatHoursMinutes(minutes / 60)}`
  }

  // A div rather than a label wrapping everything, as the action is a button
  return (
    <div className={inline ? `${styles.field} ${styles.inline}` : styles.field}>
      <label htmlFor={inputId} className={styles.label}>
        {required ? `${label} *` : label}
        {tag && <span className={styles.tag}>{tag}</span>}
      </label>
      <TextInput
        id={inputId}
        className={styles.input}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete="off"
        spellCheck={false}
        required={required}
        aria-invalid={isError}
        aria-describedby={message || status ? messageId : undefined}
        data-autofocus={autoFocus || undefined}
      />
      {action}
      <span
        id={messageId}
        className={isError ? `${styles.message} ${styles.error}` : styles.message}
        aria-live="polite"
      >
        {isError ? message : (status ?? message)}
      </span>
    </div>
  )
}
