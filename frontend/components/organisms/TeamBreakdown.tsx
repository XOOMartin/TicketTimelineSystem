import Avatar from 'components/atoms/Avatar'
import DonutChart from 'components/molecules/DonutChart'
import useOptions from 'hooks/useOptions'
import { sumBy, toSegments } from 'lib/chartData'
import { PHASE_COLORS, STATUS_COLORS, SUBMISSION_COLORS } from 'lib/colors'
import { formatHours } from 'lib/dates'
import { plural } from 'lib/text'
import { MemberWeek, ProjectRow } from 'types/worklog'

import styles from 'styles/TeamBreakdown.module.css'

const SUBMISSIONS = [
  { value: 'app', label: 'Logged in app' },
  { value: 'imported', label: 'File imported' },
  { value: 'missing', label: 'Missing' },
]

type Props = {
  projects: ProjectRow[]
  members: MemberWeek[]
}

// One row per team: its people, then its tickets, hours and submissions.
export default function TeamBreakdown(props: Props) {
  const { projects, members } = props
  const { options } = useOptions()

  // Every team someone belongs to or worked for, in the configured order
  const known = options?.projects ?? []
  const extra = [...projects.map((p) => p.project), ...members.map((m) => m.user.team)].filter(
    (t) => !known.includes(t),
  )
  const teams = [...known, ...new Set(extra)]

  return (
    <div className={styles.rows}>
      {teams.map((team) => {
        const project = projects.find((p) => p.project === team)
        const people = members.filter((m) => m.user.team === team)
        if (!project && people.length === 0) return null

        return (
          <section key={team} className={styles.row}>
            <div className={styles.team}>
              <h3>{team}</h3>
              <span className={styles.meta}>
                {plural(people.length, 'member')} · {plural(project?.tickets ?? 0, 'ticket')} ·{' '}
                {formatHours(project?.hours ?? 0)} h
              </span>
              <ul className={styles.people}>
                {people.map((m) => (
                  <li key={m.user.id}>
                    <Avatar name={m.user.name} size={22} />
                    <span className={styles.name}>{m.user.name}</span>
                    <i
                      className={styles.dot}
                      style={{ background: SUBMISSION_COLORS[m.submission] }}
                      title={SUBMISSIONS.find((s) => s.value === m.submission)?.label}
                    />
                    <span className={styles.hours}>{formatHours(m.activity.hours)} h</span>
                  </li>
                ))}
              </ul>
            </div>
            <DonutChart
              title="Ticket status"
              unit="tickets"
              segments={toSegments(project?.byStatus ?? {}, options?.statuses ?? [], STATUS_COLORS)}
            />
            <DonutChart
              title="Hours by task"
              unit="hours"
              format={formatHours}
              segments={toSegments(project?.phaseHours ?? {}, options?.phases ?? [], PHASE_COLORS)}
            />
            <DonutChart
              title="Submissions"
              unit="members"
              segments={toSegments(
                sumBy(people, (m) => m.submission),
                SUBMISSIONS,
                SUBMISSION_COLORS,
              )}
            />
          </section>
        )
      })}
    </div>
  )
}
