import { ReactNode } from 'react'

import styles from 'styles/Badge.module.css'

export type BadgeTone = 'gray' | 'blue' | 'amber' | 'red' | 'green' | 'sky' | 'teal' | 'ink' | 'accent'

// Maps status and submission values to a colour.
const TONES: Record<string, BadgeTone> = {
  not_started: 'gray',
  ongoing: 'blue',
  waiting: 'amber',
  blocked: 'red',
  done: 'green',
  app: 'green',
  imported: 'sky',
  missing: 'red',
}

type Props = {
  // A status or submission, whose colour it takes
  value?: string
  // Or the colour itself, e.g. a team's or a role's
  tone?: BadgeTone
  children: ReactNode
}

export default function Badge(props: Props) {
  const { value = '', tone, children } = props

  return <span className={`${styles.badge} ${styles[tone ?? TONES[value] ?? 'gray']}`}>{children}</span>
}
