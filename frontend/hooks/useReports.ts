import { useEffect, useState } from 'react'

import useAsyncAction from 'hooks/useAsyncAction'
import useResource from 'hooks/useResource'
import * as api from 'lib/api'
import { ReportKind } from 'types/worklog'

// The saved reports, the one picked (the newest to start with), and making
// a new one.
export default function useReports() {
  const list = useResource(api.fetchReports, [])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  // A report picked while another is loading is the one shown
  const detail = useResource(selectedId ? () => api.fetchReport(selectedId) : null, [selectedId])
  const generating = useAsyncAction()

  // The newest report, until one is picked
  useEffect(() => {
    setSelectedId((current) => current ?? list.data?.[0]?.id ?? null)
  }, [list.data])

  const generate = (kind: ReportKind, day: string) =>
    generating.run(async () => {
      const r = await api.generateReport(kind, day)
      detail.setData(r)
      setSelectedId(r.id)
      await list.reload()
    })

  const report = detail.data
  return {
    reports: list.data ?? [],
    selectedId,
    select: setSelectedId,
    report: report && report.id === selectedId ? report : null,
    isGenerating: generating.isBusy,
    error: generating.error ?? list.error ?? detail.error,
    generate,
    downloadUrl: selectedId ? api.reportXlsxUrl(selectedId) : null,
  }
}
