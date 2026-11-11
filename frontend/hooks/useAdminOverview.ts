import * as api from 'lib/api'
import useResource from 'hooks/useResource'
import useWeek from 'hooks/useWeek'

// Everything the admin dashboard shows for a week: who submitted, the live
// ticket summary, and the latest saved reports.
export default function useAdminOverview() {
  const [week, setWeek] = useWeek()
  const { data, isLoading, error } = useResource(async () => {
    const [members, overview, reports] = await Promise.all([
      api.fetchMembers(week),
      api.fetchOverview(week),
      api.fetchReports(),
    ])
    return { members, overview, reports }
  }, [week])

  return {
    week,
    setWeek,
    members: data?.members ?? null,
    overview: data?.overview ?? null,
    reports: data?.reports ?? [],
    isLoading,
    error,
  }
}
