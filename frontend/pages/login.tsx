import { useEffect } from 'react'
import { useRouter } from 'next/router'

import PageLayout from 'components/layouts/PageLayout'
import AuthForm from 'components/organisms/AuthForm'
import useAuth from 'hooks/useAuth'

export default function Login() {
  const { user, login } = useAuth()
  const { replace } = useRouter()

  useEffect(() => {
    if (user) replace(user.role === 'admin' ? '/admin' : '/')
  }, [user, replace])

  return (
    <PageLayout title="Log in" heading="Log in" subheading="Log in to track your tickets and check-ins" isPublic>
      <AuthForm mode="login" onSubmit={login} />
    </PageLayout>
  )
}
