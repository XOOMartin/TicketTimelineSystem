import Button from 'components/atoms/Button'
import Message from 'components/atoms/Message'
import PageLayout from 'components/layouts/PageLayout'
import DayList from 'components/organisms/DayList'
import PromptList from 'components/organisms/PromptList'
import ReviewPanel from 'components/organisms/ReviewPanel'
import useProgress from 'hooks/useProgress'
import { plural } from 'lib/text'

import styles from 'styles/ProgressPage.module.css'

export default function Progress() {
  const {
    days,
    selectedDay,
    selectDay,
    prompts,
    review,
    lastImport,
    isSyncing,
    isLoadingDay,
    analyzingDay,
    error,
    sync,
    analyze,
  } = useProgress()

  return (
    <PageLayout title="My Progress" heading="My Progress">
      <div className={styles.toolbar}>
        <Button onClick={sync} disabled={isSyncing}>
          {isSyncing ? 'Syncing…' : 'Sync from Claude Code'}
        </Button>
        {lastImport && (
          <Message>
            {lastImport.added === 0 ? 'Up to date.' : `Added ${plural(lastImport.added, 'new prompt')}.`}
          </Message>
        )}
      </div>

      {error && <Message variant="error">{error}</Message>}

      <div className={styles.columns}>
        <DayList days={days} selectedDay={selectedDay} onSelect={selectDay} />

        {selectedDay && (
          <div>
            <h2 className={styles.day}>{selectedDay}</h2>
            <ReviewPanel
              review={review}
              isAnalyzing={analyzingDay === selectedDay}
              // One analysis at a time; it runs the Claude CLI on the server
              canAnalyze={analyzingDay === null && prompts.length > 0 && !isLoadingDay}
              onAnalyze={analyze}
            />
            <h3>Prompts ({prompts.length})</h3>
            {isLoadingDay && prompts.length === 0 ? <Message>Loading…</Message> : <PromptList prompts={prompts} />}
          </div>
        )}
      </div>
    </PageLayout>
  )
}
