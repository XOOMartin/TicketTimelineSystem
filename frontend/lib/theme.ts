// Light or dark mode. The choice is kept per browser; "system" follows the
// operating system's setting.

export type ThemePreference = 'system' | 'light' | 'dark'

const THEME_KEY = 'theme'

const SYSTEM_LIGHT = '(prefers-color-scheme: light)'

// Runs inline in <head> (see pages/_document.tsx) before the page paints, so
// there is no flash of the wrong theme, and again when the system setting
// changes.
export const THEME_SCRIPT = `(function () {
  var media = window.matchMedia('${SYSTEM_LIGHT}')
  function apply() {
    var saved
    try { saved = localStorage.getItem('${THEME_KEY}') } catch (e) {}
    var theme = saved === 'light' || saved === 'dark' ? saved : media.matches ? 'light' : 'dark'
    document.documentElement.setAttribute('data-theme', theme)
  }
  apply()
  media.addEventListener('change', apply)
})()`

export function readPreference(): ThemePreference {
  try {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'light' || saved === 'dark') return saved
  } catch {
    // Storage can be blocked, e.g. in a private window
  }
  return 'system'
}

export function savePreference(preference: ThemePreference) {
  try {
    if (preference === 'system') localStorage.removeItem(THEME_KEY)
    else localStorage.setItem(THEME_KEY, preference)
  } catch {
    // Still switch for this page view, just without remembering it
  }
  const isLight = preference === 'system' ? window.matchMedia(SYSTEM_LIGHT).matches : preference === 'light'
  document.documentElement.setAttribute('data-theme', isLight ? 'light' : 'dark')
}
