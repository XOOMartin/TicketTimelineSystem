import { useState } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import Toast from 'components/molecules/Toast'
import WeekPicker from 'components/molecules/WeekPicker'
import CheckInDialog from 'components/organisms/CheckInDialog'
import WeekOverview from 'components/organisms/WeekOverview'
import useAuth from 'hooks/useAuth'
import useMyWeek from 'hooks/useMyWeek'

// The member's own week: what they logged and what needs attention.
export default function Dashboard() {
  const { user } = useAuth()
  const { week, setWeek, data, isLoading, error, create, notice, clearNotice } = useMyWeek()
  const [isLogging, setIsLogging] = useState(false)

  return (
    <PageLayout
      title="My dashboard"
      heading={user ? `Welcome back, ${user.name.split(' ')[0]}` : 'My dashboard'}
      subheading="Your week at a glance"
      actions={
        <>
          <WeekPicker week={week} onChange={setWeek} />
          <Button variant="primary" onClick={() => setIsLogging(true)}>
            <Icon name="plus" size={16} />
            Log check-in
          </Button>
        </>
      }
    >
      {user && (
        <>
          {error && !isLogging && <Message variant="error">{error}</Message>}

          <WeekOverview name={user.name} data={data} isLoading={isLoading} isOwn />

          {isLogging && (
            <CheckInDialog
              checkIn={null}
              defaultProject={user.team}
              knownTickets={data?.tickets ?? []}
              onLog={create}
              onClose={() => setIsLogging(false)}
            />
          )}
        </>
      )}
      <Toast message={notice} onDone={clearNotice} />
    </PageLayout>
  )
}
