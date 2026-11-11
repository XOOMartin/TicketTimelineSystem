import { Role } from 'types/worklog'

// The rules for accounts, as the server checks them (internal/domain/user):
// a username is 3 to 64 letters, digits, dots, dashes, underscores or @
// (accounts from before usernames have their email as theirs); a password
// is at least 8 characters.
export const USERNAME_PATTERN = /^[A-Za-z0-9._@-]{3,64}$/
export const MIN_PASSWORD_LENGTH = 8

export const ROLE_OPTIONS: { value: Role; label: string }[] = [
  { value: 'member', label: 'Member' },
  { value: 'admin', label: 'Admin' },
]

export const roleLabel = (role: Role) => ROLE_OPTIONS.find((r) => r.value === role)?.label ?? role
