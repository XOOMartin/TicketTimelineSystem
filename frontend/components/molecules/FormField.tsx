import { InputHTMLAttributes, useId } from 'react'

import TextInput from 'components/atoms/TextInput'
import FieldLabel from 'components/molecules/FieldLabel'

import styles from 'styles/FormField.module.css'

type Props = InputHTMLAttributes<HTMLInputElement> & {
  label: string
  // What's wrong with it, shown beside the label as it's filled in
  error?: string
  // Worth knowing but not wrong, in the same place; doesn't mark it invalid
  warning?: string
}

export default function FormField(props: Props) {
  const { label, error, warning, ...inputProps } = props
  const errorId = useId()

  return (
    <label className={styles.field}>
      <FieldLabel label={label} required={inputProps.required} error={error} warning={warning} errorId={errorId} />
      <TextInput
        {...inputProps}
        aria-invalid={error ? true : inputProps['aria-invalid']}
        aria-describedby={error || warning ? errorId : inputProps['aria-describedby']}
      />
    </label>
  )
}
