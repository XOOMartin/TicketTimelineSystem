import { useCallback, useState } from 'react'

import { errorMessage } from 'lib/errors'

// One thing a form or dialog does at a time (save, delete, record): whether
// it's running, and what went wrong. run() resolves to whether it worked, so
// the caller can close or move on; a failure shows as error instead. With
// stayBusy, a success leaves it busy: the dialog is closing or the page
// moving on, so its button mustn't come back for a second click.
export default function useAsyncAction() {
  const [isBusy, setIsBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback(async (action: () => Promise<unknown>, options: { stayBusy?: boolean } = {}) => {
    setIsBusy(true)
    setError(null)
    try {
      await action()
      if (!options.stayBusy) setIsBusy(false)
      return true
    } catch (e) {
      setError(errorMessage(e))
      setIsBusy(false)
      return false
    }
  }, [])

  return { run, isBusy, error, setError }
}
