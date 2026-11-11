import Button from 'components/atoms/Button'
import Message from 'components/atoms/Message'
import ScoreBadge from 'components/atoms/ScoreBadge'
import ImprovementCard from 'components/molecules/ImprovementCard'
import { Review } from 'types/progress'

import styles from 'styles/ReviewPanel.module.css'

type Props = {
  review: Review | null
  isAnalyzing: boolean
  canAnalyze: boolean
  onAnalyze: () => void
}

export default function ReviewPanel(props: Props) {
  const { review, isAnalyzing, canAnalyze, onAnalyze } = props

  const analyzeButton = (
    <Button variant="primary" onClick={onAnalyze} disabled={!canAnalyze}>
      {isAnalyzing ? 'Analyzing…' : review ? 'Analyze again' : 'Analyze my prompts'}
    </Button>
  )

  if (!review) {
    return (
      <section className={styles.panel}>
        <h2>AI feedback</h2>
        <Message>
          {isAnalyzing
            ? 'Reading your prompts. This usually takes under a minute.'
            : 'This day has not been analyzed yet.'}
        </Message>
        {analyzeButton}
      </section>
    )
  }

  return (
    <section className={styles.panel}>
      <div className={styles.header}>
        <h2>AI feedback</h2>
        <ScoreBadge score={review.score} />
      </div>
      <p>{review.summary}</p>

      <div className={styles.focus}>
        <span className={styles.label}>Focus for next time</span>
        <p>{review.focus}</p>
      </div>

      <h3>What went well</h3>
      <ul>
        {/* AI text can repeat, so the index is the only safe key; the list never reorders */}
        {review.strengths.map((strength, i) => (
          <li key={i}>{strength}</li>
        ))}
      </ul>

      <h3>What to improve</h3>
      <ol className={styles.improvements}>
        {review.improvements.map((improvement, i) => (
          <ImprovementCard key={i} improvement={improvement} />
        ))}
      </ol>

      <div className={styles.footer}>
        {analyzeButton}
        <Message>Analyzed {new Date(review.createdAt).toLocaleString()}</Message>
      </div>
    </section>
  )
}
