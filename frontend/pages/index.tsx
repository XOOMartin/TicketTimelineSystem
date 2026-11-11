import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import Toast from 'components/molecules/Toast'
import WeekPicker from 'components/molecules/WeekPicker'
import WeekOverview from 'components/organisms/WeekOverview'
import useAuth from 'hooks/useAuth'
import useMyWeek from 'hooks/useMyWeek'

// The member's own week: what they logged and what needs attention.
export default function Dashboard() {
  const { user } = useAuth()
  const { week, setWeek, data, isLoading, error, notice, clearNotice } = useMyWeek()

  return (
    <PageLayout
      title="My dashboard"
      heading={user ? `Welcome back, ${user.name.split(' ')[0]}` : 'My dashboard'}
      subheading="Your week at a glance"
      actions={
        <>
          <WeekPicker week={week} onChange={setWeek} />
          {/* Opens My timeline's own Input timeline, in the same week */}
          <LinkButton href={`/timeline?week=${week}&input=1`} variant="primary">
            <Icon name="plus" size={16} />
            Input timeline
          </LinkButton>
        </>
      }
    >
      {user && (
        <>
          {error && <Message variant="error">{error}</Message>}

          <WeekOverview name={user.name} data={data} isLoading={isLoading} isOwn />
        </>
      )}
      <Toast message={notice} onDone={clearNotice} />
    </PageLayout>
  )
}
