import { DaySummary } from 'types/progress'

import styles from 'styles/DayListItem.module.css'

type Props = {
  summary: DaySummary
  isSelected: boolean
  onSelect: (day: string) => void
}

export default function DayListItem(props: Props) {
  const { summary, isSelected, onSelect } = props

  return (
    <li>
      <button
        type="button"
        className={isSelected ? `${styles.item} ${styles.selected}` : styles.item}
        aria-current={isSelected ? 'date' : undefined}
        onClick={() => onSelect(summary.day)}
      >
        <span>{summary.day}</span>
        <span className={styles.count}>{summary.promptCount}</span>
      </button>
    </li>
  )
}
