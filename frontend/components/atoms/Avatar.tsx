import { seriesColor } from 'lib/colors'

import styles from 'styles/Avatar.module.css'

type Props = {
  name: string
  size?: number
  // What picks the colour, when two people can share a name (a username)
  seed?: string
}

function initials(name: string) {
  const parts = name.trim().split(/\s+/)
  return ((parts[0]?.[0] ?? '') + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase() || '?'
}

// The same name always gets the same colour.
function colorOf(name: string) {
  let hash = 0
  for (const ch of name) hash = (hash * 31 + ch.charCodeAt(0)) | 0
  return seriesColor(Math.abs(hash))
}

export default function Avatar(props: Props) {
  const { name, size = 32, seed } = props

  return (
    <span
      className={styles.avatar}
      style={{ width: size, height: size, fontSize: size * 0.38, background: colorOf(seed ?? name) }}
      aria-hidden="true"
    >
      {initials(name)}
    </span>
  )
}
