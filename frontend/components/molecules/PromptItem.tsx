import { useState } from 'react'

import Button from 'components/atoms/Button'
import { Prompt } from 'types/progress'

import styles from 'styles/PromptItem.module.css'

const PREVIEW_LENGTH = 400

type Props = {
  prompt: Prompt
}

export default function PromptItem(props: Props) {
  const { prompt } = props
  const [isExpanded, setIsExpanded] = useState(false)

  const isLong = prompt.text.length > PREVIEW_LENGTH
  const text = isLong && !isExpanded ? `${prompt.text.slice(0, PREVIEW_LENGTH)}…` : prompt.text
  const time = new Date(prompt.sentAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  const project = prompt.project.split('/').pop()

  return (
    <li className={styles.item}>
      <div className={styles.meta}>
        <span>{time}</span>
        <span>{project}</span>
        {prompt.branch && <span className={styles.branch}>{prompt.branch}</span>}
      </div>
      <p className={styles.text}>{text}</p>
      {isLong && (
        <Button onClick={() => setIsExpanded(!isExpanded)}>
          {isExpanded ? 'Show less' : 'Show more'}
        </Button>
      )}
    </li>
  )
}
