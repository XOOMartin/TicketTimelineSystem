import { useState } from 'react'

import useTouched from 'hooks/useTouched'
import { MIN_PASSWORD_LENGTH, USERNAME_PATTERN } from 'lib/accounts'
import { Role } from 'types/worklog'

// login: username and password. signup: names, username, password, team.
// add and edit: the same, plus a role, for an admin; editing keeps the
// password when it's left empty.
export type AccountFormMode = 'login' | 'signup' | 'add' | 'edit'

export type AccountValues = {
  firstName: string
  lastName: string
  username: string
  password: string
  team: string
  role: Role
}

const EMPTY: AccountValues = { firstName: '', lastName: '', username: '', password: '', team: '', role: 'member' }

// One account form's fields and their errors, for logging in, signing up
// and an admin adding or editing someone. Errors show beside each field as
// it's filled in (or all of them after a submit with problems).
export default function useAccountForm(mode: AccountFormMode, initial: Partial<AccountValues> = {}) {
  // Only the account's own fields (an admin editing passes the whole user);
  // a password always starts empty
  const [values, setValues] = useState<AccountValues>(() => ({
    firstName: initial.firstName ?? EMPTY.firstName,
    lastName: initial.lastName ?? EMPTY.lastName,
    username: initial.username ?? EMPTY.username,
    password: '',
    team: initial.team ?? EMPTY.team,
    role: initial.role ?? EMPTY.role,
  }))
  const { touch, shown, showAll } = useTouched()
  const set = (field: keyof AccountValues, value: string) => setValues((v) => ({ ...v, [field]: value }))

  // "Enter your username" to the person themselves, "Enter a username" to an admin
  const whose = mode === 'login' || mode === 'signup' ? 'your' : 'a'
  const hasNames = mode !== 'login'
  const { firstName, lastName, username, password, team } = values
  const errors: Record<keyof AccountValues, string> = {
    firstName: hasNames && firstName.trim() === '' ? `Enter ${whose} first name` : '',
    lastName: hasNames && lastName.trim() === '' ? `Enter ${whose} last name` : '',
    username:
      username.trim() === ''
        ? `Enter ${whose} username`
        : mode !== 'login' && !USERNAME_PATTERN.test(username.trim())
          ? '3 or more letters, digits, dots, dashes or underscores'
          : '',
    password:
      password === ''
        ? mode === 'edit'
          ? ''
          : `Enter ${whose} password`
        : mode !== 'login' && password.length < MIN_PASSWORD_LENGTH
          ? `At least ${MIN_PASSWORD_LENGTH} characters`
          : '',
    team: hasNames && team === '' ? `Choose ${whose} team` : '',
    role: '',
  }
  const hasErrors = Object.values(errors).some(Boolean)

  // The values to send, trimmed (a password is sent as typed)
  const trimmed = (): AccountValues => ({
    ...values,
    firstName: firstName.trim(),
    lastName: lastName.trim(),
    username: username.trim(),
  })

  return { mode, values, set, errors, hasErrors, touch, shown, showAll, trimmed }
}

export type AccountForm = ReturnType<typeof useAccountForm>
