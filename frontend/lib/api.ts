import {
  AdminUser,
  CheckIn,
  CheckInInput,
  CurrentTimer,
  MemberDashboard,
  Members,
  MyWeek,
  Options,
  OutcomeInput,
  Report,
  ReportKind,
  ReportSummary,
  SheetImport,
  SignupInput,
  StoppedTimer,
  TicketWorkInput,
  TimerInput,
  User,
  UserInput,
  Users,
} from 'types/worklog'

import { ApiError, BASE_URL, json, request } from 'lib/http'

export { ApiError }

export const fetchOptions = () => request<Options>('/options')

// Resolves to null when nobody is logged in.
export async function fetchMe(): Promise<User | null> {
  try {
    return await request<User>('/auth/me')
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return null
    throw e
  }
}

export const signup = (input: SignupInput) => request<User>('/auth/signup', json('POST', input))
export const login = (username: string, password: string) =>
  request<User>('/auth/login', json('POST', { username, password }))
export const logout = () => request<void>('/auth/logout', { method: 'POST' })

export const fetchMyWeek = (week: string) => request<MyWeek>(`/checkins?week=${week}`)
export const createCheckIns = (input: TicketWorkInput) => request<CheckIn[]>('/checkins/batch', json('POST', input))
// A ticket's routes: /timeline/{project}/{ticket}[/...], with ?repo= for a
// SPinno repo's ticket (each repo numbers its own)
type TicketId = { project: string; repo?: string; ticket: string }
const timelinePath = ({ project, repo, ticket }: TicketId, sub = '') =>
  `/timeline/${encodeURIComponent(project)}/${encodeURIComponent(ticket)}${sub}` +
  (repo ? `?repo=${encodeURIComponent(repo)}` : '')

// Corrects a timeline row: the user's total time on each phase of a ticket
export const setTicketTotals = (input: TicketWorkInput) => request<CheckIn[]>(timelinePath(input), json('PUT', input))
// Removes a timeline row: the user's time on the ticket in every round
export const deleteTicket = (id: TicketId) => request<void>(timelinePath(id), { method: 'DELETE' })
// Puts a ticket in the timeline with no time yet, without a timer; a backend
// or web ticket can go under a spec ticket (its number) straight away
export const addTicket = (id: TicketId, title: string, parent = '', type = '') =>
  request<void>(timelinePath(id), json('POST', { title, parent, type }))
// Changes a SPinno ticket's type, and so its steps
export const setType = (id: TicketId, type: string) => request<void>(timelinePath(id, '/type'), json('PUT', { type }))
// Moves a backend or web ticket under a spec ticket (its number), or none
export const setParent = (id: TicketId, parent: string) =>
  request<void>(timelinePath(id, '/parent'), json('PUT', { parent }))
// Makes a child spec ticket a parent (dragged out of any parent)
export const makeParent = (id: TicketId) => request<void>(timelinePath(id, '/make-parent'), { method: 'POST' })
// Makes a parent made by dragging a child again (dragged out of the parents)
export const makeChild = (id: TicketId) => request<void>(timelinePath(id, '/make-child'), { method: 'POST' })
// Deletes one round of a ticket: its time, or a review-back just started
export const deleteRound = (id: TicketId, round: number) =>
  request<void>(timelinePath(id, `/rounds/${round}`), { method: 'DELETE' })
// Moves a ticket to a step of its progress, e.g. skipping one
export const setStep = (id: TicketId, step: string, round: number) =>
  request<void>(timelinePath(id, '/step'), json('PUT', { step, round }))
// Finishes a confirmation or review step
export const recordOutcome = (id: TicketId, input: OutcomeInput) =>
  request<void>(timelinePath(id, '/outcome'), json('POST', input))
export const updateCheckIn = (id: string, input: CheckInInput) =>
  request<CheckIn>(`/checkins/${encodeURIComponent(id)}`, json('PUT', input))
export const deleteCheckIn = (id: string) => request<void>(`/checkins/${encodeURIComponent(id)}`, { method: 'DELETE' })
// A plain link downloads the file; the browser sends the cookie.
export const exportUrl = (week: string) => `${BASE_URL}/checkins/export?week=${week}`

export const fetchMembers = (week: string) => request<Members>(`/admin/members?week=${week}`)
export const fetchMemberWeek = (id: string, week: string) =>
  request<MemberDashboard>(`/admin/members/${encodeURIComponent(id)}/week?week=${week}`)
export const assignMember = (id: string, team: string, role: string) =>
  request<User>(`/admin/members/${encodeURIComponent(id)}`, json('PUT', { team, role }))

// User management. Deleting is soft (the account can be restored);
// deleting permanently removes a deleted account and all its data.
const userPath = (id: string, rest = '') => `/admin/users/${encodeURIComponent(id)}${rest}`
export const fetchUsers = () => request<Users>('/admin/users')
export const createUser = (input: UserInput) => request<AdminUser>('/admin/users', json('POST', input))
export const updateUser = (id: string, input: UserInput) => request<AdminUser>(userPath(id), json('PUT', input))
export const deleteUser = (id: string) => request<void>(userPath(id), { method: 'DELETE' })
export const restoreUser = (id: string) => request<AdminUser>(userPath(id, '/restore'), { method: 'POST' })
export const deleteUserPermanently = (id: string) => request<void>(userPath(id, '/permanent'), { method: 'DELETE' })

function upload(path: string, file: File) {
  const body = new FormData()
  body.append('file', file)
  return request<SheetImport>(path, { method: 'POST', body })
}

export const previewImport = (userId: string, week: string, file: File) =>
  upload(`/admin/members/${encodeURIComponent(userId)}/import/preview?week=${week}`, file)
export const importSheet = (userId: string, week: string, file: File) =>
  upload(`/admin/members/${encodeURIComponent(userId)}/import?week=${week}`, file)

// This week's report computed live, without saving it
export const fetchOverview = (week: string) => request<Report>(`/admin/overview?week=${week}`)
export const fetchReports = () => request<ReportSummary[]>('/admin/reports')
export const fetchReport = (id: string) => request<Report>(`/admin/reports/${encodeURIComponent(id)}`)
export const generateReport = (kind: ReportKind, day: string) =>
  request<Report>('/admin/reports', json('POST', { kind, day }))
export const reportXlsxUrl = (id: string) => `${BASE_URL}/admin/reports/${encodeURIComponent(id)}/xlsx`
// Everyone's whole timeline in the layout of the team's progress spreadsheet
export const teamSheetUrl = () => `${BASE_URL}/admin/team-sheet.xlsx`

export const fetchTimer = () => request<CurrentTimer>('/timer')
// A page of the site is open: the timer carries on, or, if the site stayed
// closed, it ended at the close and comes back as stopped
// fresh: the page was opened anew (a new tab or window), not reloaded nor
// already open, so a site closed with the timer running really was closed
export const openTimer = (fresh = false) => request<CurrentTimer>('/timer/open', json('POST', { fresh }))
// Sent as the page goes away (close, reload or leaving the site). A beacon is
// the one request a closing page can still send; the server ends the timer
// unless a page opens again soon, as one does on a reload.
export const sendTimerClosing = () => navigator.sendBeacon(`${BASE_URL}/timer/closing`)
// Ends any running timer first, logging its time
export const startTimer = (input: TimerInput) => request<CurrentTimer>('/timer', json('POST', input))
export const stopTimer = () => request<StoppedTimer>('/timer/stop', { method: 'POST' })
export const pauseTimer = () => request<CurrentTimer>('/timer/pause', { method: 'POST' })
export const resumeTimer = () => request<CurrentTimer>('/timer/resume', { method: 'POST' })
