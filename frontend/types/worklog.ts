// Mirrors the DTOs in internal/interfaces/rest/worklog_dto.go

export type Role = 'member' | 'admin'

export type User = {
  id: string
  name: string
  username: string
  // Empty for accounts made before signup asked for them
  firstName: string
  lastName: string
  team: string
  role: Role
}

// The first and last name are shown in the app and the team sheet; the
// username is for logging in
export type SignupInput = {
  firstName: string
  lastName: string
  username: string
  password: string
  team: string
}

export type Option = {
  value: string
  label: string
}

// A project's code repository: SPinno's spec, backend and web. hasParent
// marks those whose tickets sit under a spec ticket.
export type RepoOption = Option & { hasParent: boolean }

export type Options = {
  projects: string[]
  repos: Record<string, RepoOption[]> // by project; none for BlackBox
  phases: Option[]
  statuses: Option[]
  maxHours: number
  // The types a SPinno repo's tickets can have (the first is the default),
  // by repo, and each type's steps
  ticketTypes: Record<string, Option[]>
  workflows: Record<string, Workflow>
}

// A SPinno ticket type's steps: for the original work, and for each
// review-back round (empty when it has none)
// first: the original work; again: a review-back after Cho-san's changes;
// redo: a review-back after the leads' changes, the original steps again
export type Workflow = { first: FlowStep[]; again: FlowStep[]; redo: FlowStep[] }

export type FlowStep = {
  key: string
  label: string
  who: string // "You", the leads, Cho-san
  // work: timed (or typed in); review: someone else's, its time typed in
  // and a result; action: one click, no time
  kind: 'work' | 'review' | 'action'
  phase: string // the table column its time goes in
  hint: string
  // A review's "Changes requested": back to this step of the round; or, with
  // changesNewRound, a review-back (the next round) starting at it
  changesTo: string
  changesNewRound: boolean
  // "Changes requested" starts a review-back redoing the original steps
  changesRestart: boolean
  approveLabel: string // "" for "Approved"
  // A third result that keeps the step open, e.g. "Still waiting" on a
  // BlackBox confirmation; none for most reviews
  waitLabel?: string
}

export type CheckInInput = {
  project: string
  // SPinno: 'spec', 'backend' or 'web', each numbering its own tickets; ''
  // for BlackBox and SPinno tickets from before repos
  repo?: string
  ticket: string
  title: string
  round: number // 1 = original work, 2 = first review-back, ...
  phase: string
  status: string
  hours: number
  note: string
  workedAt: string // ISO 8601
}

// Time on several phases of one ticket at once, like a row of the progress
// sheet. Phases not done yet are left out.
export type TicketWorkInput = Omit<CheckInInput, 'phase' | 'hours'> & {
  phaseHours: Record<string, number>
  // The ticket's dates, as on the progress sheet: YYYY-MM-DD, or '' for none
  // (no end date means not finished)
  startDate: string
  endDate: string
}

export type CheckIn = CheckInInput & {
  id: string
  ticketKey: string // e.g. BlackBox#3221, or SPinno/web#120 with a repo
  source: 'app' | 'import'
}

export type Period = {
  first: string // YYYY-MM-DD
  last: string
}

export type TicketRow = {
  project: string
  repo: string // see CheckInInput
  ticket: string
  ticketKey: string
  title: string
  assignees: string[]
  phase: string
  status: string
  periodHours: number
  totalHours: number
  phaseHours: Record<string, number>
  reviewBacks: number // the latest round minus one
  rounds: RoundRow[] // original work first; adds up to phaseHours
  // Where the user is in the ticket's progress steps: the phase worked on
  // next ('' once finished) and its round
  step: string
  stepRound: number
  // SPinno: the ticket's type and the step of its workflow it's on ("" once
  // done); both "" for BlackBox
  type?: string
  flowStep?: string
  // A child spec made a parent by dragging: it can be dragged back to a child
  madeParent?: boolean
  lastUpdate: string
  note: string
  startDate: string // YYYY-MM-DD, or '' in reports saved before it was tracked
  endDate: string // YYYY-MM-DD the ticket was marked done, '' while open
  // When it went into the user's timeline (added, or its first time logged);
  // not in reports
  addedAt?: string
  // The spec ticket (its number) a SPinno backend or web ticket sits under,
  // '' for none
  parent: string
  // A spec ticket's time on the backend and web tickets under it, on top of
  // totalHours; 0 for others (and in reports)
  childHours?: number
  // That time by phase, and how many tickets sit under the spec (with time
  // or not)
  childPhaseHours?: Record<string, number>
  childCount?: number
  // A parent's: the review-backs of the tickets under it, added up
  childReviewBacks?: number
  // When work on it began: its first time logged (a spec's, the first on the
  // tickets under it); none until then, and in reports
  startedAt?: string
}

