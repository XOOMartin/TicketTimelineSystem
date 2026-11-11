import { ReactNode } from 'react'

import Badge, { BadgeTone } from 'components/atoms/Badge'

import styles from 'styles/SectionHeader.module.css'

type Props = {
  title: ReactNode
  // How many things the section has, beside its title
  count?: number
  countTone?: BadgeTone
  // For links to the section, e.g. "tickets" for #tickets
  id?: string
  // Buttons or links on the right
  actions?: ReactNode
}

// The heading over a section of a page: its title, how many it has, and
// what can be done there.
export default function SectionHeader(props: Props) {
  const { title, count, countTone = 'blue', id, actions } = props

  return (
    <div className={styles.header}>
      <h2 id={id} className={styles.title}>
        {title}
        {count !== undefined && <Badge tone={countTone}>{count}</Badge>}
      </h2>
      {actions}
    </div>
  )
}
