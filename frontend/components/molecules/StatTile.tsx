import Icon, { IconName } from 'components/atoms/Icon'

import styles from 'styles/StatTile.module.css'

type Props = {
  label: string
  value: string
  icon: IconName
  color: string // any CSS colour, e.g. var(--green), so it follows the theme
  hint?: string
  warn?: boolean
  // 0–1, drawn as a bar under the value
  progress?: number
}

export default function StatTile(props: Props) {
  const { label, value, icon, color, hint, warn = false, progress } = props

  return (
    <div className={warn ? `${styles.tile} ${styles.warn}` : styles.tile}>
      <span className={styles.icon} style={{ color, background: `color-mix(in srgb, ${color} 14%, transparent)` }}>
        <Icon name={icon} size={20} />
      </span>
      <div className={styles.text}>
        <span className={styles.label}>{label}</span>
        <span className={styles.value}>{value}</span>
        {progress !== undefined && (
          <span className={styles.track}>
            <span style={{ width: `${Math.min(progress, 1) * 100}%`, background: color }} />
          </span>
        )}
        {hint && <span className={styles.hint}>{hint}</span>}
      </div>
    </div>
  )
}
