import { ComponentProps, useId } from 'react'

import Select from 'components/atoms/Select'
import FieldLabel from 'components/molecules/FieldLabel'

import styles from 'styles/FormField.module.css'

type Props = ComponentProps<typeof Select> & {
  label: string
  // What's wrong with it, shown beside the label as it's filled in
  error?: string
}

export default function SelectField(props: Props) {
  const { label, error, ...selectProps } = props
  const errorId = useId()

  return (
    <label className={styles.field}>
      <FieldLabel label={label} required={selectProps.required} error={error} errorId={errorId} />
      <Select
        {...selectProps}
        aria-invalid={error ? true : selectProps['aria-invalid']}
        aria-describedby={error ? errorId : selectProps['aria-describedby']}
      />
    </label>
  )
}
