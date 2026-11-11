import { useEffect, useState } from 'react'

import { ThemePreference, readPreference, savePreference } from 'lib/theme'

export default function useTheme() {
  // Storage is only readable in the browser, so start at "system" and catch
  // up after mounting
  const [preference, setPreference] = useState<ThemePreference>('system')

  useEffect(() => {
    setPreference(readPreference())
  }, [])

  const choose = (next: ThemePreference) => {
    savePreference(next)
    setPreference(next)
  }

  return { preference, choose }
}
