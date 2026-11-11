import { useRouter } from 'next/router'

import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import WeekPicker from 'components/molecules/WeekPicker'
import WeekOverview from 'components/organisms/WeekOverview'
import useMemberWeek from 'hooks/useMemberWeek'

// One member's dashboard, exactly as they see it, for admins.
export default function MemberDashboard() {
  const { query } = useRouter()
  const id = typeof query.id === 'string' ? query.id : undefined
  const { week, setWeek, data, isLoading, error } = useMemberWeek(id)

  const name = data?.user.name ?? 'Member'

  return (
    <PageLayout
      title={`${name}'s dashboard`}
      heading={`${name}'s dashboard`}
      subheading={data ? `${data.user.team} · their week at a glance` : undefined}
      requireAdmin
      actions={
        <>
          <WeekPicker week={week} onChange={setWeek} />
          <LinkButton href="/admin/members">
            <Icon name="users" size={16} />
            All members
          </LinkButton>
        </>
      }
    >
      {error && <Message variant="error">{error}</Message>}
      {isLoading && !data ? (
        <Message>Loading…</Message>
      ) : (
        data && <WeekOverview name={name} data={data} isLoading={isLoading} />
      )}
    </PageLayout>
  )
}
