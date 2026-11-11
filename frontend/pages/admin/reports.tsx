import Icon from 'components/atoms/Icon'
import LinkButton from 'components/atoms/LinkButton'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import ReportGenerator from 'components/organisms/ReportGenerator'
import ReportList from 'components/organisms/ReportList'
import ReportView from 'components/organisms/ReportView'
import useReports from 'hooks/useReports'
import * as api from 'lib/api'

import cardStyles from 'styles/Card.module.css'
import styles from 'styles/ReportsPage.module.css'

export default function Reports() {
  const { reports, selectedId, select, report, isGenerating, error, generate, downloadUrl } = useReports()

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
      {/* The saved reports, with a new one generated from the same card */}
      <section className={cardStyles.card}>
        <div className={styles.toolbar}>
          <h2>Saved reports</h2>
          <ReportGenerator isGenerating={isGenerating} onGenerate={generate} />
        </div>
        <ReportList reports={reports} selectedId={selectedId} onSelect={select} />
      </section>
      {error && <Message variant="error">{error}</Message>}

      <div className={styles.view}>
        {report && downloadUrl ? (
          <ReportView report={report} downloadUrl={downloadUrl} />
        ) : (
          selectedId && <Message>Loading…</Message>
        )}
      </div>
    </PageLayout>
  )
}
