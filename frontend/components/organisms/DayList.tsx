import Message from 'components/atoms/Message'
import DayListItem from 'components/molecules/DayListItem'
import { DaySummary } from 'types/progress'

import styles from 'styles/DayList.module.css'

type Props = {
  days: DaySummary[]
  selectedDay: string | null
  onSelect: (day: string) => void
}

export default function DayList(props: Props) {
  const { days, selectedDay, onSelect } = props

  return (
    <nav className={styles.panel} aria-label="Days">
      <div className={styles.header}>
        <span>Day</span>
        <span>Prompts</span>
      </div>
      {days.length === 0 ? (
        <Message>No prompts found yet.</Message>
      ) : (
        <ul className={styles.list}>
          {days.map((summary) => (
            <DayListItem
              key={summary.day}
              summary={summary}
              isSelected={summary.day === selectedDay}
              onSelect={onSelect}
            />
          ))}
        </ul>
      )}
    </nav>
  )
}
