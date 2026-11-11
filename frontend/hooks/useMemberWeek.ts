import * as api from 'lib/api'
import useResource from 'hooks/useResource'
import useWeek from 'hooks/useWeek'

// One member's dashboard, for admins. id is undefined until the router is
// ready.
export default function useMemberWeek(id: string | undefined) {
  const [week, setWeek] = useWeek()
  const { data, isLoading, error } = useResource(id ? () => api.fetchMemberWeek(id, week) : null, [id, week])

  return { week, setWeek, data, isLoading, error }
}
