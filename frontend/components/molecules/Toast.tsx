import { useEffect } from 'react'

import Icon from 'components/atoms/Icon'

import styles from 'styles/Toast.module.css'

const SHOWN_MS = 4000

type Props = {
  message: string | null
  onDone: () => void // should be stable, e.g. from useCallback
  tone?: 'success' | 'error'
}

// A short confirmation in the corner of the screen that goes away by itself.
export default function Toast(props: Props) {
  const { message, onDone, tone = 'success' } = props

  useEffect(() => {
    if (!message) return
    const timer = setTimeout(onDone, SHOWN_MS)
    return () => clearTimeout(timer)
  }, [message, onDone])

  // The live region stays mounted so screen readers announce new messages
  return (
    <div className={styles.region} role="status" aria-live="polite">
      {message && (
        <div className={tone === 'error' ? `${styles.toast} ${styles.error}` : styles.toast}>
          <Icon name={tone === 'error' ? 'alert' : 'check'} size={16} />
          <span>{message}</span>
          <button type="button" className={styles.close} onClick={onDone} aria-label="Dismiss">
            <Icon name="close" size={14} />
          </button>
        </div>
      )}
    </div>
  )
}
