import { InputHTMLAttributes } from 'react'

import styles from 'styles/TextInput.module.css'

type Props = InputHTMLAttributes<HTMLInputElement>

export default function TextInput(props: Props) {
  const { className, ...rest } = props

  return <input className={[styles.input, className].filter(Boolean).join(' ')} {...rest} />
}
