import { ReactNode } from 'react'
import Link from 'next/link'

import { ButtonSize, ButtonVariant, buttonClass } from 'components/atoms/Button'

type Props = {
  href: string
  variant?: ButtonVariant
  size?: ButtonSize
  // Set for files the server sends, so the browser saves them
  download?: boolean
  children: ReactNode
}

// A link that looks like a Button.
export default function LinkButton(props: Props) {
  const { href, variant = 'default', size = 'md', download = false, children } = props
  const className = buttonClass(variant, size)

  if (download) {
    return (
      <a className={className} href={href} download>
        {children}
      </a>
    )
  }
  return (
    <Link className={className} href={href}>
      {children}
    </Link>
  )
}
