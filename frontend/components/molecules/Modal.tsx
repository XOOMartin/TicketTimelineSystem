import { ReactNode, useEffect, useRef } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'

import styles from 'styles/Modal.module.css'

type Props = {
  title: string
  // normal fits a short form, large a form in two columns, wide a table
  size?: 'normal' | 'large' | 'wide'

  onClose: () => void
  children: ReactNode
}

// Opens when mounted. The native dialog traps focus and closes on Escape.
// Mark a field with data-autofocus to start there.
export default function Modal(props: Props) {
  const { title, size = 'normal', onClose, children } = props
  const ref = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = ref.current
    if (dialog && !dialog.open) dialog.showModal()
    // showModal focuses the first control (the close button) unless content
    // marks a better starting point
    dialog?.querySelector<HTMLElement>('[data-autofocus]')?.focus()
  }, [])

  return (
    <dialog
      ref={ref}
      className={size === 'normal' ? styles.dialog : `${styles.dialog} ${styles[size]}`}
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      // A press on the backdrop lands on the dialog itself, not its content
      onMouseDown={(e) => e.target === ref.current && onClose()}
    >
      <div className={styles.panel}>
        <header className={styles.header}>
          <h2>{title}</h2>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="Close">
            <Icon name="close" />
          </Button>
        </header>
        {children}
      </div>
    </dialog>
  )
}
