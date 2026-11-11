import { useState } from 'react'

import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import ConfirmDialog from 'components/molecules/ConfirmDialog'
import ReportGenerator from 'components/organisms/ReportGenerator'
import ReportList from 'components/organisms/ReportList'
import ReportView from 'components/organisms/ReportView'
import useReports from 'hooks/useReports'
import * as api from 'lib/api'
import { REPORT_KIND_LABELS, reportDays } from 'lib/reports'
import { ReportSummary } from 'types/worklog'

import cardStyles from 'styles/Card.module.css'
import styles from 'styles/ReportsPage.module.css'

export default function Reports() {
  const reportsPage = useReports()
  const { reports, range, changeRange, selectedId, select, isPreview, report, isUpdating } = reportsPage
  const { isGenerating, error, generate, remove, downloadUrl } = reportsPage
  // The saved report waiting for the user to confirm deleting it
  const [deleting, setDeleting] = useState<ReportSummary | null>(null)

  return (
    <PageLayout
      title="Reports"
      heading="Reports"
      subheading="Daily and weekly summaries of everyone’s tickets"
      requireAdmin
      actions={
        // Everyone's whole timeline, laid out like the team's progress sheet
        <LinkButton href={api.teamSheetUrl()} download>
          <Icon name="download" size={16} />
          Team sheet .xlsx
        </LinkButton>
      }
    >
      {/* The type and days of report shown below, and the saved reports */}
      <section className={cardStyles.card}>
        <div className={styles.toolbar}>
          <h2>Saved reports</h2>
          <ReportGenerator range={range} onChange={changeRange} isGenerating={isGenerating} onGenerate={generate} />
        </div>
        <ReportList reports={reports} selectedId={selectedId} onSelect={select} onDelete={setDeleting} />
      </section>
      {error && <Message variant="error">{error}</Message>}

      {/* The report for the type and days picked, live; or a saved one */}
      <div className={styles.view}>
        {report ? (
          <ReportView report={report} downloadUrl={downloadUrl} isPreview={isPreview} isUpdating={isUpdating} />
        ) : (
          (isUpdating || selectedId) && <Message>Loading…</Message>
        )}
      </div>

      {deleting && (
        <ConfirmDialog
          title="Delete this report?"
          confirmLabel="Delete report"
          onConfirm={() => remove(deleting.id)}
          onClose={() => setDeleting(null)}
        >
          The <strong>{REPORT_KIND_LABELS[deleting.kind].toLowerCase()} report</strong> for{' '}
          <strong>{reportDays(deleting, true)}</strong> will be removed from Saved reports. This can’t be undone, but
          you can generate it again from the same days. Everyone’s check-ins stay as they are.
        </ConfirmDialog>
      )}
    </PageLayout>
  )
}
