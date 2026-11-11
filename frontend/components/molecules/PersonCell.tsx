import { ReactNode } from 'react'
import Link from 'next/link'

import Avatar from 'components/atoms/Avatar'

import styles from 'styles/PersonCell.module.css'

type Props = {
  name: string
  // Under the name, e.g. "@gerald"
  detail?: ReactNode
  // What picks the avatar's colour, so two people with one name differ
  seed?: string
  // Where the name links to, e.g. their dashboard
  href?: string
  hrefTitle?: string
  // Beside the name, e.g. a "You" tag
  tag?: ReactNode
  avatarSize?: number
  // Greyed, e.g. a deleted account
  muted?: boolean
}

// A person in a table: avatar, name (a link when there's somewhere to go)
// and a detail line.
export default function PersonCell(props: Props) {
  const { name, detail, seed, href, hrefTitle, tag, avatarSize, muted = false } = props

  return (
    <div className={muted ? `${styles.person} ${styles.muted}` : styles.person}>
      <Avatar name={name} seed={seed} size={avatarSize} />
      <div className={styles.text}>
        <div className={styles.name}>
          {href ? (
            <Link href={href} title={hrefTitle}>
              {name}
            </Link>
          ) : (
            name
          )}
          {tag}
        </div>
        {detail && <div className={styles.detail}>{detail}</div>}
      </div>
    </div>
  )
}
