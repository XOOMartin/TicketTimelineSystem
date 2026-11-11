import * as api from 'lib/api'
import { errorMessage } from 'lib/errors'
import useResource from 'hooks/useResource'
import useWeek from 'hooks/useWeek'

// Everyone's week, for the admin's Members page, and changing a member's
// team or role.
export default function useMembers() {
  const [week, setWeek] = useWeek()
  const { data, isLoading, error, setError, reload } = useResource(() => api.fetchMembers(week), [week])

  const assign = async (id: string, team: string, role: string) => {
    try {
      await api.assignMember(id, team, role)
      setError(null)
    } catch (e) {
      setError(errorMessage(e))
    }
    // Reload either way, so a rejected change snaps back on screen
    await reload()
  }

  return { week, setWeek, data, isLoading, error, assign, reload }
}
