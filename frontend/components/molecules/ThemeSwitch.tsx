import Icon, { IconName } from 'components/atoms/Icon'
import useTheme from 'hooks/useTheme'
import { ThemePreference } from 'lib/theme'

import styles from 'styles/ThemeSwitch.module.css'

const CHOICES: { value: ThemePreference; label: string; icon: IconName }[] = [
  { value: 'system', label: 'System', icon: 'monitor' },
  { value: 'light', label: 'Light', icon: 'sun' },
  { value: 'dark', label: 'Dark', icon: 'moon' },
]

type Props = {
  // Icons only, stacked, for the collapsed sidebar
  compact?: boolean
}

// Light, dark, or follow the computer's setting.
export default function ThemeSwitch(props: Props) {
  const { compact = false } = props
  const { preference, choose } = useTheme()

  return (
    <div className={compact ? `${styles.switch} ${styles.compact}` : styles.switch} role="group" aria-label="Theme">
      {CHOICES.map((c) => (
        <button
          key={c.value}
          type="button"
          className={c.value === preference ? `${styles.option} ${styles.active}` : styles.option}
          aria-pressed={c.value === preference}
          aria-label={compact ? c.label : undefined}
          title={c.value === 'system' ? 'Match your computer' : `${c.label} mode`}
          onClick={() => choose(c.value)}
        >
          <Icon name={c.icon} size={14} />
          {!compact && c.label}
        </button>
      ))}
    </div>
  )
}
