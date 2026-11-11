import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import WeekPicker from 'components/molecules/WeekPicker'
import Modal from 'components/molecules/Modal'
import ImportPreview from 'components/organisms/ImportPreview'
import MemberTable from 'components/organisms/MemberTable'
import useMembers from 'hooks/useMembers'
import useSheetImport from 'hooks/useSheetImport'

import styles from 'styles/Dashboard.module.css'

export default function Members() {
  const { week, setWeek, data, isLoading, error, assign, reload } = useMembers()
  const sheet = useSheetImport(week, reload)

  return (
    <PageLayout
      title="Members"
      heading="Members"
      subheading="Assign teams and roles, and import spreadsheets for anyone who tracked their week offline"
      requireAdmin
      actions={
        <WeekPicker
          week={week}
          onChange={(w) => {
            sheet.cancel()
            setWeek(w)
          }}
        />
      }
    >
      <p className={styles.hint}>
        Use the file from <em>Export .xlsx</em> on a member’s check-ins page as the import template.
      </p>

      {error && <Message variant="error">{error}</Message>}
      {sheet.error && <Message variant="error">{sheet.error}</Message>}
      {sheet.done && <Message variant="success">{sheet.done}</Message>}
      {sheet.member && sheet.isBusy && !sheet.preview && <Message>Reading {sheet.file?.name}…</Message>}
      {sheet.member && sheet.preview && (
        <Modal title={`Import ${sheet.file?.name ?? ''} for ${sheet.member.user.name}`} size="wide" onClose={sheet.cancel}>
          <ImportPreview
            preview={sheet.preview}
            isBusy={sheet.isBusy}
            onConfirm={sheet.confirm}
            onCancel={sheet.cancel}
          />
        </Modal>
      )}

      {isLoading && !data ? (
        <Message>Loading…</Message>
      ) : (
        <MemberTable members={data?.members ?? []} onAssign={assign} onChooseFile={sheet.choose} />
      )}
    </PageLayout>
  )
}
