import { TextareaHTMLAttributes } from 'react'

import styles from 'styles/TextInput.module.css'

type Props = TextareaHTMLAttributes<HTMLTextAreaElement>

export default function TextArea(props: Props) {
  const { className, ...rest } = props

  return <textarea className={[styles.input, className].filter(Boolean).join(' ')} {...rest} />
}
