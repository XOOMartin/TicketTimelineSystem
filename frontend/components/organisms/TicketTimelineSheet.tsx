import {
  CSSProperties,
  DragEvent,
  Fragment,
  KeyboardEvent,
  MouseEvent,
  ReactNode,
  useEffect,
  useRef,
  useState,
} from 'react'

import Badge from 'components/atoms/Badge'
import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import Select from 'components/atoms/Select'
import TextInput from 'components/atoms/TextInput'
import { parseDuration } from 'components/molecules/DurationField'
import ParentTicketOptions from 'components/molecules/ParentTicketOptions'
import SearchBox from 'components/molecules/SearchBox'
import useAsyncAction from 'hooks/useAsyncAction'
import useOptions from 'hooks/useOptions'
import useTimer from 'hooks/useTimer'
import { PHASE_COLORS } from 'lib/colors'
import { formatDay, formatShortDay, formatShortHours, formatTimeOfDay, toDay } from 'lib/dates'
import { shortPhase } from 'lib/phases'
import { latestRound, roundLabel } from 'lib/rounds'
import {
  hasParentRepo,
  isParentTicket,
  isTicketNumber,
  parentNumberOf,
  repoShort,
  sameTicket,
  ticketDigits,
  ticketKeyOf,
} from 'lib/tickets'
import { plural } from 'lib/text'
import { CellChange, TicketRow } from 'types/worklog'

import tableStyles from 'styles/Table.module.css'
import styles from 'styles/TicketTimelineSheet.module.css'

// A soft wash of the phase's chart colour, readable in both themes
const tint = (color: string) => `color-mix(in srgb, ${color} 18%, transparent)`

// Tickets to a page, chosen under the table; the first is the default
// The widest values each column shows, which the headers keep room for so
// the columns never move: two-digit days in long-named months, times with
// two-digit hours, and hours well past a usual ticket's
const WIDEST_DAYS = ['2026-05-28', '2026-09-30', '2026-11-30', '2026-12-31']
const WIDEST_TIMES = [new Date(2026, 0, 1, 10, 58).toISOString(), new Date(2026, 0, 1, 22, 58).toISOString()]
const WIDEST_PHASE_HOURS = 10 + 58 / 60
const WIDEST_TOTAL_HOURS = 100 + 58 / 60

const PAGE_SIZES = [5, 10, 15, 20]
const PAGE_SIZE_KEY = 'ticketTimeline.pageSize'

// The page size this browser last chose (storage can be blocked, e.g. in a
// private window, so the default stands then)
const savedPageSize = () => {
  try {
    const n = Number(localStorage.getItem(PAGE_SIZE_KEY))
    return PAGE_SIZES.includes(n) ? n : PAGE_SIZES[0]
  } catch {
    return PAGE_SIZES[0]
  }
}

// The orders the table can be sorted in, newest added first by default
const SORTS = [
  { value: 'newest', label: 'Newest added' },
  { value: 'oldest', label: 'Oldest added' },
  { value: 'recent', label: 'Recently worked' },
  { value: 'most', label: 'Most time' },
  { value: 'number', label: 'Ticket #' },
] as const
type Sort = (typeof SORTS)[number]['value']

// A time as a number to compare; rows without one (reports) sort last
const timeOf = (iso: string | undefined) => (iso ? Date.parse(iso) || 0 : 0)
// When the ticket went into the timeline, or failing that its start date
const addedOf = (t: TicketRow) => timeOf(t.addedAt) || timeOf(t.startDate)

// A ticket's time with that of the tickets under it (a spec ticket's), in
// all and on one phase
const isSpecRow = (t: TicketRow) => t.repo === 'spec'
// A parent ticket: a spec with no type. Tickets go only under one, and it
// goes under nothing (two levels: parent, then child).
const isParentRow = isParentTicket
// A child spec made a parent by dragging: "Stand-alone" in the table (amber),
// so it reads apart from a ticket added as a parent; it can be dragged back
// under a parent. Exports still count it as a parent.
const isStandAlone = (t: TicketRow) => isParentRow(t) && Boolean(t.madeParent)
// A parent's revisions add in its tickets'
const familyReviewBacks = (t: TicketRow) => t.reviewBacks + (isParentRow(t) ? (t.childReviewBacks ?? 0) : 0)
// A spec's row adds in the tickets under it to its own time
const familyHours = (t: TicketRow) => (isSpecRow(t) ? t.totalHours + (t.childHours ?? 0) : t.totalHours)
const familyPhaseHours = (t: TicketRow, phase: string) =>
  (t.phaseHours[phase] ?? 0) + (isSpecRow(t) ? (t.childPhaseHours?.[phase] ?? 0) : 0)
// A spec ticket whose row adds in the tickets under it
const hasChildTime = (t: TicketRow) => (t.childHours ?? 0) > 0
// The status the table shows: a ticket with time on any phase (its own, or a
// spec's tickets') has started, whatever was saved; otherwise as saved
const shownStatus = (t: TicketRow) => (t.status === 'not_started' && familyHours(t) > 0 ? 'ongoing' : t.status)

// Ties (and rows with no time to compare) keep the sheet's order: project,
// then ticket number
const byKey = (a: TicketRow, b: TicketRow) =>
  a.project.localeCompare(b.project) || a.ticket.localeCompare(b.ticket, undefined, { numeric: true })

const COMPARE: Record<Sort, (a: TicketRow, b: TicketRow) => number> = {
  newest: (a, b) => addedOf(b) - addedOf(a) || byKey(a, b),
  oldest: (a, b) => (addedOf(a) || Infinity) - (addedOf(b) || Infinity) || byKey(a, b),
  recent: (a, b) => timeOf(b.lastUpdate) - timeOf(a.lastUpdate) || byKey(a, b),
  most: (a, b) => familyHours(b) - familyHours(a) || byKey(a, b),
  number: byKey,
}

const isSort = (v: unknown): v is Sort => SORTS.some((s) => s.value === v)

// Whether a ticket was on the go on a day (YYYY-MM-DD): from its start date
// (or, without one, when it was added or first logged) to its end date, or
// today while it's still open
const activeOn = (t: TicketRow, day: string) => {
  if (!day) return true
  const firstLogged = (t.rounds ?? [])
    .map((r) => r.firstDay)
    .filter(Boolean)
    .sort()[0]
  const start = t.startDate || (t.addedAt ? toDay(new Date(t.addedAt)) : '') || firstLogged || ''
  const end = t.endDate || toDay(new Date())
  return (!start || start <= day) && day <= end
}

// Whether a ticket matches the search: its number ("3287", "#3287"), key
// ("BlackBox#3287"), project or title, ignoring case
const matches = (t: TicketRow, query: string) => {
  const q = query.trim().toLowerCase()
  if (!q) return true
  const number = q.startsWith('#') ? q.slice(1) : q
  return (
    t.ticketKey.toLowerCase().includes(q) ||
    t.ticket.toLowerCase().includes(number) ||
    t.project.toLowerCase().includes(q) ||
    repoShort(t.repo ?? '')
      .toLowerCase()
      .includes(q) ||
    t.title.toLowerCase().includes(q)
  )
}

// The key of the spec ticket a backend or web ticket sits under
const parentKeyOf = (t: TicketRow) => ticketKeyOf({ project: t.project, repo: 'spec', ticket: t.parent })

// The page buttons to show: every page when there are few, otherwise the
// first, the last and those around the current one, with gaps as null
const pageList = (current: number, count: number): (number | null)[] => {
  if (count <= 7) return Array.from({ length: count }, (_, i) => i + 1)
  const near = [1, current - 1, current, current + 1, count].filter((p) => p >= 1 && p <= count)
  const pages = [...new Set(near)].sort((a, b) => a - b)
  return pages.flatMap((p, i) => (i > 0 && p - pages[i - 1] > 1 ? [null, p] : [p]))
}

