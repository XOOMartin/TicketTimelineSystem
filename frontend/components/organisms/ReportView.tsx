import { useState } from 'react'

import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import DonutChart from 'components/molecules/DonutChart'
import TicketTable from 'components/organisms/TicketTable'
import useOptions from 'hooks/useOptions'
import { sumBy, toSegments, toSeriesSegments } from 'lib/chartData'
import { PHASE_COLORS, STATUS_COLORS } from 'lib/colors'
import { formatDay, formatHours } from 'lib/dates'
import { REPORT_KIND_LABELS, reportDays } from 'lib/reports'
import { Report } from 'types/worklog'

import cardStyles from 'styles/Card.module.css'
import viewStyles from 'styles/ReportView.module.css'
import styles from 'styles/Table.module.css'

const TABS = ['Tickets', 'Projects', 'People'] as const

// The column of hours worked in the report's days
const PERIOD_LABELS: Record<Report['kind'], string> = {
  daily: 'Today',
  weekly: 'This week',
}

type Props = {
  report: Report
  downloadUrl: string
  // Built from the current data for the days picked, not saved yet
  isPreview?: boolean
  // A newer version is loading; this one stays until it arrives
  isUpdating?: boolean
}

export default function ReportView(props: Props) {
  const { report, downloadUrl, isPreview = false, isUpdating = false } = props
  const { options } = useOptions()
  const [tab, setTab] = useState<(typeof TABS)[number]>('Tickets')

  const phases = options?.phases ?? []
  const statuses = options?.statuses ?? []

  return (
    <section className={cardStyles.card} aria-busy={isUpdating || undefined}>
      <div className={viewStyles.header}>
        <div>
          <span className={viewStyles.eyebrow}>
            {REPORT_KIND_LABELS[report.kind]} report
            {isPreview && (
              <span className={viewStyles.live} title="Up to date with everyone’s latest check-ins. Generate saves it.">
                Live · not saved
              </span>
            )}
            {isUpdating && <span className={viewStyles.updating}>Updating…</span>}
          </span>
          <h2>{reportDays(report, true)}</h2>
        </div>
        <LinkButton href={downloadUrl} download>
          <Icon name="download" size={16} />
          .xlsx
        </LinkButton>
      </div>

      <div className={viewStyles.charts}>
        <DonutChart
          title="Tickets by status"
          unit="tickets"
          segments={toSegments(
            sumBy(report.tickets, (t) => t.status),
            statuses,
            STATUS_COLORS,
          )}
        />
        <DonutChart
          title="Hours by task"
          unit="hours"
          format={formatHours}
          segments={toSegments(
            sumBy(
              report.projects.flatMap((p) => Object.entries(p.phaseHours)),
              ([phase]) => phase,
              ([, h]) => h,
            ),
            phases,
            PHASE_COLORS,
          )}
        />
        <DonutChart
          title="Hours by project"
          unit="hours"
          format={formatHours}
          segments={toSeriesSegments(
            sumBy(
              report.projects,
              (p) => p.project,
              (p) => p.hours,
            ),
          )}
        />
      </div>

      {/* The tabs switch the table under them; the charts are the whole report */}
      <div className={viewStyles.tabs} role="tablist">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            className={viewStyles.tab}
            aria-selected={t === tab}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === 'Tickets' && (
        <TicketTable rows={report.tickets} periodLabel={PERIOD_LABELS[report.kind]} showAssignees />
      )}

      {tab === 'Projects' &&
        (report.projects.length === 0 ? (
          <div className={`${styles.wrap} ${styles.empty}`}>No tickets were worked on in this period.</div>
        ) : (
          <div className={styles.wrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Project</th>
                  <th className={styles.num}>Tickets</th>
                  {statuses.map((s) => (
                    <th key={s.value} className={styles.num}>
                      {s.label}
                    </th>
                  ))}
                  <th className={styles.num}>Hours</th>
                  {phases.map((p) => (
                    <th key={p.value} className={styles.num}>
                      {p.label} (h)
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {report.projects.map((p) => (
                  <tr key={p.project}>
                    <td>{p.project}</td>
                    <td className={styles.num}>{p.tickets}</td>
                    {statuses.map((s) => (
                      <td key={s.value} className={styles.num}>
                        {p.byStatus[s.value] ?? 0}
                      </td>
                    ))}
                    <td className={styles.num}>{formatHours(p.hours)}</td>
                    {phases.map((ph) => (
                      <td key={ph.value} className={styles.num}>
                        {p.phaseHours[ph.value] ? formatHours(p.phaseHours[ph.value]) : ''}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}

      {tab === 'People' && (
        <div className={styles.wrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Name</th>
                <th>Team</th>
                <th className={styles.num}>Check-ins</th>
                <th className={styles.num}>Hours / expected</th>
                <th>Days without check-ins</th>
              </tr>
            </thead>
            <tbody>
              {report.people.map((p) => (
                <tr key={p.userId}>
                  <td>{p.name}</td>
                  <td>{p.team}</td>
                  <td className={styles.num}>{p.checkIns}</td>
                  <td className={styles.num}>
                    {formatHours(p.hours)} / {formatHours(p.expectedHours)}
                  </td>
                  <td className={styles.muted}>{p.missedDays.map(formatDay).join(', ') || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
