import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import ListCard, { ListCardItem } from 'components/molecules/ListCard'
import StatTile from 'components/molecules/StatTile'
import WeekPicker from 'components/molecules/WeekPicker'
import SectionHeader from 'components/molecules/SectionHeader'
import TeamBreakdown from 'components/organisms/TeamBreakdown'
import TicketTable from 'components/organisms/TicketTable'
import useAdminOverview from 'hooks/useAdminOverview'
import useOptions from 'hooks/useOptions'
import { STATUS_COLORS, SUBMISSION_COLORS } from 'lib/colors'
import { formatDateTime, formatDay, formatHours } from 'lib/dates'
import { REPORT_KIND_LABELS, reportDays } from 'lib/reports'

import styles from 'styles/Dashboard.module.css'

// The whole team's week: who is behind, what is stuck, and where the
// hours went.
export default function AdminDashboard() {
  const { statusLabel } = useOptions()
  const { week, setWeek, members, overview, reports, isLoading, error } = useAdminOverview()

  const people = members?.members ?? []
  const tickets = overview?.tickets ?? []
  const submitted = people.filter((m) => m.submission !== 'missing').length
  const hours = people.reduce((sum, m) => sum + m.activity.hours, 0)
  const expected = people.reduce((sum, m) => sum + m.activity.expectedHours, 0)
  const stuck = tickets.filter((t) => t.status === 'blocked' || t.status === 'waiting')

  const behind: ListCardItem[] = people
    .filter((m) => m.submission === 'missing' || m.activity.missedDays.length > 0)
    .sort((a, b) => b.activity.missedDays.length - a.activity.missedDays.length)
    .map((m) => ({
      key: m.user.id,
      color: SUBMISSION_COLORS[m.submission],
      primary: m.user.name,
      secondary:
        m.submission === 'missing'
          ? `${m.user.team} · nothing logged yet`
          : `${m.user.team} · no check-ins ${m.activity.missedDays.map(formatDay).join(', ')}`,
      aside: `${formatHours(m.activity.hours)}/${formatHours(m.activity.expectedHours)} h`,
    }))

  const stuckItems: ListCardItem[] = stuck.map((t) => ({
    key: t.ticketKey,
    color: STATUS_COLORS[t.status],
    primary: `${t.ticketKey} · ${t.title}`,
    secondary: `${statusLabel(t.status)} · ${t.assignees.join(', ')}${t.note ? ` — ${t.note}` : ''}`,
  }))

  const reportItems: ListCardItem[] = reports.map((r) => ({
    key: r.id,
    color: r.kind === 'weekly' ? 'var(--accent)' : 'var(--blue)',
    primary: `${REPORT_KIND_LABELS[r.kind]} · ${reportDays(r, true)}`,
    secondary: `${r.generatedBy} · ${formatDateTime(r.generatedAt)}`,
    aside: `${r.ticketCount} tickets`,
  }))

  return (
    <PageLayout
      title="Admin dashboard"
      heading="Admin dashboard"
      subheading="How every team is doing this week"
      requireAdmin
      actions={
        <>
          <WeekPicker week={week} onChange={setWeek} />
          <LinkButton href="/admin/reports" variant="primary">
            <Icon name="report" size={16} />
            Reports
          </LinkButton>
        </>
      }
    >
      {error && <Message variant="error">{error}</Message>}
      {isLoading && !members ? (
        <Message>Loading…</Message>
      ) : (
        <>
          <div className={styles.stats}>
            <StatTile icon="users" color="var(--accent)" label="Members" value={String(people.length)} />
            <StatTile
              icon="check"
              color="var(--green)"
              label="Submitted this week"
              value={`${submitted} / ${people.length}`}
              hint={`${people.length - submitted} still missing`}
              progress={people.length > 0 ? submitted / people.length : undefined}
              warn={submitted < people.length}
            />
            <StatTile
              icon="clock"
              color="var(--blue)"
              label="Hours logged"
              value={`${formatHours(hours)} h`}
              hint={`of ${formatHours(expected)} h expected so far`}
              progress={expected > 0 ? hours / expected : undefined}
            />
            <StatTile
              icon="alert"
              color="var(--pink)"
              label="Blocked or waiting"
              value={String(stuck.length)}
              hint={`${tickets.length} tickets worked on`}
              warn={stuck.length > 0}
            />
          </div>

          <div className={styles.lists}>
            <ListCard
              title="Behind on check-ins"
              items={behind}
              emptyText="Everyone is up to date."
              action={{ label: 'Manage members', href: '/admin/members' }}
            />
            <ListCard
              title="Blocked & waiting"
              items={stuckItems}
              emptyText="Nothing is stuck."
              action={{ label: 'All tickets', href: '#tickets' }}
            />
            <ListCard
              title="Recent reports"
              items={reportItems}
              emptyText="No reports yet."
              action={{ label: 'All reports', href: '/admin/reports' }}
            />
          </div>

          <SectionHeader title="Teams" />
          <TeamBreakdown projects={overview?.projects ?? []} members={people} />

          <SectionHeader id="tickets" title="Tickets this week" count={tickets.length} />
          <TicketTable rows={tickets} periodLabel="This week" showAssignees />
        </>
      )}
    </PageLayout>
  )
}
