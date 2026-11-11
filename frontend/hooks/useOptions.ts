import { useEffect, useState } from 'react'

import { fetchOptions } from 'lib/api'
import { Options } from 'types/worklog'

// The choices never change while the app runs, so every caller shares one
// request.
let cached: Promise<Options> | null = null

// The most time one entry can hold (a working day), as the server says;
// this stands in until its options have loaded
const MAX_HOURS = 8

export default function useOptions() {
  const [options, setOptions] = useState<Options | null>(null)

  useEffect(() => {
    cached ??= fetchOptions().catch((e) => {
      cached = null // retry on the next mount
      throw e
    })
    let isCurrent = true
    cached.then((o) => isCurrent && setOptions(o)).catch(() => {})
    return () => {
      isCurrent = false
    }
  }, [])

  const phaseLabel = (value: string) => options?.phases.find((p) => p.value === value)?.label ?? value
  const statusLabel = (value: string) => options?.statuses.find((s) => s.value === value)?.label ?? value
  // A SPinno ticket's type, as the forms name it
  const typeLabel = (repo: string, type: string) =>
    options?.ticketTypes[repo]?.find((t) => t.value === type)?.label ?? type

  const maxHours = options?.maxHours ?? MAX_HOURS

  return { options, maxHours, phaseLabel, statusLabel, typeLabel }
}
