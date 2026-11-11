import { useCallback, useState } from 'react'

import { errorMessage } from 'lib/errors'

// Changes saved to the server: each one says what it did (a notice, for a
// toast), then reloads so the page shows the result. A failed one sets the
// page's error and is re-thrown, so the dialog it came from stays open with
// its input.
export default function useMutation(reload: () => Promise<void>, setError: (error: string | null) => void) {
  const [notice, setNotice] = useState<string | null>(null)
  // Stable, as the toast showing the notice needs
  const clearNotice = useCallback(() => setNotice(null), [])

  const mutate = async (action: () => Promise<unknown>, success: string) => {
    try {
      await action()
      setError(null)
    } catch (e) {
      setError(errorMessage(e))
      throw e
    }
    setNotice(success)
    await reload()
  }

  return { mutate, notice, clearNotice }
}
