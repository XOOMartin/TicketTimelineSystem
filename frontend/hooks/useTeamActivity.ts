import { useEffect, useState } from 'react'

import useResource from 'hooks/useResource'
import * as api from 'lib/api'
import { MemberActivity } from 'types/worklog'

// How often the team is asked for again, while the page is showing
const REFRESH_MS = 30_000

// Who on the team is busy (a ticket ongoing) or available. It's fetched again every 30 seconds
// (and as soon as the tab is looked at again), as JSON in the background,
// and the running timers count up in between without asking the server.
export default function useTeamActivity() {
  const activity = useResource(api.fetchTeamActivity, [])
  const { data, reload } = activity
  const [fetchedAt, setFetchedAt] = useState(() => Date.now())
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (data) setFetchedAt(Date.now())
  }, [data])

  useEffect(() => {
    const isShowing = () => document.visibilityState === 'visible'
    const poll = window.setInterval(() => isShowing() && reload(), REFRESH_MS)
    const tick = window.setInterval(() => setNow(Date.now()), 10_000)
    const onVisible = () => {
      if (isShowing()) {
        setNow(Date.now())
        reload()
      }
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      window.clearInterval(poll)
      window.clearInterval(tick)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [reload])

  // Seconds on the current ticket now: a running timer keeps counting
  const elapsedOf = (m: MemberActivity) =>
    m.timer === 'running' ? m.elapsedSeconds + Math.max(0, (now - fetchedAt) / 1000) : m.elapsedSeconds

  return {
    members: data?.members ?? null,
    // Only the first load; refreshes keep the table showing
    isLoading: activity.isLoading && !data,
    error: activity.error,
    now,
    elapsedOf,
  }
}
