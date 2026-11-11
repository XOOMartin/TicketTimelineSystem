import { ReactNode, useId } from 'react'

import FieldLabel from 'components/molecules/FieldLabel'

import styles from 'styles/FormField.module.css'

type Props = {
  label: string
  value: ReactNode
  // Why it can't be changed, on hover
  title?: string
}

// A value a form shows but can't change, e.g. the project, fixed by the
// user's team: laid out like the fields beside it, as plain text.
export default function ReadOnlyField(props: Props) {
  const { label, value, title } = props
  const labelId = useId()

  return (
    <div className={styles.field}>
      <span id={labelId}>
        <FieldLabel label={label} errorId={`${labelId}-note`} />
      </span>
      <div className={styles.readOnly} aria-labelledby={labelId} title={title}>
        {value}
      </div>
    </div>
  )
}
