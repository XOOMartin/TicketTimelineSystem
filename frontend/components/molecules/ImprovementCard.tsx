import { Improvement } from 'types/progress'

import styles from 'styles/ImprovementCard.module.css'

type Props = {
  improvement: Improvement
}

export default function ImprovementCard(props: Props) {
  const { improvement } = props

  return (
    <li className={styles.card}>
      <p className={styles.issue}>{improvement.issue}</p>
      <div className={styles.compare}>
        <div className={styles.before}>
          <span className={styles.label}>You wrote</span>
          <p>{improvement.example}</p>
        </div>
        <div className={styles.after}>
          <span className={styles.label}>Try instead</span>
          <p>{improvement.better}</p>
        </div>
      </div>
    </li>
  )
}
