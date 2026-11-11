import { ButtonHTMLAttributes } from 'react'

import styles from 'styles/Button.module.css'

export type ButtonVariant = 'default' | 'primary' | 'danger' | 'destructive' | 'ghost' | 'outline'
export type ButtonSize = 'md' | 'sm' | 'icon'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  size?: ButtonSize
}

export function buttonClass(variant: ButtonVariant = 'default', size: ButtonSize = 'md', className?: string) {
  return [styles.button, styles[variant], styles[size], className].filter(Boolean).join(' ')
}

export default function Button(props: Props) {
  const { variant = 'default', size = 'md', type = 'button', className, ...rest } = props

  return <button type={type} className={buttonClass(variant, size, className)} {...rest} />
}
