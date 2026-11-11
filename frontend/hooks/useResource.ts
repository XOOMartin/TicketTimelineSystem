import { DependencyList, useCallback, useEffect, useRef, useState } from 'react'

import { errorMessage } from 'lib/errors'

// Something loaded from the server: the data, whether it's loading, what went
// wrong, and reload() for after a change. It loads again whenever deps
// change, and an answer that arrives after deps changed again (another week
// picked meanwhile) is ignored. load is null while there's nothing to load
// yet, e.g. no id until the router is ready.
export default function useResource<T>(load: (() => Promise<T>) | null, deps: DependencyList) {
  const [data, setData] = useState<T | null>(null)
  const [isLoading, setIsLoading] = useState(load !== null)
  const [error, setError] = useState<string | null>(null)
  // The latest load, so reload() always fetches for the current deps
  const loadRef = useRef(load)
  loadRef.current = load

  const fetchInto = useCallback(async (isCurrent: () => boolean) => {
    const run = loadRef.current
    if (!run) return
    setIsLoading(true)
    try {
      const result = await run()
      if (!isCurrent()) return
      setData(result)
      setError(null)
    } catch (e) {
      if (isCurrent()) setError(errorMessage(e))
    } finally {
      if (isCurrent()) setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    let isCurrent = true
    fetchInto(() => isCurrent)
    return () => {
      isCurrent = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  const reload = useCallback(() => fetchInto(() => true), [fetchInto])

  return { data, setData, isLoading, error, setError, reload }
}