// One cell changed in the table: a round's time on a phase (in minutes), the
// ticket's title or status, or the spec ticket a backend or web ticket sits
// under (its number, '' for none)
export type CellChange =
  | { round: number; phase: string; minutes: number }
  | { title: string }
  | { status: string }
  | { parent: string }

// One pass through the phases: the original work, or the rework after a
// review sent the ticket back
export type RoundRow = {
  round: number
  phaseHours: Record<string, number>
  hours: number
  firstDay: string // YYYY-MM-DD, or "" in a review-back with no time yet
  lastDay: string
  startedAt?: string // when the round's first time was logged
}

export type ProjectRow = {
  project: string
  tickets: number
  byStatus: Record<string, number>
  phaseHours: Record<string, number>
  hours: number
}

export type PersonRow = {
  userId: string
  name: string
  team: string
  checkIns: number
  hours: number
  expectedHours: number
  missedDays: string[]
}

export type MyWeek = {
  period: Period
  checkIns: CheckIn[]
  tickets: TicketRow[]
  person: PersonRow | null
}

// A member's dashboard as an admin sees it
export type MemberDashboard = MyWeek & {
  user: User
}

export type Submission = 'app' | 'imported' | 'missing'

export type ImportRecord = {
  filename: string
  added: number
  skipped: number
  importedAt: string
}

export type MemberWeek = {
  user: User
  activity: PersonRow
  lastImport: ImportRecord | null
  submission: Submission
}

// An account in the admin's user management: deletedAt is set once deleted
// (soft), until it's restored or deleted permanently
export type AdminUser = User & {
  createdAt: string
  deletedAt?: string
}

export type Users = {
  active: AdminUser[]
  deleted: AdminUser[]
}

// An account as an admin adds or edits it; an empty password keeps the
// current one when editing
export type UserInput = {
  firstName: string
  lastName: string
  username: string
  password: string
  team: string
  role: Role
}

export type Members = {
  period: Period
  members: MemberWeek[]
}

export type ImportRow = {
  line: number
  checkIn: CheckIn | null
  error: string
  duplicate: boolean
}

export type SheetImport = {
  period: Period
  rows: ImportRow[]
  valid: number
  skipped: number
  added: number
}

export type ReportKind = 'daily' | 'weekly'

export type ReportSummary = {
  id: string
  kind: ReportKind
  period: Period
  generatedBy: string
  generatedAt: string
  ticketCount: number
}

export type Report = ReportSummary & {
  tickets: TicketRow[]
  projects: ProjectRow[]
  people: PersonRow[]
}

// A timer counting time on one phase of a ticket; ending it logs the time
export type Timer = {
  project: string
  repo?: string
  ticket: string
  ticketKey: string
  title: string
  round: number
  phase: string
  startedAt: string // ISO 8601
  // The clock shows (pausedAt, or now) − startedAt − pausedSeconds
  pausedAt: string | null
  pausedSeconds: number
  // When the site was closed (the timer ends there unless a page comes back)
  closingAt?: string | null
  // A typed ticket's workflow step being timed; '' for none
  step?: string
}

export type TimerInput = Pick<Timer, 'project' | 'repo' | 'ticket' | 'title' | 'round' | 'phase' | 'step'>

export type StoppedTimer = {
  timer: Timer
  minutes: number // logged; 0 when under a minute
  checkIns: CheckIn[]
}

export type CurrentTimer = {
  timer: Timer | null
  stopped?: StoppedTimer // the timer a start ended on the way
}

// How a confirmation or a review went; it decides the next step
export type Outcome = 'no_changes' | 'changes_requested' | 'waiting' | 'approved' | 'done'

export type OutcomeInput = {
  // A typed ticket's workflow step; then phase isn't needed
  step?: string
  // The confirmation or review; or, with outcome done, a timed phase whose
  // time is typed in instead of timed
  phase?: string
  outcome: Outcome
  round: number
  hours: number // the time it took; 0 for none
  comment: string
  title: string // starts a ticket that has no time yet
}
