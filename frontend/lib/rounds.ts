import { TicketRow } from 'types/worklog'

// Round 1 is the original work; each review that sends a ticket back for
// changes starts the next round, with its own time on each phase.
export const roundLabel = (round: number) => (round <= 1 ? 'Original' : `Revision ${round - 1}`)

// The ticket's latest round so far (1 if it has none yet)
export const latestRound = (row: TicketRow | undefined) =>
  Math.max(1, ...(row?.rounds ?? []).map((r) => r.round))

// One round's hours per phase. Rows from a server older than rounds have
// none, hence the ?.
export const roundPhaseHours = (row: TicketRow | undefined, round: number): Record<string, number> =>
  row?.rounds?.find((r) => r.round === round)?.phaseHours ?? {}
