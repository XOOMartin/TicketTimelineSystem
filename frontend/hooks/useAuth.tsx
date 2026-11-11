import { ReactNode, createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'

import * as api from 'lib/api'
import { SignupInput, User } from 'types/worklog'

type AuthState = {
  user: User | null
  // True until the first /auth/me answer, so pages don't redirect early
  isLoading: boolean
  login: (username: string, password: string) => Promise<void>
  signup: (input: SignupInput) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider(props: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    api
      .fetchMe()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setIsLoading(false))
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    setUser(await api.login(username, password))
  }, [])

  const signup = useCallback(async (input: SignupInput) => {
    setUser(await api.signup(input))
  }, [])

  const logout = useCallback(async () => {
    await api.logout()
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({ user, isLoading, login, signup, logout }),
    [user, isLoading, login, signup, logout],
  )
  return <AuthContext.Provider value={value}>{props.children}</AuthContext.Provider>
}

export default function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
