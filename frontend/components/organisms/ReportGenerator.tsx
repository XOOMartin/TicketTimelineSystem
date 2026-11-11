import { FormEvent, useState } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import Select from 'components/atoms/Select'
import TextInput from 'components/atoms/TextInput'
import { today } from 'lib/dates'
import { ReportKind } from 'types/worklog'

import styles from 'styles/ReportsPage.module.css'

const KINDS = [
  { value: 'weekly', label: 'Weekly' },
  { value: 'daily', label: 'Daily' },
]

type Props = {
  isGenerating: boolean
  onGenerate: (kind: ReportKind, day: string) => void
}

// One row: which report, a day in its period, Generate. What happens on its
// own (Fridays) is on the info icon rather than spelled out.
export default function ReportGenerator(props: Props) {
  const { isGenerating, onGenerate } = props
  const [kind, setKind] = useState<ReportKind>('weekly')
  const [day, setDay] = useState(today)

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    onGenerate(kind, day)
  }

  return (
    <form className={styles.generator} onSubmit={handleSubmit}>
      <span
        className={styles.info}
        title="Weekly reports are also generated automatically every Friday at 17:00. Generating a period again replaces its earlier report."
        aria-label="About generating reports"
      >
        <Icon name="help" size={16} />
      </span>
      <Select
        className={styles.kind}
        aria-label="Report"
        value={kind}
        onChange={(e) => setKind(e.target.value as ReportKind)}
        options={KINDS}
      />
      <TextInput
        className={styles.day}
        type="date"
        aria-label={kind === 'weekly' ? 'Any day in the week' : 'Day'}
        title={kind === 'weekly' ? 'Any day in the week (Mon–Sun)' : 'The day'}
        value={day}
        max={today()}
        onChange={(e) => setDay(e.target.value)}
        required
      />
      <Button type="submit" variant="primary" disabled={isGenerating || !day || day > today()}>
        <Icon name="plus" size={16} />
        {isGenerating ? 'Generating…' : 'Generate'}
      </Button>
    </form>
  )
}
