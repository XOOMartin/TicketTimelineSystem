import { FormEvent } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import Select from 'components/atoms/Select'
import TextInput from 'components/atoms/TextInput'
import { today } from 'lib/dates'
import { ReportRange, defaultReportRange } from 'lib/reports'
import { ReportKind } from 'types/worklog'

import styles from 'styles/ReportsPage.module.css'

const KINDS: { value: ReportKind; label: string }[] = [
  { value: 'weekly', label: 'Weekly' },
  { value: 'daily', label: 'Daily' },
]

type Props = {
  range: ReportRange
  // Every change shows the report for it straight away, below
  onChange: (range: ReportRange) => void
  isGenerating: boolean
  // Saves the report for the days picked
  onGenerate: () => void
}

// One row: which report, its start and end date, then Generate. Picking
// Weekly or Daily fills in this week (Monday to Friday) or today; either
// date can then be changed.
export default function ReportGenerator(props: Props) {
  const { range, onChange, isGenerating, onGenerate } = props
  const { kind, start, end } = range

  const isBackwards = Boolean(start && end && end < start)
  const isFuture = Boolean(start && start > today())

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    onGenerate()
  }

  return (
    <form className={styles.generator} onSubmit={handleSubmit}>
      <span
        className={styles.info}
        title={
          'Weekly covers Monday to Friday, Daily one day. The report below updates as soon as you pick the type or ' +
          'the days, and Generate saves it to the list. Weekly reports (Monday to Friday) are also saved ' +
          'automatically every Friday at 17:00. Generating the same report and days again replaces the earlier one.'
        }
        aria-label="About generating reports"
      >
        <Icon name="help" size={16} />
      </span>
      <Select
        className={styles.kind}
        aria-label="Report"
        value={kind}
        onChange={(e) => onChange(defaultReportRange(e.target.value as ReportKind))}
        options={KINDS}
      />
      <label className={styles.dateField}>
        <span>Start</span>
        <TextInput
          className={styles.day}
          type="date"
          value={start}
          max={today()}
          onChange={(e) => {
            const value = e.target.value
            // Keep the range the right way round
            onChange({ kind, start: value, end: value && end && end < value ? value : end })
          }}
          required
        />
      </label>
      <label className={styles.dateField}>
        <span>End</span>
        <TextInput
          className={styles.day}
          type="date"
          value={end}
          min={start || undefined}
          onChange={(e) => onChange({ kind, start, end: e.target.value })}
          aria-invalid={isBackwards || undefined}
          title={isBackwards ? 'The end date is before the start date' : undefined}
          required
        />
      </label>
      <Button
        type="submit"
        variant="primary"
        disabled={isGenerating || !start || !end || isBackwards || isFuture}
        title={isFuture ? 'The report can’t start in the future' : 'Save this report to the list'}
      >
        <Icon name="plus" size={16} />
        {isGenerating ? 'Saving…' : 'Generate'}
      </Button>
    </form>
  )
}
