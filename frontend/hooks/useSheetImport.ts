import { useState } from 'react'

import useAsyncAction from 'hooks/useAsyncAction'
import * as api from 'lib/api'
import { plural } from 'lib/text'
import { MemberWeek, SheetImport } from 'types/worklog'

// Two steps: choosing a file shows a preview; confirming saves the rows.
export default function useSheetImport(week: string, onImported: () => Promise<void>) {
  const [member, setMember] = useState<MemberWeek | null>(null)
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<SheetImport | null>(null)
  const { run, isBusy, error, setError } = useAsyncAction()
  const [done, setDone] = useState<string | null>(null)

  const choose = (m: MemberWeek, f: File) => {
    setMember(m)
    setFile(f)
    setPreview(null)
    setDone(null)
    return run(async () => setPreview(await api.previewImport(m.user.id, week, f)))
  }

  const confirm = () =>
    member && file
      ? run(async () => {
          const result = await api.importSheet(member.user.id, week, file)
          setDone(
            `Imported ${plural(result.added, 'check-in')} for ${member.user.name}` +
              (result.skipped > 0 ? ` (${result.skipped} skipped).` : '.'),
          )
          setMember(null)
          setPreview(null)
          await onImported()
        })
      : Promise.resolve(false)

  const cancel = () => {
    setMember(null)
    setFile(null)
    setPreview(null)
    setError(null)
  }

  return { member, file, preview, isBusy, error, done, choose, confirm, cancel }
}
