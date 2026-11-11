import { FormEvent, Fragment, useState } from 'react'

import Button from 'components/atoms/Button'
import Icon from 'components/atoms/Icon'
import FormField from 'components/molecules/FormField'
import Modal from 'components/molecules/Modal'
import SelectField from 'components/molecules/SelectField'
import ParentTicketOptions from 'components/molecules/ParentTicketOptions'
import FormActions from 'components/molecules/FormActions'
import ReadOnlyField from 'components/molecules/ReadOnlyField'
import useOptions from 'hooks/useOptions'
import useAsyncAction from 'hooks/useAsyncAction'
import useTouched from 'hooks/useTouched'
import {
  hasParentRepo,
  isTicketNumber,
  parentCandidates,
  parentNumberOf,
  parseTicketInput,
  reposOf,
  sameTicket,
  ticketDigits,
} from 'lib/tickets'
import { TicketRow } from 'types/worklog'

import styles from 'styles/Card.module.css'

// repo: SPinno's spec, backend or web ('' for BlackBox); parent: the spec
// ticket number a backend or web ticket sits under ('' for none); type: a
// SPinno ticket's type, which decides its steps ('' for BlackBox)
export type TicketDetails = {
  project: string
  repo: string
  ticket: string
  title: string
  parent: string
  type: string
}

type Props = {
  defaultProject: string
  // Adding a ticket under this spec ticket (its + in the table): its project,
  // and the parent fixed to it (a spec, backend or web ticket)
  underSpec?: TicketRow
  // What was entered before (going back from the progress steps' Previous)
  initial?: TicketDetails
  // Tickets the user already has, offered as suggestions
  knownTickets: TicketRow[]
  // May save the parent spec first; a failure (no such spec) keeps it open
  onContinue: (details: TicketDetails) => void | Promise<void>
  // Saves a new ticket straight to the table, with no time and no timer
  onAdd?: (details: TicketDetails) => Promise<void>
  onClose: () => void
}

// The number in "120", "#120" or "SPinno/spec#120"