type Props = {
  name: string // whose tickets these are
  rows: TicketRow[]
  // Add Edit and Delete buttons at the end of each row. Double-clicking a
  // row edits it too, with field saying which cell: 'title', 'note',
  // 'status', 'start' or a phase (undefined for the others).
  onEdit?: (row: TicketRow, field?: string) => void
  onDelete?: (row: TicketRow) => void
  // The same per round, on each line of a ticket's breakdown
  onEditRound?: (row: TicketRow, round: number, field?: string) => void
  onDeleteRound?: (row: TicketRow, round: number) => void
  // Adds a button beside Edit that starts the ticket's next revision; on a
  // spec ticket, the button adds a ticket under it instead
  onAddRound?: (row: TicketRow) => Promise<void>
  onAddChild?: (spec: TicketRow) => void
  // Makes a child spec a parent: it's dragged out of any parent
  onMakeParent?: (row: TicketRow) => Promise<void>
  // Makes a parent made by dragging a child again: it's dragged out
  onMakeChild?: (row: TicketRow) => Promise<void>
  // Makes a child spec under no parent a parent, with child under it: another
  // child is dragged onto it
  onMakeParentOf?: (parent: TicketRow, child: TicketRow) => Promise<void>
  // Lets a cell be edited where it is, by double-clicking it: a phase's time
  // (in one round), the title or the status. Other cells open onEdit.
  onSaveCell?: (row: TicketRow, change: CellChange) => Promise<void>
  // Adds a Start button before each ticket that opens its progress and timer
  onOpen?: (row: TicketRow) => void
  editingKey?: string | null
}

