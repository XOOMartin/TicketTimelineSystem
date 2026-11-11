import styles from 'styles/ScoreBadge.module.css'

type Props = {
  score: number // 1-10
}

export default function ScoreBadge(props: Props) {
  const { score } = props

  const level = score >= 8 ? 'high' : score >= 5 ? 'mid' : 'low'

  return (
    <span className={`${styles.badge} ${styles[level]}`} title="Prompt quality score">
      {score}/10
    </span>
  )
}
