import { TextareaHTMLAttributes, useId } from 'react'

import TextArea from 'components/atoms/TextArea'
import FieldLabel from 'components/molecules/FieldLabel'

import styles from 'styles/FormField.module.css'

type Props = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  label: string
  // What's wrong with it, shown beside the label as it's filled in
  error?: string
}

export default function TextAreaField(props: Props) {
  const { label, error, ...textAreaProps } = props
  const errorId = useId()

  return (
    <label className={styles.field}>
      <FieldLabel label={label} required={textAreaProps.required} error={error} errorId={errorId} />
      <TextArea
        {...textAreaProps}
        aria-invalid={error ? true : textAreaProps['aria-invalid']}
        aria-describedby={error ? errorId : textAreaProps['aria-describedby']}
      />
    </label>
  )
}