// One person's tickets in the order of their tab in the team's
// Ticket_Progress_Timeline spreadsheet: ticket details, hours per phase, then
// the result. Each ticket opens into its rounds (original work, then each
// revision) from its arrow or by clicking the row. With onOpen, each
// ticket starts with a button that opens its progress steps and timer.
// Narrow screens get one card per ticket instead of a scrollbar. A search box
// filters the tickets, shown 5, 10, 15 or 20 to a page (5 by default).
export default function TicketTimelineSheet(props: Props) {
  const { name, rows, onEdit, onDelete, onEditRound, onDeleteRound, onAddRound, onAddChild, onSaveCell, onOpen } = props
  const { onMakeParent, onMakeChild, onMakeParentOf } = props
  const { editingKey = null } = props
  const hasActions = Boolean(onEdit || onDelete || onEditRound || onDeleteRound || onAddRound)
  const hasRoundActions = Boolean(onEditRound || onDeleteRound)
  const { options, maxHours, statusLabel, typeLabel } = useOptions()
  // A SPinno ticket's type, as the form names it
  const typeLabelOf = (t: TicketRow) => typeLabel(t.repo, t.type ?? '')
  const { timer, end: stopTimer, isBusy: isTimerBusy } = useTimer()
  // Ending from the table asks first ("End the timer?"), like the header
  const phases = options?.phases ?? []
  // Tickets whose rounds are shown
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set())
  // A server from before rounds existed sends none
  const roundsOf = (t: TicketRow) => t.rounds ?? []
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState('')
  // A day picked to show only the tickets on the go that day
  const [day, setDay] = useState('')
  const [sort, setSort] = useState<Sort>('newest')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(PAGE_SIZES[0])
  // Read after the first render, so the server's page and the browser's match
  useEffect(() => setPageSize(savedPageSize()), [])
  const changePageSize = (value: string) => {
    const n = Number(value)
    if (!PAGE_SIZES.includes(n)) return
    setPageSize(n)
    setPage(1)
    try {
      localStorage.setItem(PAGE_SIZE_KEY, String(n))
    } catch {
      // Still used for this visit, just not remembered
    }
  }
  // Every visit starts with the newest added ticket at the top
  const changeSort = (value: string) => {
    if (!isSort(value)) return
    setSort(value)
    setPage(1)
  }
  const shown = rows
    .filter((t) => matches(t, query) && (status === '' || shownStatus(t) === status) && activeOn(t, day))
    .sort(COMPARE[sort])
  const isFiltered = query.trim() !== '' || status !== '' || day !== ''
  // SPinno families: a spec ticket, then the tickets under it that show too
  // (backend, web, or smaller specs with their own), nested, kept together and
  // paged as one. The rest stand alone. All keep the chosen order.
  const shownKeys = new Set(shown.map((t) => t.ticketKey))
  const shownByKey = new Map(shown.map((t) => [t.ticketKey, t]))
  const isUnder = (t: TicketRow) => hasParentRepo(t) && Boolean(t.parent) && shownKeys.has(parentKeyOf(t))
  const children = new Map<string, TicketRow[]>()
  for (const t of shown) {
    if (isUnder(t)) children.set(parentKeyOf(t), [...(children.get(parentKeyOf(t)) ?? []), t])
  }
  // The specs above a ticket in the table, nearest first (a loop, which the
  // server refuses, would stop the walk)
  const ancestorsOf = (t: TicketRow) => {
    const up: TicketRow[] = []
    for (let p = t; isUnder(p); ) {
      const parent = shownByKey.get(parentKeyOf(p))
      if (!parent || parent === t || up.includes(parent)) break
      up.push(parent)
      p = parent
    }
    return up
  }
  // A ticket, then each ticket under it in turn with its own
  const familyOf = (t: TicketRow, seen = new Set<string>()): TicketRow[] => {
    if (seen.has(t.ticketKey)) return []
    seen.add(t.ticketKey)
    return [t, ...(children.get(t.ticketKey) ?? []).flatMap((k) => familyOf(k, seen))]
  }
  const groups = shown.filter((t) => !isUnder(t)).map((t) => familyOf(t))
  const pageCount = Math.max(1, Math.ceil(groups.length / pageSize))
  // Stays in range when tickets go away (a delete, another week)
  const current = Math.min(page, pageCount)
  const first = (current - 1) * pageSize
  const pageGroups = groups.slice(first, first + pageSize)
  // Every column: timer, 5 ticket details, the phases, revisions and
  // total, actions
  const columnCount = (onOpen ? 1 : 0) + 5 + phases.length + 2 + (hasActions ? 1 : 0)
  // A spec ticket's breakdown holds its tickets: shown when it's opened, or
  // while searching or filtering, so a match is never tucked away
  const familyOpen = (t: TicketRow) => expanded.has(t.ticketKey) || isFiltered
  // Every ticket is in the table, those under a closed spec (at any level)
  // collapsed: a collapsed row takes no space but still counts toward the
  // column widths, so opening or closing a breakdown never moves the columns
  const pageRows = pageGroups.flat()
  const collapsedKids = new Set(
    pageRows.filter((t) => ancestorsOf(t).some((a) => !familyOpen(a))).map((t) => t.ticketKey),
  )
  // Where each spec's "other issues" line goes: after the last row of its
  // part of the family (an inner spec's line before an outer one's)
  const familyEnd = new Map<string, TicketRow[]>()
  for (const t of pageRows) {
    if (t.repo !== 'spec') continue
    const family = familyOf(t)
    const last = family[family.length - 1].ticketKey
    familyEnd.set(last, [t, ...(familyEnd.get(last) ?? [])])
  }
  // A ticket opens into its rounds; a spec ticket also into its tickets
  const hasBreakdown = (t: TicketRow) =>
    roundsOf(t).length > 0 || hasChildTime(t) || (children.get(t.ticketKey)?.length ?? 0) > 0
  const anyRounds = pageRows.some(hasBreakdown)

  // A backend or web ticket's whole row can be dragged: onto a spec ticket
  // (or a ticket under one) to put it under that spec, or out of its spec
  // (onto a ticket outside any spec family, or the bar over the header) to
  // take it out
  // A child can be dragged; a parent only if it was made one by dragging
  // (it can be dragged back to a child) and has nothing under it, never one
  // added as a parent. Why one can't, for the message trying to shows; ''
  // when it can (or it isn't a parent).
  const cantDragWhy = (t: TicketRow) => {
    if (!onSaveCell || !hasParentRepo(t) || !isParentRow(t)) return ''
    if (!t.madeParent) return 'A parent ticket can’t be dragged'
    if ((children.get(t.ticketKey)?.length ?? 0) > 0) return 'Move its tickets out first to drag it'
    return ''
  }
  const canMove = (t: TicketRow) => Boolean(onSaveCell) && hasParentRepo(t) && cantDragWhy(t) === ''
  // The parent someone just tried to drag, and why it can't be: a red note
  // on its row for a few seconds
  const [dragDenied, setDragDenied] = useState<{ key: string; why: string } | null>(null)
  useEffect(() => {
    if (!dragDenied) return
    const hide = window.setTimeout(() => setDragDenied(null), 3000)
    return () => window.clearTimeout(hide)
  }, [dragDenied])
  // A child spec under no parent dragged out becomes a parent itself; one
  // dragged out of its parent just leaves it, still a child
  const becomesParent = (t: TicketRow | null) =>
    Boolean(t && onMakeParent && t.repo === 'spec' && !isParentRow(t) && !t.parent)
  // ...and a parent made by dragging, with nothing under it, a child again
  const becomesChild = (t: TicketRow | null) =>
    Boolean(t && onMakeChild && isParentRow(t) && t.madeParent && (children.get(t.ticketKey)?.length ?? 0) === 0)
  const [dragKey, setDragKey] = useState<string | null>(null)
  // What's being hovered: a spec family (its spec's key), 'out' for the bar,
  // or 'out:<key>' for a ticket outside any family
  const [dropKey, setDropKey] = useState<string | null>(null)
  const dragged = dragKey ? (rows.find((t) => t.ticketKey === dragKey) ?? null) : null
  // A child spec under no parent: dropping another child on it makes it a
  // parent (made by dragging, so it can be a child again)
  const becomesParentOf = (t: TicketRow) => Boolean(onMakeParentOf) && t.repo === 'spec' && !isParentRow(t) && !t.parent
  // The parent a row belongs to: the parent itself, or the one it's under (a
  // child spec in a family is no place to drop: its parent is), or a child
  // spec under no parent, which becomes the parent
  const specFor = (t: TicketRow) => (isParentRow(t) || becomesParentOf(t) ? t.ticket : isUnder(t) ? t.parent : null)
  const familyKey = (t: TicketRow) => ticketKeyOf({ project: t.project, repo: 'spec', ticket: specFor(t) ?? '' })
  // The spec a drop on this row moves the dragged ticket under ('' for
  // none), or null when there's nothing to do there
  const dropTo = (t: TicketRow): string | null => {
    if (!dragged || t.ticketKey === dragged.ticketKey || t.project !== dragged.project) return null
    const spec = specFor(t)
    if (spec !== null) {
      // A spec can't go under itself or a spec below it (a loop), and a
      // parent with tickets under it can't become a child
      const target = shownByKey.get(familyKey(t))
      if (target && (target === dragged || ancestorsOf(target).includes(dragged))) return null
      if (isParentRow(dragged) && (children.get(dragged.ticketKey)?.length ?? 0) > 0) return null
      return spec === (dragged.parent ?? '') ? null : spec
    }
    // Outside any parent: out of its parent, (a child spec) a parent, or
    // (a parent made by dragging) a child again
    return dragged.parent || becomesParent(dragged) || becomesChild(dragged) ? '' : null
  }
  // The card that follows the pointer: just the ticket being dragged, so its
  // spec (or the rows around it) never come along in the picture
  const ghost = useRef<HTMLElement | null>(null)
  const removeGhost = () => {
    ghost.current?.remove()
    ghost.current = null
  }
  useEffect(() => removeGhost, [])
  const endDrag = () => {
    setDragKey(null)
    setDropKey(null)
    removeGhost()
  }
  const moveTo = async (parent: string) => {
    const d = dragged
    endDrag()
    if (!d || !onSaveCell) return
    if (parent === '' && (becomesParent(d) || becomesChild(d))) {
      try {
        await (becomesParent(d) ? onMakeParent : onMakeChild)?.(d)
      } catch {
        // The page shows what went wrong
      }
      return
    }
    // Open the spec it goes under, to show it there
    if (parent)
      setExpanded((prev) => new Set(prev).add(ticketKeyOf({ project: d.project, repo: 'spec', ticket: parent })))
    // Dropped on a child spec: that spec becomes its parent
    const target = parent
      ? rows.find((r) => r.project === d.project && r.repo === 'spec' && r.ticket === parent)
      : undefined
    try {
      if (target && becomesParentOf(target)) await onMakeParentOf?.(target, d)
      else await onSaveCell(d, { parent })
    } catch {
      // The page shows what went wrong
    }
  }
  // Where the mouse went down: a drag doesn't start from a cell a double-click
  // edits (the hours, the title, the parent chip), so a slightly shaky
  // double-click there always edits; the rest of the row drags
  const pressedOn = useRef<HTMLElement | null>(null)
  const startsEdit = (el: HTMLElement | null) => {
    const field = el?.closest<HTMLElement>('[data-field]')?.dataset.field
    return field !== undefined && (isPhase(field) || field === 'title' || field === 'parent')
  }
  const startDrag = (e: DragEvent<HTMLTableRowElement>, t: TicketRow) => {
    if (startsEdit(pressedOn.current)) {
      e.preventDefault()
      return
    }
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', t.ticketKey)
    // e.g. [Web] #130 · Cart badge wrong count, drawn off screen and used as
    // the drag picture (it must be in the page when the drag begins)
    removeGhost()
    const card = document.createElement('div')
    card.className = styles.dragGhost
    const chip = document.createElement('span')
    chip.className = `${styles.repoChip} ${styles[`repo_${t.repo}`] ?? ''}`
    chip.textContent = repoShort(t.repo)
    const number = document.createElement('span')
    number.className = styles.number
    number.textContent = `#${t.ticket}`
    const title = document.createElement('span')
    title.className = styles.dragGhostTitle
    title.textContent = t.title
    card.append(chip, number, title)
    document.body.append(card)
    ghost.current = card
    e.dataTransfer.setDragImage(card, 18, 18)
    // After the drag has begun: changing the page in dragstart can cancel it
    window.setTimeout(() => setDragKey(t.ticketKey), 0)
  }
  const dragHandlers = (t: TicketRow) => ({
    ...(canMove(t) && cell === null
      ? {
          draggable: true,
          onMouseDown: (e: MouseEvent<HTMLTableRowElement>) => {
            pressedOn.current = e.target as HTMLElement
          },
          onDragStart: (e: DragEvent<HTMLTableRowElement>) => startDrag(e, t),
          onDragEnd: endDrag,
        }
      : {}),
    // A parent that can't move: trying to drag it says why instead
    ...(cantDragWhy(t) && cell === null
      ? {
          draggable: true,
          onMouseDown: (e: MouseEvent<HTMLTableRowElement>) => {
            pressedOn.current = e.target as HTMLElement
          },
          onDragStart: (e: DragEvent<HTMLTableRowElement>) => {
            e.preventDefault()
            // Selecting text in a cell a double-click edits isn't a drag
            if (!startsEdit(pressedOn.current)) setDragDenied({ key: t.ticketKey, why: cantDragWhy(t) })
          },
        }
      : {}),
    ...(dragged
      ? {
          onDragOver: (e: DragEvent<HTMLTableRowElement>) => {
            const to = dropTo(t)
            if (to === null) return
            e.preventDefault()
            e.dataTransfer.dropEffect = 'move'
            const key = to === '' ? `out:${t.ticketKey}` : familyKey(t)
            if (dropKey !== key) setDropKey(key)
          },
          onDragLeave: (e: DragEvent<HTMLTableRowElement>) => {
            if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDropKey(null)
          },
          onDrop: (e: DragEvent<HTMLTableRowElement>) => {
            const to = dropTo(t)
            if (to === null) return
            e.preventDefault()
            moveTo(to)
          },
        }
      : {}),
  })
  // A spec's last line: what its tickets in the table don't account for, by
  // phase and in all (collapsed while the spec is closed); nothing when they
  // all show
  const otherIssues = (spec: TicketRow) => {
    const kids = children.get(spec.ticketKey) ?? []
    // (A smaller spec's time is its own tickets')
    const rest = (spec.childHours ?? 0) - kids.reduce((sum, k) => sum + familyHours(k), 0)
    if (rest < 0.01) return null
    const restOn = (phase: string) =>
      Math.max(0, (spec.childPhaseHours?.[phase] ?? 0) - kids.reduce((sum, k) => sum + familyPhaseHours(k, phase), 0))
    const count = Math.max(0, (spec.childCount ?? 0) - kids.length)
    return (
      <tr
        className={[
          styles.roundRow,
          styles.issuesRow,
          styles.childRounds,
          (!familyOpen(spec) || collapsedKids.has(spec.ticketKey)) && styles.collapsed,
        ]
          .filter(Boolean)
          .join(' ')}
        style={{ '--depth': ancestorsOf(spec).length } as CSSProperties}
      >
        {onOpen && <td className={styles.blank} />}
        <td className={styles.ticket}>
          <span
            className={styles.roundName}
            title="Tickets under this spec that aren't in the table: worked another week, or filtered out"
          >
            <span className={styles.branch} aria-hidden="true" />
            Other issues
            {count > 0 && <span className={styles.issuesCount}>{count}</span>}
          </span>
        </td>
        <td className={`${styles.blank} ${styles.status}`} />
        <td className={styles.dates} colSpan={3} data-label="Worked">
          <span className={styles.end}>Not in this view</span>
        </td>
        {phases.map((p) => phaseCell(spec, -1, p.value, p.label, restOn(p.value)))}
        <td className={styles.blank} />
        <td data-label="Other issues" className={`${styles.num} ${styles.roundTotal}`}>
          {formatShortHours(rest)}
        </td>
        {hasActions && <td className={styles.blank} />}
      </tr>
    )
  }

  // A round's own start: the round the ticket is on gets the play button,
  // opening its steps (a revision's, as its type lays them out); a round
  // already behind it is done, so it shows a tick instead
  const roundStart = (t: TicketRow, round: number) => {
    if (isParentRow(t)) return <td className={styles.blank} />
    const step = t.type ? t.flowStep : t.step
    const isCurrent = round === (t.stepRound || latestRound(t)) && (step ?? '') !== ''
    const isTiming = timer !== null && sameTicket(timer, t) && timer.round === round
    return (
      <td className={styles.start}>
        {isCurrent ? (
          <Button
            variant="primary"
            size="icon"
            className={isTiming ? `${styles.startButton} ${styles.timing}` : styles.startButton}
            onClick={isTiming ? stopTimer : () => onOpen?.(t)}
            disabled={isTiming && isTimerBusy}
            aria-label={
              isTiming
                ? `Stop the timer on ${roundLabel(round)} of ${t.ticketKey}`
                : `Start ${roundLabel(round)} of ${t.ticketKey}`
            }
            title={isTiming ? 'Timing now · stop the timer' : `Start / open the steps of ${roundLabel(round)}`}
          >
            <Icon name={isTiming ? 'stop' : 'play'} size={isTiming ? 16 : 12} />
          </Button>
        ) : (
          <span
            className={styles.roundDone}
            title={`${roundLabel(round)} is done`}
            aria-label={`${roundLabel(round)} is done`}
          >
            <Icon name="check" size={12} />
          </span>
        )}
      </td>
    )
  }

  // Whether the row is in the spec family the dragged ticket is over
  const isDropTarget = (t: TicketRow) =>
    dropKey !== null && !dropKey.startsWith('out') && specFor(t) !== null && familyKey(t) === dropKey
  const isDragOut = dropKey !== null && dropKey.startsWith('out')

  // The ticket whose revision is being added
  const [addingKey, setAddingKey] = useState<string | null>(null)
  // Why a ticket can't get a revision now, or '' if it can
  const cantAddRound = (t: TicketRow) => {
    const rounds = roundsOf(t)
    const last = rounds[rounds.length - 1]
    if (t.totalHours <= 0) return 'Log some time on it first'
    if (t.status === 'done') return 'This ticket is done. Change its status to add a revision.'
    if (t.type && (options?.workflows[t.type]?.again.length ?? 0) === 0)
      return `${typeLabelOf(t)} tickets have no revisions`
    if (last && last.round > 1 && last.hours <= 0) return `${roundLabel(last.round)} has no time yet`
    return ''
  }
  // Adds it, then opens the breakdown to show the new round
  const addRound = async (t: TicketRow) => {
    if (!onAddRound || addingKey) return
    setAddingKey(t.ticketKey)
    try {
      await onAddRound(t)
      setExpanded((prev) => new Set(prev).add(t.ticketKey))
    } catch {
      // The page shows what went wrong
    } finally {
      setAddingKey(null)
    }
  }

  const toggle = (key: string) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })

  // A click on a row waits a moment before opening its rounds, so a
  // double-click (to edit) doesn't also open and close them
  const clickTimer = useRef<number | null>(null)
  const cancelClick = () => {
    if (clickTimer.current !== null) window.clearTimeout(clickTimer.current)
    clickTimer.current = null
  }
  useEffect(() => cancelClick, [])
  const isControl = (e: MouseEvent) => Boolean((e.target as HTMLElement).closest('button, a, input, select, textarea'))
  // The cell that was double-clicked, from its data-field
  const fieldAt = (e: MouseEvent) => (e.target as HTMLElement).closest<HTMLElement>('[data-field]')?.dataset.field

  // The cell being edited in place: its ticket, its line (a round, or null
  // for the ticket's own line) and which field
  const [cell, setCell] = useState<{
    key: string
    round: number | null
    field: string
  } | null>(null)
  const [draft, setDraft] = useState('')
  // Saving a cell; its error box also says what's wrong with what was typed
  const { run: saveCell, isBusy: isSaving, error: cellError, setError: setCellError } = useAsyncAction()
  // Escape, or a save in flight: the blur that follows doesn't save (again)
  const skipBlur = useRef(false)
  const isPhase = (field: string | undefined) => phases.some((p) => p.value === field)
  const isCell = (t: TicketRow, round: number | null, field: string) =>
    cell?.key === t.ticketKey && cell.round === round && cell.field === field
  const hoursIn = (t: TicketRow, round: number, phase: string) =>
    roundsOf(t).find((r) => r.round === round)?.phaseHours[phase] ?? 0
  // The round a phase cell on the ticket's own line stands for: its only one
  const onlyRound = (t: TicketRow) => roundsOf(t)[0]?.round ?? 1

  // Opens a cell for editing; false if it can't be edited in place
  const startCell = (t: TicketRow, round: number | null, field: string | undefined) => {
    if (!onSaveCell || !field) return false
    // A parent's time is its tickets': nothing opens
    if (isPhase(field) && isParentRow(t)) return true
    if (isPhase(field)) {
      // With revisions the ticket's line adds the rounds up (and a child
      // spec's adds in its tickets), so the latest round's own line is the
      // one edited, opened to show it
      if (round === null && (roundsOf(t).length > 1 || (hasChildTime(t) && roundsOf(t).length > 0))) {
        round = latestRound(t)
        setExpanded((prev) => new Set(prev).add(t.ticketKey))
      }
      const hours = hoursIn(t, round ?? onlyRound(t), field)
      setDraft(hours > 0 ? formatShortHours(hours) : '')
    } else if (round === null && field === 'title') {
      setDraft(t.title)
    } else if (round === null && field === 'status') {
      setDraft(shownStatus(t))
    } else if (round === null && field === 'parent' && hasParentRepo(t)) {
      setDraft(t.parent ?? '')
    } else {
      return false
    }
    setCell({ key: t.ticketKey, round, field })
    setCellError(null)
    skipBlur.current = false
    return true
  }

  const closeCell = () => {
    skipBlur.current = true
    setCell(null)
    setCellError(null)
  }

  // Saves the cell (if it changed); on a problem it stays open and says why
  const commitCell = async (t: TicketRow, value = draft) => {
    if (!cell || !onSaveCell || isSaving) return
    const { round, field } = cell
    let change: CellChange | null = null
    if (field === 'title') {
      const title = value.trim()
      if (title === '') return setCellError('Add a short title')
      if (title !== t.title) change = { title }
    } else if (field === 'status') {
      if (value !== shownStatus(t)) change = { status: value }
    } else if (field === 'parent') {
      // "45", "#45" or "SPinno/spec#45"; blank takes it out from under any spec
      const parent = parentNumberOf(value)
      if (parent !== '' && !isTicketNumber(parent)) return setCellError('A parent ticket number, like 45')
      if (parent !== (t.parent ?? '')) change = { parent }
    } else {
      const minutes = parseDuration(value) ?? 0
      if (Number.isNaN(minutes)) return setCellError('Try 1h30m, 1:30 or 90m')
      const r = round ?? onlyRound(t)
      const was = Math.round(hoursIn(t, r, field) * 60)
      if (minutes - was > maxHours * 60) return setCellError(`Add at most ${maxHours} h at a time`)
      if (minutes !== was) change = { round: r, phase: field, minutes }
    }
    if (!change) return closeCell()
    skipBlur.current = true
    const saved = await saveCell(() => onSaveCell(t, change))
    if (saved) closeCell()
    else skipBlur.current = false
  }

  const cellKeys = (t: TicketRow) => (e: KeyboardEvent<HTMLElement>) => {
    if (e.key === 'Escape') {
      // Only the cell: the page's dialogs aren't involved
      e.preventDefault()
      closeCell()
    } else if (e.key === 'Enter') {
      e.preventDefault()
      commitCell(t)
    }
  }
  const cellBlur = (t: TicketRow) => () => {
    if (!skipBlur.current) commitCell(t)
  }
  const cellErrorNote = cellError && (
    <span className={styles.cellError} role="alert">
      {cellError}
    </span>
  )

  // Double-click: edit the cell where it is if it can be, else in the form
  const editOnDoubleClick = (e: MouseEvent, t: TicketRow, round: number | null, edit?: (field?: string) => void) => {
    if (isControl(e)) return
    cancelClick()
    // A double-click selects a word; editing is what they wanted
    window.getSelection()?.removeAllRanges()
    const field = fieldAt(e)
    if (startCell(t, round, field)) return
    edit?.(field)
  }

  // Under a header, the widest values its column can hold, invisible: the
  // column is always that wide, so a value changing (a status, an end date
  // set or cleared) never moves the columns
  const sizer = (...values: ReactNode[]) => (
    <span className={styles.sizer} aria-hidden="true">
      {values.map((v, i) => (
        <span key={i}>{v}</span>
      ))}
    </span>
  )
  const widestDays = WIDEST_DAYS.map(formatShortDay)

  // A ticket's start: the date set on it, or (a spec ticket's) the day its
  // first ticket under it started
  const startDayOf = (t: TicketRow) => t.startDate || (t.startedAt ? toDay(new Date(t.startedAt)) : '')
  // When work began: just the time of day, on hover too
  const startedAtCell = (startedAt: string | undefined) => (
    <td
      className={`${styles.dates} ${styles.startedAt}`}
      data-label="Started at"
      title={startedAt ? `Started at ${formatTimeOfDay(startedAt)}` : 'No time logged yet'}
    >
      {startedAt ? <span>{formatTimeOfDay(startedAt)}</span> : <span className={styles.none}>—</span>}
    </td>
  )

  // Whether double-clicking a phase cell edits it (in place or in the form);
  // round -1 is a spec's "other issues" line, which can't be
  // A parent's time is its tickets': its phase cells can't be edited
  const canEditPhase = (t: TicketRow, round: number | null) =>
    round !== -1 && !isParentRow(t) && Boolean(onSaveCell || (round === null ? onEdit : onEditRound))
  const phaseCell = (t: TicketRow, round: number | null, key: string, label: string, hours: number) => (
    <td
      key={key}
      data-field={key}
      data-label={shortPhase(key, label)}
      title={
        (hours > 0 ? `${label}: ${formatShortHours(hours)}` : `${label}: not started`) +
        (canEditPhase(t, round) && !isCell(t, round, key) ? ' · double-click to edit' : '')
      }
      className={[
        styles.time,
        styles.phase,
        isCell(t, round, key) && styles.cellEditing,
        canEditPhase(t, round) && !isCell(t, round, key) && styles.editableCell,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {isCell(t, round, key) ? (
        <>
          {/* The cell's own content, hidden, keeps the column's width; the
              box floats over it */}
          <span className={styles.cellKeep} aria-hidden="true">
            {hours > 0 ? (
              <span className={styles.pill}>{formatShortHours(hours)}</span>
            ) : (
              <span className={styles.none}>·</span>
            )}
          </span>
          <input
            className={cellError ? `${styles.cellInput} ${styles.cellInvalid}` : styles.cellInput}
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value)
              setCellError(null)
            }}
            onKeyDown={cellKeys(t)}
            onBlur={cellBlur(t)}
            onFocus={(e) => e.target.select()}
            disabled={isSaving}
            placeholder="0m"
            autoFocus
            autoComplete="off"
            spellCheck={false}
            aria-label={`${label} time${round ? `, ${roundLabel(round)}` : ''}`}
            aria-invalid={Boolean(cellError)}
          />
          {cellErrorNote}
          {/* A spec's own time is what's typed; its issues' time is added on */}
          {!cellError && round === null && (t.childPhaseHours?.[key] ?? 0) > 0 && (
            <span className={styles.cellHint}>
              Own time · + {formatShortHours(t.childPhaseHours?.[key] ?? 0)} from issues
            </span>
          )}
        </>
      ) : hours > 0 ? (
        <span className={styles.pill} style={{ background: tint(PHASE_COLORS[key]) }}>
          {formatShortHours(hours)}
        </span>
      ) : (
        <span className={styles.none} aria-label="none">
          ·
        </span>
      )}
    </td>
  )

  return (
    <div className={styles.wrap}>
      <div className={styles.header}>
        <div className={styles.caption}>{name} — Ticket Progress Timeline</div>
        {rows.length > 0 && (
          <div className={styles.controls}>
            <SearchBox
              className={styles.search}
              placeholder="Search ticket # or title"
              value={query}
              onChange={(value) => {
                setQuery(value)
                setPage(1)
              }}
              label="Search tickets"
            />
            <Select
              className={styles.filter}
              aria-label="Filter by status"
              value={status}
              onChange={(e) => {
                setStatus(e.target.value)
                setPage(1)
              }}
              options={[{ value: '', label: 'All statuses' }, ...(options?.statuses ?? [])]}
            />
            <div className={day ? `${styles.dateFilter} ${styles.dateFilterOn}` : styles.dateFilter}>
              <TextInput
                type="date"
                aria-label="Filter by date"
                title="Only the tickets on the go that day"
                value={day}
                onChange={(e) => {
                  setDay(e.target.value)
                  setPage(1)
                }}
              />
              {day && (
                <button
                  type="button"
                  className={styles.dateClear}
                  onClick={() => {
                    setDay('')
                    setPage(1)
                  }}
                  aria-label="Clear the date"
                  title="Any date"
                >
                  <Icon name="close" size={14} />
                </button>
              )}
            </div>
            <Select
              className={styles.filter}
              aria-label="Sort tickets"
              title="Sort tickets"
              value={sort}
              onChange={(e) => changeSort(e.target.value)}
              options={SORTS.map((s) => ({
                value: s.value,
                label: `Sort: ${s.label}`,
              }))}
            />
          </div>
        )}
      </div>
      {/* While a child is dragged: drop here to take it out of its parent,
          or (a child spec) to make it a parent */}
      {(dragged?.parent || becomesParent(dragged) || becomesChild(dragged)) && dragged && (
        <div
          className={isDragOut ? `${styles.dropOut} ${styles.dropOutOver}` : styles.dropOut}
          onDragOver={(e) => {
            e.preventDefault()
            e.dataTransfer.dropEffect = 'move'
            if (dropKey !== 'out') setDropKey('out')
          }}
          onDragLeave={() => setDropKey(null)}
          onDrop={(e) => {
            e.preventDefault()
            moveTo('')
          }}
        >
          {becomesParent(dragged)
            ? `Drop here to make #${dragged.ticket} a parent ticket`
            : becomesChild(dragged)
              ? `Drop here to make #${dragged.ticket} a child ticket again`
              : `Drop here to take #${dragged.ticket} out of parent #${dragged.parent}`}
        </div>
      )}
      {/* The parent tickets here, suggested when putting a ticket under one */}
      <ParentTicketOptions id="sheet-spec-tickets" tickets={rows.filter(isParentRow)} />
      {rows.length === 0 ? (
        <div className={tableStyles.empty}>No tickets were worked on this week.</div>
      ) : shown.length === 0 ? (
        <div className={tableStyles.empty}>
          {`No ${status ? `${statusLabel(status).toLowerCase()} ` : ''}tickets${
            query.trim() ? ` match “${query.trim()}”` : ''
          }${day ? ` on ${formatShortDay(day)}` : query.trim() ? '' : ' this week'}.`}{' '}
          {isFiltered && (
            <button
              type="button"
              className={styles.clearFilters}
              onClick={() => {
                setQuery('')
                setStatus('')
                setDay('')
                setPage(1)
              }}
            >
              Show all
            </button>
          )}
        </div>
      ) : (
        <table className={styles.sheet}>
          <thead>
            <tr className={styles.groups}>
              {onOpen && <th />}
              <th colSpan={5}>
                <span>Ticket details</span>
              </th>
              <th colSpan={phases.length}>
                <span>Hours per phase</span>
              </th>
              <th colSpan={2}>
                <span>Result</span>
              </th>
              {hasActions && <th />}
            </tr>
            <tr>
              {onOpen && <th aria-label="Timer" />}
              <th>Ticket</th>
              <th>
                Status
                {sizer(
                  ...(options?.statuses ?? []).map((st) => (
                    <Badge key={st.value} value={st.value}>
                      {st.label}
                    </Badge>
                  )),
                )}
              </th>
              <th>
                Start
                {sizer(...widestDays)}
              </th>
              <th>
                End
                {sizer(...widestDays)}
              </th>
              {/* The Start column has the day; this is the time of day, so one word fits */}
              <th title="Time work on it started: its first time logged">
                Time
                {sizer(...WIDEST_TIMES.map(formatTimeOfDay))}
              </th>
              {phases.map((p) => (
                <th key={p.value} title={p.label} className={styles.time}>
                  <span className={styles.phaseHead}>
                    <i style={{ background: PHASE_COLORS[p.value] }} />
                    {shortPhase(p.value, p.label)}
                  </span>
                  {sizer(<span className={styles.pill}>{formatShortHours(WIDEST_PHASE_HOURS)}</span>)}
                </th>
              ))}
              <th className={styles.time} title="Times a review sent the ticket back">
                {/* Can break as "Revi-/sions" when the table is short of room */}
                Revi&shy;sions
              </th>
              <th className={styles.num}>
                Total
                {sizer(<span className={styles.total}>{formatShortHours(WIDEST_TOTAL_HOURS)}</span>)}
              </th>
              {hasActions && <th aria-label="Actions" />}
            </tr>
          </thead>
          <tbody>
            {pageRows.map((t) => {
              // Every ticket opens into its rounds, even if only the original
              const hasRounds = hasBreakdown(t)
              const isOpen = hasRounds && (expanded.has(t.ticketKey) || (isFiltered && children.has(t.ticketKey)))
              // Clicking the row opens the breakdown, double-clicking edits it;
              // buttons do their own thing
              const onRowClick = (e: MouseEvent<HTMLTableRowElement>) => {
                if (isControl(e) || !hasRounds) return
                if (!onEdit) return toggle(t.ticketKey)
                if (e.detail > 1) return
                cancelClick()
                clickTimer.current = window.setTimeout(() => {
                  clickTimer.current = null
                  toggle(t.ticketKey)
                }, 220)
              }
              const isTiming = timer !== null && sameTicket(timer, t)
              // Shown under its spec ticket, just above; collapsed while it's closed
              const isChild = isUnder(t)
              const isCollapsed = collapsedKids.has(t.ticketKey)
              // How many specs up it sits: its indent
              const depth = { '--depth': ancestorsOf(t).length } as CSSProperties
              const childCount = children.get(t.ticketKey)?.length ?? 0
              return (
                <Fragment key={t.ticketKey}>
                  <tr
                    className={[
                      (hasRounds || onEdit) && styles.clickable,
                      t.ticketKey === editingKey && styles.editing,
                      isChild && styles.childRow,
                      isCollapsed && styles.collapsed,
                      childCount > 0 && isOpen && styles.parentRow,
                      canMove(t) && styles.movable,
                      isDropTarget(t) && styles.dropTarget,
                      dropKey === `out:${t.ticketKey}` && styles.dropOutRow,
                      t.ticketKey === dragKey && styles.dragging,
                      dragDenied?.key === t.ticketKey && styles.dragDenied,
                    ]
                      .filter(Boolean)
                      .join(' ')}
                    style={depth}
                    {...dragHandlers(t)}
                    onClick={onRowClick}
                    onDoubleClick={
                      onEdit || onSaveCell
                        ? (e) => editOnDoubleClick(e, t, null, onEdit && ((field) => onEdit(t, field)))
                        : undefined
                    }
                    title={
                      canMove(t)
                        ? 'Drag onto a parent ticket to put it under it, or out of its parent · double-click a cell to edit it'
                        : onSaveCell
                          ? 'Double-click a cell to edit it'
                          : onEdit
                            ? 'Double-click to edit'
                            : undefined
                    }
                  >
                    {/* A parent takes no time of its own: no start, its mark instead */}
                    {onOpen && isParentRow(t) && (
                      <td className={styles.start}>
                        <span
                          className={isStandAlone(t) ? `${styles.parentMark} ${styles.standAlone}` : styles.parentMark}
                          title={
                            isStandAlone(t)
                              ? 'Stand-alone ticket: a child made a parent by dragging; its time is the tickets’ under it'
                              : 'Parent ticket: its time is the tickets’ under it'
                          }
                          aria-label={isStandAlone(t) ? 'Stand-alone ticket' : 'Parent ticket'}
                        >
                          <Icon name="layers" size={14} />
                        </span>
                      </td>
                    )}
                    {onOpen && !isParentRow(t) && (
                      <td className={styles.start}>
                        <Button
                          variant="primary"
                          size="icon"
                          className={isTiming ? `${styles.startButton} ${styles.timing}` : styles.startButton}
                          onClick={isTiming ? stopTimer : () => onOpen(t)}
                          disabled={isTiming && isTimerBusy}
                          aria-label={isTiming ? `Stop the timer on ${t.ticketKey}` : `Start ${t.ticketKey}`}
                          title={isTiming ? 'Timing now · stop the timer' : 'Start / open the timer'}
                        >
                          <Icon name={isTiming ? 'stop' : 'play'} size={isTiming ? 16 : 12} />
                        </Button>
                      </td>
                    )}
                    {/* Title first, then one quiet line: number, project, note */}
                    <td className={styles.ticket}>
                      <div className={styles.ticketLine}>
                        {hasRounds ? (
                          <button
                            type="button"
                            className={isOpen ? `${styles.toggle} ${styles.open}` : styles.toggle}
                            onClick={() => toggle(t.ticketKey)}
                            aria-expanded={isOpen}
                            aria-label={`${isOpen ? 'Hide' : 'Show'} the rounds of ${t.ticketKey}`}
                            title={isOpen ? 'Hide rounds' : 'Show each round'}
                          >
                            <Icon name="chevronRight" size={16} />
                          </button>
                        ) : (
                          anyRounds && <span className={styles.toggleSpace} />
                        )}
                        <div className={styles.ticketText}>
                          {isCell(t, null, 'title') ? (
                            <div className={styles.cellEditing}>
                              <div className={`${styles.title} ${styles.cellKeep}`} aria-hidden="true">
                                {t.title}
                              </div>
                              <textarea
                                className={
                                  cellError
                                    ? `${styles.cellInput} ${styles.titleInput} ${styles.cellInvalid}`
                                    : `${styles.cellInput} ${styles.titleInput}`
                                }
                                value={draft}
                                onChange={(e) => {
                                  setDraft(e.target.value.replace(/\s*\n\s*/g, ' '))
                                  setCellError(null)
                                }}
                                onKeyDown={cellKeys(t)}
                                onBlur={cellBlur(t)}
                                onFocus={(e) => e.target.select()}
                                disabled={isSaving}
                                maxLength={200}
                                autoFocus
                                aria-label={`Title of ${t.ticketKey}`}
                                aria-invalid={Boolean(cellError)}
                                rows={2}
                              />
                              {cellErrorNote}
                            </div>
                          ) : (
                            <div
                              className={onSaveCell ? `${styles.title} ${styles.editableText}` : styles.title}
                              title={onSaveCell ? `${t.title} · double-click to edit` : t.title}
                              data-field="title"
                            >
                              {t.title}
                            </div>
                          )}
                          <div
                            className={isCell(t, null, 'parent') ? `${styles.meta} ${styles.metaEditing}` : styles.meta}
                          >
                            {/* A parent says so in its repo chip's place (a parent is
                                always a spec): tickets go under it */}
                            {isParentRow(t) && (
                              <span
                                className={
                                  isStandAlone(t) ? `${styles.parentTag} ${styles.standAlone}` : styles.parentTag
                                }
                                title={
                                  isStandAlone(t)
                                    ? 'Stand-alone: a child ticket made a parent by dragging. Tickets can go under it; drag it onto a parent to make it a child again. Exports count it as a parent.'
                                    : 'Parent ticket (a spec): tickets go under it, and its time is theirs'
                                }
                              >
                                {isStandAlone(t) ? 'Stand-alone' : 'Parent'}
                              </span>
                            )}
                            {/* A repo's ticket leads with its repo, which says the project too */}
                            {t.repo && !isParentRow(t) && (
                              <span className={`${styles.repoChip} ${styles[`repo_${t.repo}`] ?? ''}`}>
                                {repoShort(t.repo)}
                              </span>
                            )}
                            <span className={styles.number}>#{t.ticket}</span>
                            {t.type && <span className={styles.typeTag}>{typeLabelOf(t)}</span>}
                            {!t.repo && <span>{t.project}</span>}
                            {/* A backend or web ticket's spec, or "No spec": double-click
                                to put it under one, move it, or take it out */}
                            {/* (A parent ticket has no chip: it goes under nothing) */}
                            {hasParentRepo(t) &&
                              (!isParentRow(t) || Boolean(t.parent) || isCell(t, null, 'parent')) &&
                              (isCell(t, null, 'parent') ? (
                                <span className={`${styles.parentNote} ${styles.cellEditing}`}>
                                  <input
                                    className={
                                      cellError ? `${styles.parentInput} ${styles.cellInvalid}` : styles.parentInput
                                    }
                                    value={draft}
                                    onChange={(e) => {
                                      setDraft(ticketDigits(e.target.value))
                                      setCellError(null)
                                    }}
                                    inputMode="numeric"
                                    onKeyDown={cellKeys(t)}
                                    onBlur={cellBlur(t)}
                                    onFocus={(e) => e.target.select()}
                                    disabled={isSaving}
                                    list="sheet-spec-tickets"
                                    placeholder="Parent # · none"
                                    autoFocus
                                    autoComplete="off"
                                    aria-label={`Parent ticket of ${t.ticketKey}`}
                                    aria-invalid={Boolean(cellError)}
                                  />
                                  {cellErrorNote}
                                </span>
                              ) : (
                                <span
                                  className={t.parent ? styles.parentNote : `${styles.parentNote} ${styles.noParent}`}
                                  data-field="parent"
                                  title={
                                    onSaveCell
                                      ? t.parent
                                        ? 'Double-click to move it under another parent, or none'
                                        : 'Double-click to put it under a parent ticket'
                                      : undefined
                                  }
                                >
                                  {/* Shown under its spec already, so the number is enough */}
                                  {t.parent ? (isChild ? `↳ #${t.parent}` : `↳ Parent #${t.parent}`) : 'No parent'}
                                </span>
                              ))}
                            {(t.childCount ?? childCount) > 0 && (
                              <span className={styles.parentNote} title="Backend and web tickets under it">
                                {plural(t.childCount ?? childCount, 'issue')}
                              </span>
                            )}
                            {t.note && (
                              <span className={styles.remarks} title={t.note} data-field="note">
                                {t.note}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                      {dragDenied?.key === t.ticketKey && (
                        <p className={styles.dragDeniedNote} role="alert">
                          <Icon name="alert" size={14} />
                          {dragDenied.why}
                        </p>
                      )}
                    </td>
                    <td
                      className={isCell(t, null, 'status') ? `${styles.status} ${styles.cellEditing}` : styles.status}
                      data-label="Status"
                      data-field="status"
                    >
                      {isCell(t, null, 'status') ? (
                        <>
                          <span className={styles.cellKeep} aria-hidden="true">
                            <Badge value={shownStatus(t)}>{statusLabel(shownStatus(t))}</Badge>
                          </span>
                          {/* Choosing a status saves it straight away */}
                          <Select
                            className={styles.cellSelect}
                            value={draft}
                            onChange={(e) => {
                              setDraft(e.target.value)
                              commitCell(t, e.target.value)
                            }}
                            onKeyDown={cellKeys(t)}
                            onBlur={cellBlur(t)}
                            disabled={isSaving}
                            options={options?.statuses ?? []}
                            autoFocus
                            aria-label={`Status of ${t.ticketKey}`}
                          />
                          {cellErrorNote}
                        </>
                      ) : (
                        <Badge value={shownStatus(t)}>{statusLabel(shownStatus(t))}</Badge>
                      )}
                    </td>
                    {/* Start and End dates (double-click: edit them in the form), and
                        when work began, from the time logged */}
                    <td className={styles.dates} data-label="Start" data-field="start">
                      {startDayOf(t) ? (
                        <span title={formatDay(startDayOf(t))}>{formatShortDay(startDayOf(t))}</span>
                      ) : (
                        <span className={styles.none}>—</span>
                      )}
                    </td>
                    <td className={styles.dates} data-label="End" data-field="end">
                      {t.endDate ? (
                        <span title={formatDay(t.endDate)}>{formatShortDay(t.endDate)}</span>
                      ) : (
                        <span className={styles.none} title="Not finished">
                          —
                        </span>
                      )}
                    </td>
                    {startedAtCell(t.startedAt)}
                    {/* A spec ticket's phases add in the tickets under it */}
                    {phases.map((p) => phaseCell(t, null, p.value, p.label, familyPhaseHours(t, p.value)))}
                    <td data-label="Revisions" className={`${styles.time} ${styles.review}`}>
                      {/* A parent's adds in its tickets' revisions */}
                      {familyReviewBacks(t) > 0 ? (
                        <button
                          type="button"
                          className={styles.backs}
                          onClick={() => toggle(t.ticketKey)}
                          disabled={!hasRounds}
                          title={
                            isParentRow(t) && (t.childReviewBacks ?? 0) > 0
                              ? `${t.childReviewBacks} from the tickets under it` +
                                (t.reviewBacks > 0 ? `, ${t.reviewBacks} its own` : '')
                              : hasRounds
                                ? 'Show each round'
                                : undefined
                          }
                        >
                          {familyReviewBacks(t)}
                        </button>
                      ) : (
                        <span className={styles.none}>0</span>
                      )}
                    </td>
                    <td
                      data-label="Total"
                      className={`${styles.num} ${styles.total}`}
                      title={
                        (t.childHours ?? 0) > 0
                          ? `${formatShortHours(familyHours(t))} in all: ${formatShortHours(t.totalHours)} on this spec, ` +
                            `${formatShortHours(t.childHours ?? 0)} on the tickets under it`
                          : undefined
                      }
                    >
                      {/* A spec ticket's total takes in the tickets under it; a
                          ticket added without time has none yet */}
                      {familyHours(t) > 0 ? (
                        formatShortHours(familyHours(t))
                      ) : (
                        <span className={styles.none} title="No time logged yet">
                          —
                        </span>
                      )}
                      {isSpecRow(t) && (t.childHours ?? 0) > 0 && (
                        <span className={styles.familySplit}>
                          {t.totalHours > 0 ? `${formatShortHours(t.totalHours)} own · ` : ''}
                          from {plural(t.childCount ?? 0, 'issue')}
                        </span>
                      )}
                      {/* The ticket's time split by phase, in the chart colours */}
                      {familyHours(t) > 0 && (
                        <span className={styles.split} aria-hidden="true">
                          {phases.map((p) =>
                            familyPhaseHours(t, p.value) > 0 ? (
                              <i
                                key={p.value}
                                style={{
                                  flexGrow: familyPhaseHours(t, p.value),
                                  background: PHASE_COLORS[p.value],
                                }}
                              />
                            ) : null,
                          )}
                        </span>
                      )}
                    </td>
                    {hasActions && (
                      <td className={styles.action}>
                        <div className={styles.actionButtons}>
                          {/* On a parent ticket, + adds a ticket under it (it has no
                              rounds of its own); it opens, to show the new one arrive.
                              A child, spec or not, gets a revision instead */}
                          {onAddRound && isParentRow(t) && !onAddChild && <span className={styles.rowActionSpace} />}
                          {onAddChild && isParentRow(t) && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className={styles.rowAction}
                              title={`Add a ticket under parent #${t.ticket}`}
                              onClick={() => {
                                setExpanded((prev) => new Set(prev).add(t.ticketKey))
                                onAddChild(t)
                              }}
                              aria-label={`Add a ticket under ${t.ticketKey}`}
                            >
                              <Icon name="plus" size={16} />
                            </Button>
                          )}
                          {onAddRound && !isParentRow(t) && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className={styles.rowAction}
                              title={cantAddRound(t) || `Add ${roundLabel(latestRound(t) + 1)}`}
                              onClick={() => addRound(t)}
                              disabled={Boolean(cantAddRound(t)) || addingKey === t.ticketKey}
                              aria-label={`Add a revision to ${t.ticketKey}`}
                            >
                              <Icon name="plus" size={16} />
                            </Button>
                          )}
                          {onEdit && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className={styles.rowAction}
                              title="Edit"
                              onClick={() => onEdit(t)}
                              aria-label={`Edit ${t.ticketKey}`}
                            >
                              <Icon name="edit" size={16} />
                            </Button>
                          )}
                          {onDelete && (
                            <Button
                              variant="ghost"
                              size="icon"
                              className={`${styles.rowAction} ${styles.deleteAction}`}
                              onClick={() => onDelete(t)}
                              aria-label={`Delete ${t.ticketKey}`}
                              title="Delete ticket"
                            >
                              <Icon name="trash" size={16} />
                            </Button>
                          )}
                        </div>
                      </td>
                    )}
                  </tr>
                  {/* One line per round: its own time on each phase, adding up to the
                      row above; always there, collapsed while closed, so opening
                      it doesn't move the columns */}
                  {roundsOf(t).map((r) => (
                    <tr
                      key={r.round}
                      className={[
                        styles.roundRow,
                        onEditRound && styles.clickable,
                        isChild && styles.childRounds,
                        (!isOpen || isCollapsed) && styles.collapsed,
                      ]
                        .filter(Boolean)
                        .join(' ')}
                      style={depth}
                      onDoubleClick={
                        onEditRound
                          ? (e) => editOnDoubleClick(e, t, r.round, (field) => onEditRound(t, r.round, field))
                          : undefined
                      }
                      title={onEditRound ? `Double-click to edit ${roundLabel(r.round)}` : undefined}
                    >
                      {onOpen && roundStart(t, r.round)}
                      <td className={styles.ticket}>
                        <span className={styles.roundName}>
                          <span className={styles.branch} aria-hidden="true" />
                          <span
                            className={r.round > 1 ? styles.roundChip : `${styles.roundChip} ${styles.original}`}
                          >
                            {roundLabel(r.round)}
                          </span>
                        </span>
                      </td>
                      {/* Empty, but holds the Status column so the values below line up */}
                      <td className={`${styles.blank} ${styles.status}`} />
                      {/* The round's first and last day worked, and when it began */}
                      {r.firstDay ? (
                        <>
                          <td className={styles.dates} data-label="Start">
                            <span title={formatDay(r.firstDay)}>{formatShortDay(r.firstDay)}</span>
                          </td>
                          <td className={styles.dates} data-label="Last worked">
                            {r.lastDay && r.lastDay !== r.firstDay && (
                              <span className={styles.end} title={formatDay(r.lastDay)}>
                                {formatShortDay(r.lastDay)}
                              </span>
                            )}
                          </td>
                        </>
                      ) : (
                        // A revision just started has no time, so no days yet: the
                        // note spans Start and End, so Start stays as narrow as a date
                        <td className={styles.dates} colSpan={2} data-label="Start">
                          <span className={styles.end}>Not started</span>
                        </td>
                      )}
                      {startedAtCell(r.startedAt)}
                      {phases.map((p) => phaseCell(t, r.round, p.value, p.label, r.phaseHours[p.value] ?? 0))}
                      <td className={styles.blank} />
                      <td data-label="Round total" className={`${styles.num} ${styles.roundTotal}`}>
                        {r.hours > 0 ? (
                          formatShortHours(r.hours)
                        ) : (
                          <span className={styles.none} title="No time logged yet">
                            —
                          </span>
                        )}
                      </td>
                      {hasActions &&
                        (hasRoundActions ? (
                          <td className={styles.action}>
                            <div className={styles.actionButtons}>
                              {onEditRound && (
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className={styles.rowAction}
                                  title="Edit"
                                  onClick={() => onEditRound(t, r.round)}
                                  aria-label={`Edit ${roundLabel(r.round)} of ${t.ticketKey}`}
                                >
                                  <Icon name="edit" size={16} />
                                </Button>
                              )}
                              {onDeleteRound && (
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className={`${styles.rowAction} ${styles.deleteAction}`}
                                  onClick={() => onDeleteRound(t, r.round)}
                                  aria-label={`Delete ${roundLabel(r.round)} of ${t.ticketKey}`}
                                  title={`Delete ${roundLabel(r.round)}`}
                                >
                                  <Icon name="trash" size={16} />
                                </Button>
                              )}
                            </div>
                          </td>
                        ) : (
                          <td className={styles.blank} />
                        ))}
                    </tr>
                  ))}
                  {/* The end of an open spec's family: time on its tickets that
                      aren't in the table (another week, or filtered out), so the
                      breakdown adds up to the spec's row */}
                  {(familyEnd.get(t.ticketKey) ?? []).map((spec) => (
                    <Fragment key={`other-${spec.ticketKey}`}>{otherIssues(spec)}</Fragment>
                  ))}
                </Fragment>
              )
            })}
            {/* A page short of tickets keeps the height of a full one, so the
                table doesn't shrink and the pager stays put; a single page
                is only kept to the smallest size's height */}
            {Array.from(
              { length: Math.max(0, (pageCount > 1 ? pageSize : PAGE_SIZES[0]) - pageGroups.length) },
              (_, i) => (
                <tr key={`filler-${i}`} className={styles.fillerRow} aria-hidden="true">
                  <td colSpan={columnCount} />
                </tr>
              ),
            )}
          </tbody>
        </table>
      )}
      {shown.length > 0 && (
        <nav className={styles.pager} aria-label={`${name}'s ticket pages`}>
          <span className={styles.pageInfo}>
            {pageGroups.length > 1 ? `${first + 1}–${first + pageGroups.length}` : first + 1} of{' '}
            {plural(groups.length, 'ticket')}
            {/* Those inside a spec's breakdown are counted apart */}
            {shown.length > groups.length && ` · ${shown.length - groups.length} under specs`}
          </span>
          {pageCount > 1 && (
            <div className={styles.pages}>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setPage(current - 1)}
                disabled={current === 1}
                aria-label="Previous page"
              >
                <Icon name="chevronLeft" size={16} />
              </Button>
              {pageList(current, pageCount).map((p, i) =>
                p === null ? (
                  <span key={`gap-${i}`} className={styles.gap}>
                    …
                  </span>
                ) : (
                  <button
                    key={p}
                    type="button"
                    className={p === current ? `${styles.pageButton} ${styles.currentPage}` : styles.pageButton}
                    onClick={() => setPage(p)}
                    aria-current={p === current ? 'page' : undefined}
                    aria-label={`Page ${p}`}
                  >
                    {p}
                  </button>
                ),
              )}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setPage(current + 1)}
                disabled={current === pageCount}
                aria-label="Next page"
              >
                <Icon name="chevronRight" size={16} />
              </Button>
            </div>
          )}
          <label className={styles.pageSize}>
            Rows per page
            <Select
              className={styles.pageSizeSelect}
              value={String(pageSize)}
              onChange={(e) => changePageSize(e.target.value)}
              options={PAGE_SIZES.map((n) => ({ value: String(n), label: String(n) }))}
            />
          </label>
        </nav>
      )}
    </div>
  )
}
