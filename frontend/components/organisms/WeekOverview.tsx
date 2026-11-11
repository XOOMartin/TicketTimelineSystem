import { useRouter } from 'next/router'

import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import DonutChart from 'components/molecules/DonutChart'
import ListCard, { ListCardItem } from 'components/molecules/ListCard'
import StatTile from 'components/molecules/StatTile'
import SectionHeader from 'components/molecules/SectionHeader'
import TicketTimelineSheet from 'components/organisms/TicketTimelineSheet'
import useOptions from 'hooks/useOptions'
import { sumBy, toSegments, toSeriesSegments } from 'lib/chartData'
import { PHASE_COLORS, STATUS_COLORS } from 'lib/colors'
import { formatDateTime, formatHoursMinutes, formatShortDay, formatShortHours } from 'lib/dates'
import { MyWeek, TicketRow } from 'types/worklog'

import cardStyles from 'styles/Card.module.css'
import styles from 'styles/Dashboard.module.css'

type Props = {
  name: string // whose week it is
  data: MyWeek | null
  isLoading: boolean
  // The member's own view links each ticket to editing it in their
  // timeline; an admin's view of someone else doesn't.
  isOwn?: boolean
}

// A member's week: the tickets they worked on and where the time went.
export default function WeekOverview(props: Props) {
  const { name, data, isLoading, isOwn = false } = props
  const { options, phaseLabel, statusLabel } = useOptions()
  const router = useRouter()

  // Opens the ticket in My timeline, in the same week: its progress steps,
  // or the form to edit its times
  const timelineHref = (t: TicketRow, open: 'progress' | 'edit') =>
    isOwn && data ? `/timeline?week=${data.period.first}&${open}=${encodeURIComponent(t.ticketKey)}` : undefined

  // The form, opened on a round's tab and in a field when given
  const editHref = (t: TicketRow, at: { round?: number; field?: string }) => {
    const href = timelineHref(t, 'edit')
    if (!href) return '/timeline'
    const extra = [at.round ? `round=${at.round}` : '', at.field ? `field=${encodeURIComponent(at.field)}` : ''].filter(
      Boolean,
    )
    return extra.length ? `${href}&${extra.join('&')}` : href
  }

  const person = data?.person
  const checkIns = data?.checkIns ?? []
  const tickets = data?.tickets ?? []

  // Two lists that never overlap: what's still open, and what got finished.
  // Both most recently worked on first.
  const byLatest = [...tickets].sort((a, b) => b.lastUpdate.localeCompare(a.lastUpdate))
  const toItem = (t: TicketRow, secondary: string): ListCardItem => ({
    key: t.ticketKey,
    color: STATUS_COLORS[t.status],
    primary: `${t.ticketKey} · ${t.title}`,
    secondary,
    aside: formatHoursMinutes(t.totalHours),
    href: timelineHref(t, 'progress'),
  })

  const inProgress = byLatest
    .filter((t) => t.status !== 'done')
    .map((t) => toItem(t, `${statusLabel(t.status)} · ${phaseLabel(t.phase)} · ${formatDateTime(t.lastUpdate)}`))

  const done = byLatest
    .filter((t) => t.status === 'done')
    .map((t) => toItem(t, `Finished ${t.endDate ? formatShortDay(t.endDate) : formatDateTime(t.lastUpdate)}`))

  return (
    <>
      <div className={styles.stats}>
        <StatTile icon="list" color="var(--blue)" label="Check-ins" value={String(person?.checkIns ?? 0)} />
        <StatTile icon="ticket" color="var(--green)" label="Tickets" value={String(tickets.length)} />
      </div>

      <div className={styles.lists}>
        <ListCard
          title="In progress"
          items={inProgress}
          emptyText="Nothing in progress this week."
          action={isOwn ? { label: 'My timeline', href: '/timeline' } : undefined}
        />
        <ListCard
          title="Done this week"
          items={done}
          emptyText="Nothing finished yet this week."
          action={{ label: 'Ticket details', href: '#tickets' }}
        />
      </div>

      <section className={cardStyles.card}>
        <div className={cardStyles.header}>
          <h2>This week in numbers</h2>
          {isLoading && <span className={cardStyles.hint}>Updating…</span>}
        </div>
        <div className={styles.charts}>
          <DonutChart
            title="Hours by task"
            unit="hours"
            format={formatShortHours}
            segments={toSegments(
              sumBy(
                checkIns,
                (c) => c.phase,
                (c) => c.hours,
              ),
              options?.phases ?? [],
              PHASE_COLORS,
            )}
          />
          <DonutChart
            title="Tickets by status"
            unit="tickets"
            segments={toSegments(
              sumBy(tickets, (t) => t.status),
              options?.statuses ?? [],
              STATUS_COLORS,
            )}
          />
          <DonutChart
            title="Hours by project"
            unit="hours"
            format={formatShortHours}
            segments={toSeriesSegments(
              sumBy(
                checkIns,
                (c) => c.project,
                (c) => c.hours,
              ),
            )}
          />
        </div>
      </section>

      <SectionHeader
        id="tickets"
        title="Tickets"
        count={tickets.length}
        actions={
          isOwn && (
            <LinkButton href="/timeline" variant="outline" size="sm">
              <Icon name="edit" size={14} />
              Log or edit in My timeline
            </LinkButton>
          )
        }
      />
      {isLoading && !data ? (
        <Message>Loading…</Message>
      ) : (
        <TicketTimelineSheet
          name={name}
          rows={tickets}
          onEdit={isOwn ? (t, field) => router.push(editHref(t, { field })) : undefined}
          onEditRound={isOwn ? (t, round, field) => router.push(editHref(t, { round, field })) : undefined}
          onOpen={isOwn ? (t) => router.push(timelineHref(t, 'progress') ?? '/timeline') : undefined}
        />
      )}
    </>
  )
}
