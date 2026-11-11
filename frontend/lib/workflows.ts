import { FlowStep, TicketRow, Workflow } from 'types/worklog'

// The server's key for BlackBox's steps (its phases) among the workflows in
// /options, beside each SPinno ticket type's
const BLACKBOX = 'blackbox'

// The steps a ticket goes through: a SPinno ticket's type's, or BlackBox's
// phases. Both come from the server, which also moves tickets between them.
export const workflowOf = (row: TicketRow, workflows: Record<string, Workflow>): Workflow | undefined =>
  workflows[row.type || BLACKBOX]

// The step a review-back starts at when a review asks for changes: the one
// the review names (BlackBox: the spec after the confirmation, planning after
// the final review), else the workflow's first review-back step
export const reviewBackStart = (review: FlowStep, workflow: Workflow): FlowStep | undefined =>
  workflow.again.find((s) => s.key === review.changesTo) ?? workflow.again[0]


// Its current step is kept as flowStep (a typed ticket) or step (a phase)
export const currentStepOf = (row: TicketRow) => (row.type ? row.flowStep : row.step) ?? ''
