import { SelectHTMLAttributes } from 'react'

import { Option } from 'types/worklog'

import styles from 'styles/TextInput.module.css'

type Props = SelectHTMLAttributes<HTMLSelectElement> & {
  options: (Option | string)[]
  placeholder?: string
}

export default function Select(props: Props) {
  const { options, placeholder, className, ...rest } = props

  return (
    <select className={[styles.input, className].filter(Boolean).join(' ')} {...rest}>
      {placeholder && (
        <option value="" disabled>
          {placeholder}
        </option>
      )}
      {options.map((o) => {
        const { value, label } = typeof o === 'string' ? { value: o, label: o } : o
        return (
          <option key={value} value={value}>
            {label}
          </option>
        )
      })}
    </select>
  )
}
