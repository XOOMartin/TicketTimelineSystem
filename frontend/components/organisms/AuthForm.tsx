import { FormEvent } from 'react'
import Link from 'next/link'

import Button from 'components/atoms/Button'
import Message from 'components/atoms/Message'
import AccountFields from 'components/organisms/AccountFields'
import useAccountForm from 'hooks/useAccountForm'
import useAsyncAction from 'hooks/useAsyncAction'
import { SignupInput } from 'types/worklog'

import styles from 'styles/Card.module.css'

type Props =
  | { mode: 'login'; onSubmit: (username: string, password: string) => Promise<void> }
  | { mode: 'signup'; onSubmit: (input: SignupInput) => Promise<void> }

// Logging in (username, password) or signing up (names, username, password,
// team), with the account fields every account form shares.
export default function AuthForm(props: Props) {
  const isSignup = props.mode === 'signup'
  const form = useAccountForm(props.mode)
  // Busy until the page moves on to the app
  const { run, isBusy: isSubmitting, error } = useAsyncAction()

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    // Say what's missing beside each field rather than sending it
    if (form.hasErrors) {
      form.showAll()
      return
    }
    const { firstName, lastName, username, password, team } = form.trimmed()
    await run(
      () =>
        props.mode === 'signup'
          ? props.onSubmit({ firstName, lastName, username, password, team })
          : props.onSubmit(username, password),
      { stayBusy: true },
    )
  }

  return (
    <form className={styles.authForm} onSubmit={handleSubmit} noValidate>
      {error && <Message variant="error">{error}</Message>}
      <AccountFields form={form} className={styles.stack} />
      <Button className={styles.submit} type="submit" variant="primary" disabled={isSubmitting}>
        {isSubmitting ? 'Please wait…' : isSignup ? 'Create account' : 'Log in'}
      </Button>
      <div className={styles.switch}>
        {isSignup ? (
          <span>
            Already have an account? <Link href="/login">Log in</Link>
          </span>
        ) : (
          <span>
            New here? <Link href="/signup">Create an account</Link>
          </span>
        )}
      </div>
    </form>
  )
}
