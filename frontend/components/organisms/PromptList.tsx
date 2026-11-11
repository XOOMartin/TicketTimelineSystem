import Message from 'components/atoms/Message'
import PromptItem from 'components/molecules/PromptItem'
import { Prompt } from 'types/progress'

import styles from 'styles/PromptList.module.css'

type Props = {
  prompts: Prompt[]
}

export default function PromptList(props: Props) {
  const { prompts } = props

  if (prompts.length === 0) {
    return <Message>No prompts on this day.</Message>
  }

  return (
    <ol className={styles.list}>
      {prompts.map((prompt) => (
        <PromptItem key={prompt.id} prompt={prompt} />
      ))}
    </ol>
  )
}
