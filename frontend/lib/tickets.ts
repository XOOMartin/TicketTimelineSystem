import { Options, TicketRow } from 'types/worklog'

export type TicketId = { project: string; repo?: string; ticket: string }

// A ticket's key, as the server writes it: "BlackBox#3221", or
// "SPinno/web#120" for a repo's ticket (each repo numbers its own)
export const ticketKeyOf = ({ project, repo, ticket }: TicketId) =>
  repo ? `${project}/${repo}#${ticket}` : `${project}#${ticket}`

// What a ticket number box keeps of what was typed or picked: the part after
// the last "#" (so a suggestion like "SPinno/spec#45" gives 45), digits only
export const ticketDigits = (value: string) => value.slice(value.lastIndexOf('#') + 1).replace(/\D/g, '')

// What a ticket number box was given: "3221", "#3221", or a suggestion like
// "SPinno/web#120", which also says the repo. Only the user's own project
// (current.project) is taken from a prefix; another project's is ignored.
export function parseTicketInput(
  value: string,
  current: { project: string; repo?: string },
  options: Options | null | undefined,
): Required<TicketId> {
  let repo = current.repo ?? ''
  const hash = value.lastIndexOf('#')
  if (hash > 0) {
    const [prefix, prefixRepo = ''] = value.slice(0, hash).split('/')
    if (prefix.toLowerCase() === current.project.toLowerCase())
      repo = reposOf(options, current.project).some((r) => r.value === prefixRepo) ? prefixRepo : ''
  }
  return { project: current.project, repo, ticket: ticketDigits(value) }
}

// A parent box's ticket number: the part after the last "#", as typed (so
// "#120" or a suggestion "SPinno/spec#120" give "120"), checked with
// isTicketNumber
export const parentNumberOf = (value: string) => value.slice(value.lastIndexOf('#') + 1).trim()
export const isTicketNumber = (value: string) => /^\d+$/.test(value)

// Whether two describe the same ticket
export const sameTicket = (a: TicketId, b: TicketId) =>
  a.project === b.project && (a.repo ?? '') === (b.repo ?? '') && a.ticket === b.ticket

// The project's repos (SPinno's spec, backend and web); none for BlackBox
export const reposOf = (options: Options | null | undefined, project: string) => options?.repos?.[project] ?? []

// A repo's short name for chips in the table, e.g. "Backend"
export const repoShort = (repo: string) => ({ spec: 'Spec', backend: 'Backend', web: 'Web' })[repo] ?? repo

// A parent ticket: a spec with no type. Tickets go under it, and its time
// is theirs.
export const isParentTicket = (t: Pick<TicketRow, 'repo' | 'type'>) => t.repo === 'spec' && !t.type

// The parent tickets a ticket in project can be put under (never itself)
export const parentCandidates = (rows: TicketRow[], project: string, exceptKey?: string) =>
  rows.filter((t) => t.project === project && isParentTicket(t) && t.ticketKey !== exceptKey)

// Whether the row's ticket can sit under a spec ticket: any SPinno repo's (a
// bug under its spec, or a spec under a bigger spec)
export const hasParentRepo = (row: Pick<TicketRow, 'repo'>) =>
  row.repo === 'spec' || row.repo === 'backend' || row.repo === 'web'
