import { useCallback, useState } from 'react'

// Which fields of a form have been filled in or left, so their errors show
// as you go (in real time) without a fresh form opening full of red. After a
// submit with problems, every field's error shows.
export default function useTouched() {
  const [touched, setTouched] = useState<Record<string, boolean>>({})
  const [isAll, setIsAll] = useState(false)

  const touch = useCallback((name: string) => setTouched((t) => (t[name] ? t : { ...t, [name]: true })), [])
  // The error to show for a field: its own, once it's been touched
  const shown = useCallback((name: string, error: string) => (isAll || touched[name] ? error : ''), [isAll, touched])
  const showAll = useCallback(() => setIsAll(true), [])

  return { touch, shown, showAll }
}
