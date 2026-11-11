import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import { addDays, formatDay, mondayOf, today } from 'lib/dates'

import styles from 'styles/WeekPicker.module.css'

type Props = {
  week: string // the Monday
  onChange: (week: string) => void
}

export default function WeekPicker(props: Props) {
  const { week, onChange } = props

  const thisWeek = mondayOf(today())

  return (
    <div className={styles.picker}>
      <Button variant="ghost" size="icon" onClick={() => onChange(addDays(week, -7))} aria-label="Previous week">
        <Icon name="chevronLeft" />
      </Button>
      <span className={styles.label}>
        {formatDay(week)} – {formatDay(addDays(week, 6))}
      </span>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => onChange(addDays(week, 7))}
        disabled={week >= thisWeek}
        aria-label="Next week"
      >
        <Icon name="chevronRight" />
      </Button>
      {week !== thisWeek && (
        <Button size="sm" onClick={() => onChange(thisWeek)}>
          This week
        </Button>
      )}
    </div>
  )
}
