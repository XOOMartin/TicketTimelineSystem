import { useEffect } from 'react'
import { useRouter } from 'next/router'

import PageLayout from 'components/layouts/PageLayout'
import AuthForm from 'components/organisms/AuthForm'
import useAuth from 'hooks/useAuth'

export default function Signup() {
  const { user, signup } = useAuth()
  const { replace } = useRouter()

  useEffect(() => {
    if (user) replace(user.role === 'admin' ? '/admin' : '/')
  }, [user, replace])

  return (
    <PageLayout title="Sign up" heading="Create your account" subheading="Join your team and start logging your work" isPublic>
      <AuthForm mode="signup" onSubmit={signup} />
    </PageLayout>
  )
}
