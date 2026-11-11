import { useState } from 'react'

import useAsyncAction from 'hooks/useAsyncAction'
import useResource from 'hooks/useResource'
import * as api from 'lib/api'
import { ReportRange, defaultReportRange, isValidRange } from 'lib/reports'

// The Reports page: the report type and days picked, shown at once as a live
// preview (fetched as JSON, no page reload) whenever they change; the saved
// reports, one of which can be opened instead; and Generate, which saves the
// days picked.
export default function useReports() {
  const list = useResource(api.fetchReports, [])
  const [range, setRange] = useState<ReportRange>(() => defaultReportRange('weekly'))
  // A saved report opened from the list; null shows the live preview
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const generating = useAsyncAction()

  const isValid = isValidRange(range)
  const { kind, start, end } = range
  // A newer pick's answer replaces an older one still on its way
  const preview = useResource(
    selectedId === null && isValid ? () => api.previewReport(kind, start, end) : null,
    [selectedId, kind, start, end, isValid],
  )
  const detail = useResource(selectedId ? () => api.fetchReport(selectedId) : null, [selectedId])

  // Changing the type or days goes back to the live preview
  const changeRange = (next: ReportRange) => {
    setRange(next)
    setSelectedId(null)
  }

  const generate = () =>
    generating.run(async () => {
      const r = await api.generateReport(kind, start, end)
      detail.setData(r)
      setSelectedId(r.id)
      await list.reload()
    })

  // Deleting the report on screen goes back to the live preview
  const remove = async (id: string) => {
    await api.deleteReport(id)
    if (selectedId === id) setSelectedId(null)
    await list.reload()
  }

  const isPreview = selectedId === null
  const saved = detail.data && detail.data.id === selectedId ? detail.data : null
  return {
    reports: list.data ?? [],
    range,
    changeRange,
    selectedId,
    select: setSelectedId,
    isPreview,
    // While a new preview loads, the last one stays on screen
    report: isPreview ? (isValid ? preview.data : null) : saved,
    isUpdating: isPreview ? preview.isLoading : detail.isLoading,
    isGenerating: generating.isBusy,
    error: generating.error ?? list.error ?? (isPreview ? preview.error : detail.error),
    generate,
    remove,
    downloadUrl: selectedId === null ? api.previewXlsxUrl(kind, start, end) : api.reportXlsxUrl(selectedId),
  }
}
