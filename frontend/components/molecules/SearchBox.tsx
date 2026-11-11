import Icon from 'components/atoms/Icon'
import TextInput from 'components/atoms/TextInput'

import styles from 'styles/SearchBox.module.css'

type Props = {
  value: string
  onChange: (value: string) => void
  placeholder: string
  // What it searches, for screen readers, e.g. "Search tickets"
  label: string
  // Its width where it sits
  className?: string
}

// A search field with a magnifier, for filtering a list as you type.
export default function SearchBox(props: Props) {
  const { value, onChange, placeholder, label, className } = props

  return (
    <label className={className ? `${styles.search} ${className}` : styles.search}>
      <Icon name="search" size={16} />
      <TextInput
        type="search"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-label={label}
      />
    </label>
  )
}
