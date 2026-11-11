import * as api from 'lib/api'
import useMutation from 'hooks/useMutation'
import useResource from 'hooks/useResource'
import { AdminUser, UserInput } from 'types/worklog'

// The admin's user management: the accounts, active and deleted, and what
// can be done to them. Each change reloads the lists and says what happened;
// a failed one keeps its error and is re-thrown so its dialog stays open.
export default function useUsers() {
  const { data, isLoading, error, setError, reload } = useResource(api.fetchUsers, [])
  const { mutate, notice, clearNotice } = useMutation(reload, setError)

  const nameOf = (u: AdminUser) => u.name || u.username
  const create = (input: UserInput) =>
    mutate(() => api.createUser(input), `Added ${input.firstName} ${input.lastName} (${input.username})`)
  const update = (u: AdminUser, input: UserInput) =>
    mutate(() => api.updateUser(u.id, input), `Saved ${input.firstName} ${input.lastName}`)
  const remove = (u: AdminUser) =>
    mutate(() => api.deleteUser(u.id), `Deleted ${nameOf(u)}: they can't log in. Restore them from Deleted accounts.`)
  const restore = (u: AdminUser) => mutate(() => api.restoreUser(u.id), `Restored ${nameOf(u)}: they can log in again`)
  const removePermanently = (u: AdminUser) =>
    mutate(() => api.deleteUserPermanently(u.id), `Permanently deleted ${nameOf(u)} and all their data`)

  return {
    data,
    isLoading,
    error,
    notice,
    clearNotice,
    create,
    update,
    remove,
    restore,
    removePermanently,
  }
}
