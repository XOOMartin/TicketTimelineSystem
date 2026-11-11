import { ReactNode } from 'react'

import styles from 'styles/Message.module.css'

type Props = {
  variant?: 'muted' | 'error' | 'success'
  children: ReactNode
}

export default function Message(props: Props) {
  const { variant = 'muted', children } = props

  return <p className={styles[variant]}>{children}</p>
}
