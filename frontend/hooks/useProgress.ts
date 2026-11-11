import { useCallback, useEffect, useState } from 'react'

import useAsyncAction from 'hooks/useAsyncAction'
import useResource from 'hooks/useResource'
import * as progressApi from 'lib/progressApi'
import { ImportResult, Review } from 'types/progress'

export default function useProgress() {
  const [selectedDay, setSelectedDay] = useState<string | null>(null)
  const [lastImport, setLastImport] = useState<ImportResult | null>(null)
  // Bumped after each sync so the selected day reloads with any new prompts
  const [syncCount, setSyncCount] = useState(0)
  // Reviews made here, keyed by day, so an analysis that finishes after the
  // user switched days lands on the right day instead of the one on screen
  const [analyzed, setAnalyzed] = useState<Record<string, Review | null>>({})
  const [analyzingDay, setAnalyzingDay] = useState<string | null>(null)

  const days = useResource(progressApi.fetchDays, [])
  const syncing = useAsyncAction()
  const analyzing = useAsyncAction()
  // The picked day's prompts and saved review
  const day = useResource(
    selectedDay
      ? () => Promise.all([progressApi.fetchPrompts(selectedDay), progressApi.fetchReview(selectedDay)])
      : null,
    [selectedDay, syncCount],
  )

  // Imports new prompts from Claude Code's transcripts, then reloads the day list.
  const { run: runSync } = syncing
  const { reload: reloadDays } = days
  const sync = useCallback(
    () =>
      runSync(async () => {
        setLastImport(await progressApi.importPrompts())
        await reloadDays()
        setSyncCount((n) => n + 1)
      }),
    [runSync, reloadDays],
  )

  useEffect(() => {
    sync()
  }, [sync])

  // The newest day, until one is picked
  useEffect(() => {
    setSelectedDay((current) => current ?? days.data?.[0]?.day ?? null)
  }, [days.data])

  const analyze = async () => {
    if (!selectedDay) return
    const target = selectedDay
    setAnalyzingDay(target)
    await analyzing.run(async () => {
      const review = await progressApi.generateReview(target)
      setAnalyzed((current) => ({ ...current, [target]: review }))
    })
    setAnalyzingDay(null)
  }

  const savedReview = day.data?.[1] ?? null
  return {
    days: days.data ?? [],
    selectedDay,
    selectDay: setSelectedDay,
    prompts: day.data?.[0] ?? [],
    review: selectedDay ? (selectedDay in analyzed ? analyzed[selectedDay] : savedReview) : null,
    lastImport,
    isSyncing: syncing.isBusy,
    isLoadingDay: day.isLoading,
    analyzingDay,
    error: syncing.error ?? analyzing.error ?? days.error ?? day.error,
    sync,
    analyze,
  }
}