// The first part of Input timeline: which ticket. The progress steps come
// next, the same as when clicking a ticket in the table, or a new ticket can
// just be added to the table to start later. A project with repos (SPinno)
// asks which repo too, and a backend or web ticket which spec ticket it sits
// under.
export default function TicketDetailsDialog(props: Props) {
  const { defaultProject, underSpec, initial, knownTickets, onContinue, onAdd, onClose } = props
  const { options } = useOptions()
  const [project, setProject] = useState(initial?.project ?? underSpec?.project ?? defaultProject)
  const [repo, setRepo] = useState(initial?.repo ?? '')
  const [ticket, setTicket] = useState(initial?.ticket ?? '')
  const [title, setTitle] = useState(initial?.title ?? '')
  const [parent, setParent] = useState(initial?.parent ?? underSpec?.ticket ?? '')
  const [type, setType] = useState(initial?.type ?? '')
  // Parent or child, chosen before the repo: a parent ticket is a spec
  // (tickets go under it); a child is a spec, backend or web ticket that
  // sits under a spec (a backend or web one can get its spec later). From a
  // spec's +, a child of it.
  const levelFor = (r: string, p: string) => (r === 'spec' && !p ? 'parent' : 'child')
  const [level, setLevel] = useState<'' | 'parent' | 'child'>(
    underSpec ? 'child' : initial ? levelFor(initial.repo, initial.parent) : '',
  )
  // Busy until the dialog closes or moves on to the ticket's steps
  const { run, isBusy: isAdding, error, setError } = useAsyncAction()

  // All of the project's repos, also under a spec: a backend or web ticket,
  // or a smaller spec, can sit under one
  const repos = reposOf(options, project)
  const isSpec = repo === 'spec'
  const hasParent = hasParentRepo({ repo })
  const known = knownTickets.find((t) => sameTicket(t, { project, repo, ticket: ticket.trim() }))
  // A SPinno ticket's type: a spec is an investigation or an integration
  // test, a backend or web ticket an implementation or a review. A ticket
  // the user already has keeps its own (changed in Edit).
  // A parent ticket has no type: its steps are the six phases
  const isParentLevel = level === 'parent'
  const types = isParentLevel ? [] : (options?.ticketTypes[repo] ?? [])
  const shownType = isParentLevel ? '' : known?.type || type || types[0]?.value || ''
  const steps = isParentLevel ? [] : (options?.workflows[shownType]?.first ?? [])
  // The parent tickets the user has in this project (specs with no type), to
  // put a ticket under; a child spec can't have tickets under it
  const specs = parentCandidates(knownTickets, project)
  const childSpec = (n: string) =>
    knownTickets.find((t) => t.project === project && t.repo === 'spec' && t.ticket === n && Boolean(t.type))

  // A parent ticket is a spec, so choosing it picks the Spec repo
  const changeLevel = (value: string) => {
    const next = value === 'parent' ? 'parent' : 'child'
    setLevel(next)
    setError(null)
    if (next === 'parent') {
      setParent('')
      if (repo !== 'spec') {
        setRepo('spec')
        setType('')
      }
    }
  }
  const levelRepos = level === 'parent' ? repos.filter((r) => r.value === 'spec') : repos

  const changeRepo = (value: string) => {
    setRepo(value)
    setType('')
    setError(null)
    if (underSpec) setParent(underSpec.ticket)
    else if (!hasParentRepo({ repo: value })) setParent('')
  }

  // Accepts "3221", "#3221" or a suggestion like "BlackBox#3221" or
  // "SPinno/web#120", and fills in the title (and parent) of a ticket the
  // user already has
  const handleTicketChange = (value: string) => {
    const { project: proj, repo: rep, ticket: number } = parseTicketInput(value, { project, repo }, options)
    setTicket(number)
    setProject(proj)
    setRepo(rep)
    const seen = knownTickets.find((t) => sameTicket(t, { project: proj, repo: rep, ticket: number }))
    if (seen) {
      setTitle(seen.title)
      setParent(seen.parent ?? '')
      setLevel(levelFor(rep, seen.parent ?? ''))
    }
  }

  const parentNumber = parentNumberOf(parent)
  let missing = ''
  if (repos.length > 0 && !level) missing = 'Choose whether it’s a parent or a child ticket'
  else if (repos.length > 0 && !repo) missing = `Choose the ${project} repo`
  else if (ticket.trim() === '') missing = 'Enter the ticket number'
  else if (title.trim() === '') missing = 'Add a short title'
  else if (parent.trim() && !isTicketNumber(parentNumber)) missing = 'The parent is a ticket number, like 120'
  else if (level === 'child' && parentNumber !== '' && childSpec(parentNumber))
    missing = `#${parentNumber} is a child ticket: tickets go only under a parent`

  const details = (): TicketDetails => ({
    project,
    repo,
    ticket: ticket.trim(),
    title: title.trim(),
    parent: hasParent && level === 'child' ? parentNumber : '',
    type: types.length > 0 ? shownType : '',
  })

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    // A parent ticket has no steps to continue to: Enter adds it
    if (parentOnly) return handleAdd()
    if (missing) return
    await run(async () => onContinue(details()), { stayBusy: true })
  }

  // A parent ticket has no steps of its own (its time is its tickets'), so
  // it's only added to the table: no Continue
  const parentOnly = isParentLevel && Boolean(onAdd)

  // The dialog closes once the ticket is saved; on failure it stays open
  const handleAdd = async () => {
    if (missing || !onAdd) return
    await run(async () => onAdd(details()), { stayBusy: true })
  }

  // The parent spec must already be in the timeline; the server has the last
  // word (a spec from an earlier week isn't in this list)
  const unknownSpec = hasParent && parentNumber !== '' && !specs.some((t) => t.ticket === parentNumber)

  // Each field's problem, shown beside it once it's been filled in or left
  const { touch, shown } = useTouched()
  const fieldErrors = {
    level: repos.length > 0 && !level ? 'Choose one' : '',
    repo: repos.length > 0 && level && !repo ? 'Choose a repo' : '',
    ticket: ticket.trim() === '' ? 'Enter the number' : '',
    title: title.trim() === '' ? 'Add a short title' : '',
    // Optional for every child: its spec can be set later
    parent:
      level === 'child' && parentNumber !== '' && childSpec(parentNumber)
        ? 'A child ticket, not a parent'
        : level === 'child' && unknownSpec
          ? 'Not in your timeline'
          : '',
  }

  let hint = missing
  if (error) hint = `Couldn’t save it: ${error}`
  else if (!missing && unknownSpec)
    hint = childSpec(parentNumber)
      ? `#${parentNumber} is a child ticket: tickets go only under a parent`
      : `Parent #${parentNumber} isn’t in this week’s table: it must already be in your timeline`
  else if (!missing && parentOnly && known) hint = 'You already have this ticket.'
  else if (!missing && parentOnly) hint = 'A new parent ticket.'
  else if (!missing && isSpec && known) hint = 'You already have this spec ticket; Continue with timer opens its steps.'
  else if (!missing && known) hint = 'You already have this ticket; Continue with timer opens its progress.'
  else if (!missing) hint = 'A new ticket.'

  const ticketField = (
    <FormField
      label="Ticket #"
      value={ticket}
      onChange={(e) => {
        handleTicketChange(e.target.value)
        touch('ticket')
      }}
      onBlur={() => touch('ticket')}
      error={shown('ticket', fieldErrors.ticket)}
      // No suggestions of tickets you have: it only warns when it's one
      warning={known ? 'You already have this ticket' : undefined}
      inputMode="numeric"
      data-autofocus={repos.length === 0 || undefined}
      aria-required
    />
  )

  return (
    <Modal title={underSpec ? `Add a ticket under parent #${underSpec.ticket}` : 'Input timeline'} onClose={onClose}>
      {/* One fixed size whatever it holds (BlackBox or SPinno, which fields
          show, hints), so the form doesn't jump while it's filled in */}
      <form onSubmit={handleSubmit} className={styles.detailsForm}>
        <p className={`${styles.hint} ${styles.stepHint} ${styles.detailsIntro}`}>
          {underSpec
            ? `A ticket under “${underSpec.title}”: a backend or web ticket, or a smaller spec. Its time counts toward the spec’s.`
            : repos.length > 0
              ? 'Parent or child first, then the repo and its details. A parent is a spec ticket; a child sits under one.'
              : 'Ticket details first, then time it now or add it to start later.'}
        </p>
        <div className={styles.pairs}>
          {/* Fixed: the project the user chose at signup (or, going back to
              a ticket they already have, that ticket's) */}
          <ReadOnlyField label="Project" value={project} title="Your team’s project" />
          {repos.length > 0 ? (
            // SPinno: parent or child first; it decides the repos offered
            <SelectField
              label="Ticket level"
              value={level}
              onChange={(e) => {
                changeLevel(e.target.value)
                touch('level')
              }}
              onBlur={() => touch('level')}
              error={shown('level', fieldErrors.level)}
              options={[
                { value: 'parent', label: 'Parent ticket' },
                { value: 'child', label: 'Child ticket' },
              ]}
              placeholder={level ? undefined : 'Parent or child?'}
              // From a spec's +, it's that spec's child
              disabled={Boolean(underSpec) || Boolean(known)}
              title={
                underSpec
                  ? `It goes under parent #${underSpec.ticket}`
                  : known
                    ? 'Move it under a parent in its Edit form or in the table'
                    : 'A parent is a spec ticket that others go under; a child sits under a spec'
              }
              data-autofocus
              aria-required
            />
          ) : (
            ticketField
          )}
          {repos.length > 0 && (
            <SelectField
              label="Repo"
              value={repo}
              onChange={(e) => {
                changeRepo(e.target.value)
                touch('repo')
              }}
              onBlur={() => touch('repo')}
              error={shown('repo', fieldErrors.repo)}
              options={levelRepos}
              placeholder={level ? 'Choose a repo' : 'Choose the ticket level first'}
              disabled={!level}
              aria-required
            />
          )}
          {repos.length > 0 && ticketField}
          {/* SPinno: the type shows from the start, waiting for the repo
              (its choices depend on it) */}
          {/* (A parent ticket has none: Title moves up into its row) */}
          {repos.length > 0 && !isParentLevel && (
            <SelectField
              label="Type"
              value={repo ? shownType : ''}
              onChange={(e) => setType(e.target.value)}
              options={types}
              placeholder={repo ? undefined : 'Choose a repo first'}
              disabled={!repo || Boolean(known)}
              title={known ? 'Change a ticket’s type in its Edit form' : undefined}
              aria-required
            />
          )}
          {/* A child's spec, beside the type; the place stays when there's none.
              Not shown when adding from a spec's +: the title says which spec
              it goes under */}
          {repos.length > 0 &&
            !isParentLevel &&
            (repo && level === 'child' && !underSpec ? (
              <FormField
                label="Parent ticket # (optional)"
                value={parent}
                onChange={(e) => {
                  setParent(ticketDigits(e.target.value))
                  touch('parent')
                }}
                onBlur={() => touch('parent')}
                error={shown('parent', fieldErrors.parent)}
                inputMode="numeric"
                list="details-spec-tickets"
              />
            ) : (
              <div aria-hidden="true" />
            ))}
          <ParentTicketOptions id="details-spec-tickets" tickets={specs} />
          <div className={styles.wide}>
            <FormField
              label="Title / short description"
              value={title}
              onChange={(e) => {
                setTitle(e.target.value)
                touch('title')
              }}
              onBlur={() => touch('title')}
              error={shown('title', fieldErrors.title)}
              maxLength={200}
              aria-required
            />
          </div>
          {/* Under the inputs: the type's steps, so it's clear what it means */}
          {repos.length > 0 && (
            <div className={`${styles.flowSteps} ${styles.wide}`} aria-label="The type's steps">
              <span className={styles.flowTitle}>{isParentLevel ? 'Parent ticket' : 'Steps'}</span>
              {isParentLevel ? (
                <span className={styles.flowEmpty}>
                  No type or steps of its own: add it, then add tickets under it with its +.
                </span>
              ) : steps.length > 0 && repo ? (
                steps.map((s, i) => (
                  <Fragment key={s.key}>
                    {i > 0 && (
                      <span className={styles.flowArrow} aria-hidden="true">
                        →
                      </span>
                    )}
                    <span className={styles.flowStep}>{s.label}</span>
                  </Fragment>
                ))
              ) : (
                <span className={styles.flowEmpty}>They show here once the repo is chosen.</span>
              )}
            </div>
          )}
        </div>
        <FormActions hint={hint} isError={Boolean(error)} compact onCancel={onClose} isBusy={isAdding}>
          {onAdd && !known && (
            // For a parent it's the form's button, so Enter adds it too
            <Button
              type={parentOnly ? 'submit' : 'button'}
              variant={parentOnly ? 'primary' : 'outline'}
              onClick={parentOnly ? undefined : handleAdd}
              disabled={Boolean(missing) || isAdding}
            >
              <Icon name="plus" size={16} />
              {/* A parent takes no time of its own, so no timer to speak of */}
              {isAdding ? 'Adding…' : parentOnly ? 'Add parent ticket' : 'Add without timer'}
            </Button>
          )}
          {!parentOnly && (
            <Button type="submit" variant="primary" disabled={Boolean(missing) || isAdding}>
              <Icon name="play" size={16} />
              Continue with timer
            </Button>
          )}
        </FormActions>
      </form>
    </Modal>
  )
}
