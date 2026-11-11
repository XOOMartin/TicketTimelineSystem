import Avatar from 'components/atoms/Avatar'
import { formatShortHours, formatTimeAgo } from 'lib/dates'
import { roundLabel } from 'lib/rounds'
import { plural } from 'lib/text'
import useTeamActivity from 'hooks/useTeamActivity'
import { ActivityState, MemberActivity, WorkingOn } from 'types/worklog'

import cardStyles from 'styles/Card.module.css'
import styles from 'styles/TeamActivity.module.css'

const STATES: Record<ActivityState, { label: string; hint: string }> = {
  busy: { label: 'Busy', hint: 'Has a ticket ongoing' },
  available: { label: 'Available', hint: 'No ticket ongoing' },
}

type Props = {
  // The person looking, whose row says "You"
  currentUserId?: string
}

// "1h 12m", or "<1m" for a timer that just started
function formatElapsed(seconds: number): string {
  return seconds < 60 ? '<1m' : formatShortHours(Math.floor(seconds / 60) / 60)
}

// The ticket and what's being done on it: the phase, and the revision if
// it's not the original work
function Ticket(props: { on: WorkingOn; isQuiet?: boolean }) {
  const { on, isQuiet = false } = props
  return (
    <div className={isQuiet ? `${styles.ticket} ${styles.quiet}` : styles.ticket}>
      <div className={styles.ticketLine}>
        <span className={styles.key}>{on.ticketKey}</span>
        <span className={styles.phase}>{on.phaseLabel}</span>
        {on.round > 1 && <span className={styles.round}>{roundLabel(on.round)}</span>}
      </div>
      <span className={styles.title} title={on.title}>
        {on.title}
      </span>
    </div>
  )
}

// Everyone on the team and whether they're busy (a ticket ongoing) or
// available: busy people first, with the ticket they're on (the one they're
// timing, else their latest ongoing one), then the available with what they
// last worked on. Updates on its own.
export default function TeamActivity(props: Props) {
  const { currentUserId } = props
  const { members, isLoading, error, now, elapsedOf } = useTeamActivity()

  const busy = members?.filter((m) => m.state === 'busy').length ?? 0

  const workingOn = (m: MemberActivity) => {
    if (m.current) {
      // Above the ticket: timing it now, or when they last logged time on it
      const label =
        m.timer === 'running' ? (
          <span className={styles.timing}>Timing now</span>
        ) : m.timer === 'paused' ? (
          <span className={styles.timingPaused}>Timer paused</span>
        ) : m.currentAt ? (
          <span className={styles.lastLabel}>
            Last logged ·{' '}
            <span title={new Date(m.currentAt).toLocaleString()}>{formatTimeAgo(m.currentAt, now)}</span>
          </span>
        ) : null
      return (
        <div className={styles.lastLine}>
          {label}
          <Ticket on={m.current} />
        </div>
      )
    }
    if (m.last && m.lastAt) {
      return (
        <div className={styles.lastLine}>
          <span className={styles.lastLabel}>
            Last worked on · <span title={new Date(m.lastAt).toLocaleString()}>{formatTimeAgo(m.lastAt, now)}</span>
          </span>
          <Ticket on={m.last} isQuiet />
        </div>
      )
    }
    return <span className={styles.none}>Nothing logged yet</span>
  }

  return (
    <section className={`${cardStyles.card} ${styles.card}`}>
      <div className={styles.header}>
        <div>
          <h2 className={styles.heading}>Team activity</h2>
          <p className={styles.subheading}>
            Busy: a ticket ongoing. Available: none. Updates on its own.
          </p>
        </div>
        {members && (
          <div className={styles.summary} aria-label="Team summary">
            <span className={`${styles.count} ${styles.busy}`}>
              <i /> {busy} busy
            </span>
            <span className={`${styles.count} ${styles.available}`}>
              <i /> {members.length - busy} available
            </span>
          </div>
        )}
      </div>

      {error && !members && <p className={styles.message}>{error}</p>}
      {isLoading && <p className={styles.message}>Loading…</p>}
      {members && members.length === 0 && <p className={styles.message}>No members yet.</p>}

      {members && members.length > 0 && (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Member</th>
              <th>Status</th>
              <th>Working on</th>
              <th className={styles.num} title="Time on the ticket they're timing now, pauses left out">
                Time on it
              </th>
              <th className={styles.num} title="Time logged today">
                Today
              </th>
              <th className={styles.num} title="Tickets they’re working on now (status Ongoing)">
                Ongoing
              </th>
            </tr>
          </thead>
          <tbody>
            {members.map((m) => {
              const isYou = m.id === currentUserId
              return (
                <tr key={m.id} className={isYou ? styles.you : undefined}>
                  <td className={styles.memberCell} data-label="Member">
                    <div className={styles.member}>
                      <Avatar name={m.name} size={34} seed={m.id} />
                      <div className={styles.memberText}>
                        <span className={styles.name}>
                          {m.name}
                          {isYou && <span className={styles.youTag}>You</span>}
                        </span>
                        <span className={styles.team}>{m.team}</span>
                      </div>
                    </div>
                  </td>
                  <td data-label="Status">
                    <span className={`${styles.state} ${styles[m.state]}`} title={STATES[m.state].hint}>
                      <i />
                      {STATES[m.state].label}
                    </span>
                  </td>
                  <td className={styles.workingOn} data-label="Working on">
                    {workingOn(m)}
                  </td>
                  <td className={styles.num} data-label="Time on it">
                    {m.timer ? (
                      <span className={m.timer === 'running' ? styles.elapsed : styles.elapsedPaused}>
                        {formatElapsed(elapsedOf(m))}
                      </span>
                    ) : (
                      <span className={styles.none}>—</span>
                    )}
                  </td>
                  <td className={styles.num} data-label="Today">
                    {m.todayHours > 0 ? formatShortHours(m.todayHours) : <span className={styles.none}>—</span>}
                  </td>
                  <td className={styles.num} data-label="Ongoing">
                    <span className={m.ongoingTickets > 0 ? styles.ongoing : styles.none}>
                      {plural(m.ongoingTickets, 'ticket')}
                    </span>
                    {/* Tickets on hold, under it, so they aren't counted as being worked on */}
                    {(m.waitingTickets > 0 || m.blockedTickets > 0) && (
                      <span className={styles.onHold}>
                        {[
                          m.waitingTickets > 0 && `${m.waitingTickets} waiting`,
                          m.blockedTickets > 0 && `${m.blockedTickets} blocked`,
                        ]
                          .filter(Boolean)
                          .join(' · ')}
                      </span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </section>
  )
}
