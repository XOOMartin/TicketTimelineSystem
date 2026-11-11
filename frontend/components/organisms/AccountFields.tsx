import FormField from 'components/molecules/FormField'
import SelectField from 'components/molecules/SelectField'
import { AccountForm, AccountValues } from 'hooks/useAccountForm'
import useOptions from 'hooks/useOptions'
import { ROLE_OPTIONS } from 'lib/accounts'

type Props = {
  form: AccountForm
  // The layout around the fields: a column, or pairs side by side
  className?: string
  // Whether it's the admin's own account: its role can't be changed
  isSelf?: boolean
  // The first field takes the focus
  autoFocus?: boolean
}

// The fields of an account, for every form that has them: logging in
// (username, password), signing up (names, username, password, team), and an
// admin adding or editing someone (the same, plus a role).
export default function AccountFields(props: Props) {
  const { form, className, isSelf = false, autoFocus = true } = props
  const { mode, values, set, errors, touch, shown } = form
  const { options } = useOptions()
  const isAdmin = mode === 'add' || mode === 'edit'
  const hasNames = mode !== 'login'

  // A text field: errors as you type, or (username, password) once you leave
  // it, since half-typed isn't wrong yet
  const text = (field: keyof AccountValues, label: string, extra: { checkOnLeave?: boolean } = {}) => ({
    label,
    value: values[field],
    onChange: (e: { target: { value: string } }) => {
      set(field, e.target.value)
      if (!extra.checkOnLeave) touch(field)
    },
    onBlur: () => touch(field),
    error: shown(field, errors[field]),
  })

  return (
    <div className={className}>
      {hasNames && (
        <>
          <FormField
            {...text('firstName', 'First name')}
            autoComplete={isAdmin ? 'off' : 'given-name'}
            data-autofocus={autoFocus || undefined}
            autoFocus={autoFocus && !isAdmin}
            required
          />
          <FormField {...text('lastName', 'Last name')} autoComplete={isAdmin ? 'off' : 'family-name'} required />
        </>
      )}
      <FormField
        {...text('username', 'Username', { checkOnLeave: true })}
        autoComplete={isAdmin ? 'off' : 'username'}
        autoCapitalize="none"
        spellCheck={false}
        autoFocus={autoFocus && !hasNames}
        required
      />
      <FormField
        {...text('password', mode === 'edit' ? 'New password (leave empty to keep it)' : 'Password')}
        type="password"
        autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
        required={mode !== 'edit'}
      />
      {hasNames && (
        <SelectField
          {...text('team', 'Team')}
          options={options?.projects ?? []}
          placeholder={isAdmin ? 'Choose a team' : 'Choose your team'}
          required
        />
      )}
      {isAdmin && (
        <SelectField
          label="Role"
          value={values.role}
          onChange={(e) => set('role', e.target.value)}
          options={ROLE_OPTIONS}
          // Admins can't demote themselves and lock everyone out
          disabled={isSelf}
          title={isSelf ? "You can't change your own role" : undefined}
          required
        />
      )}
    </div>
  )
}
