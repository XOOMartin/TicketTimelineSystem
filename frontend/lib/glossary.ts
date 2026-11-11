// The Help page's definitions of terms, grouped as the app uses them. Kept
// here as plain data so wording can change without touching the page.

export type Term = {
  term: string
  // How it's also shown or written, e.g. a phase's full sheet name
  aka?: string
  definition: string
  // A status shown as its badge, or a phase with its chart colour
  status?: string
  phase?: string
}

export type TermGroup = { id: string; title: string; intro?: string; terms: Term[] }

export const GLOSSARY: TermGroup[] = [
  {
    id: 'tickets',
    title: 'Tickets',
    terms: [
      {
        term: 'Ticket',
        definition:
          'One piece of work you log time on, named by its project and number, e.g. BlackBox#3221 or SPinno/web#120. Each row of My timeline is one ticket.',
      },
      {
        term: 'Project',
        definition: 'The product a ticket belongs to: BlackBox or SPinno.',
      },
      {
        term: 'Repo',
        aka: 'Spec · Backend · Web',
        definition:
          'SPinno only: the repository a ticket belongs to, Spinno next Spec, Spinno next (backend) or Spinno next web (frontend). Each repo numbers its own tickets, so Backend #120 and Web #120 are different tickets.',
      },
      {
        term: 'Ticket #',
        definition: 'The ticket’s number, digits only. Picking a suggestion such as “SPinno/web#120” keeps just 120.',
      },
      {
        term: 'Title',
        definition: 'A short name for the ticket, shown in bold in the table.',
      },
      {
        term: 'Note',
        definition:
          'A short comment on the ticket, shown on the quiet line under its title. Required when the status is Blocked, to say what is blocking it.',
      },
    ],
  },
  {
    id: 'phases',
    title: 'Phases',
    intro:
      'The kinds of work on a ticket, in the order a ticket usually moves through them. The table uses the short name; the full name matches the team’s progress sheet.',
    terms: [
      {
        term: 'Spec',
        aka: 'Check Spec / Investigation',
        phase: 'investigation',
        definition: 'Reading the spec and investigating what the ticket needs.',
      },
      {
        term: 'Confirm',
        aka: 'Confirmation / Review to SE/MGRs',
        phase: 'confirmation',
        definition:
          'Confirming the approach with the SE or managers. Its time is typed in rather than timed, since it depends on them.',
      },
      {
        term: 'Planning',
        aka: 'Development / Spec Changes (Planning)',
        phase: 'planning',
        definition: 'Planning the development, or changing the spec. A revision from the final review starts here.',
      },
      {
        term: 'Coding',
        aka: 'Coding (Implementation)',
        phase: 'coding',
        definition: 'Writing the code.',
      },
      {
        term: 'Testing',
        phase: 'testing',
        definition: 'Testing the change.',
      },
      {
        term: 'Review',
        aka: 'Review After Implementation',
        phase: 'review',
        definition:
          'The review of the finished work. Its time is typed in rather than timed, since it depends on the reviewer.',
      },
    ],
  },
  {
    id: 'statuses',
    title: 'Statuses',
    terms: [
      {
        term: 'Not started',
        status: 'not_started',
        definition: 'No work yet. A ticket saved as Not started but with time logged shows as Ongoing in the table.',
      },
      { term: 'Ongoing', status: 'ongoing', definition: 'Being worked on.' },
      {
        term: 'Waiting',
        status: 'waiting',
        definition: 'Waiting on someone else, e.g. after choosing “Still waiting” on the Confirm step.',
      },
      {
        term: 'Blocked',
        status: 'blocked',
        definition: 'Can’t move on. Needs a note saying what is blocking it.',
      },
      {
        term: 'Done',
        status: 'done',
        definition:
          'Finished. Set when a review is Approved, which also ends the ticket today. A Done ticket can’t get a new revision until its status changes.',
      },
    ],
  },
  {
    id: 'rounds',
    title: 'Rounds and steps',
    terms: [
      {
        term: 'Round',
        definition:
          'One pass of work on a ticket. The first is the Original; each review that asks for changes adds a revision.',
      },
      { term: 'Original', definition: 'The first round of work on a ticket.' },
      {
        term: 'Revision',
        aka: 'Revision 1, 2… (formerly Review-back)',
        definition:
          'Another round of work after a review asked for changes: Confirm’s starts at Spec, the final Review’s at Planning. It shows in the ticket’s breakdown straight away, and the Revisions column counts it.',
      },
      {
        term: 'Breakdown',
        definition:
          'The lines under a ticket, opened with ▾ or by clicking its row: one per round, and for a spec, the tickets under it. They add up to the ticket’s row.',
      },
      {
        term: 'Progress steps',
        definition:
          'The six phases as steps, opened with ▷. Each step is timed, typed in or marked “Done, no time”, then moves on to the next; there’s no skipping ahead. Previous goes back.',
      },
      {
        term: 'Outcome',
        definition:
          'How a Confirm or Review step ended. Confirm: No changes (on to Planning), Changes requested (starts the next revision, from Spec) or Still waiting. Review: Approved (Done) or Changes requested (starts the next revision, from Planning).',
      },
    ],
  },
  {
    id: 'types',
    title: 'Ticket types (SPinno)',
    intro:
      'Every SPinno child ticket has a type, chosen when it’s added; it decides the ticket’s steps. Each step’s time goes in one of the six phase columns. A parent ticket has no type, steps or time of its own: its total is the time on the tickets under it.',
    terms: [
      {
        term: 'Investigation',
        aka: 'Spec tickets',
        definition:
          'Draft issue → Leads review → Cho-san review. If Cho-san asks for changes, a revision: Revise issue → Cho-san review.',
      },
      {
        term: 'Integration Testing',
        aka: 'Spec tickets',
        definition:
          'Draft issue → Leads review → Test on staging → Cho-san reviews results. If Cho-san asks for changes, a revision: Retest on staging → Cho-san reviews results.',
      },
      {
        term: 'Implementation',
        aka: 'Backend and Web tickets',
        definition:
          'Draft issue → Leads review → Code → Test → Push & open PR → PR review (the leads, then Cho-san; its time goes in Review) → Close PR & issue. If the PR review asks for changes, a revision: Revise code → Test → Cho-san review → Close PR & issue.',
      },
      {
        term: 'Review',
        aka: 'Backend and Web tickets',
        definition:
          'Draft issue → Leads review → Cho-san’s review, marked done when Cho-san finishes. It has no revisions.',
      },
      {
        term: 'Work step',
        aka: 'Draft, Code, Test…',
        definition:
          'Your own work: time it with Start and End, or type the time it took. Its time goes in the step’s column (Draft issue in Spec, Code in Coding, Test in Testing).',
      },
      {
        term: 'Review step',
        aka: 'Leads, Cho-san',
        definition:
          'Someone else’s review: type how long it took if you like (the leads’ review of the issue goes in Confirm; the PR review and Cho-san’s reviews go in Review), then Approved or Changes requested. Either asking for changes starts a revision: the leads’ redoes the steps from Revise issue on; Cho-san’s (or the PR review’s) follows the type’s revision steps.',
      },
      {
        term: 'Action step',
        aka: 'Push & open PR, Close PR & issue',
        definition: 'A quick thing to do: one click on Mark as done, no time.',
      },
      {
        term: 'Changing the type',
        definition: 'In the Edit form. The ticket’s steps start over in the new type’s, where its time puts it.',
      },
    ],
  },
  {
    id: 'specs',
    title: 'Specs and child tickets (SPinno)',
    terms: [
      {
        term: 'Spec ticket',
        aka: 'Parent',
        definition:
          'A SPinno ticket in the Spec repo: an Investigation or Integration Testing. Backend, Web and smaller Spec tickets can sit under it, and its row shows its own time plus theirs.',
      },
      {
        term: 'Child ticket',
        definition: 'A ticket under a spec. It keeps its own hours, rounds, status and steps.',
      },
      {
        term: '+ on a spec',
        definition: 'Adds a ticket under that spec, with the parent already filled in.',
      },
      {
        term: 'Parent ticket #',
        aka: '↳ Parent #45 · No parent',
        definition:
          'The parent ticket a child sits under. Optional, and can be set or changed later. It must already exist. Only a parent can have tickets under it, and a parent can’t go under another parent: two levels, parent then child.',
      },
      {
        term: 'Stand-alone',
        aka: 'Amber chip, in place of the red Parent chip',
        definition:
          'A child spec made a parent by dragging it out of its parent (or by dropping another child on it). It works like a parent — tickets can go under it — but can be dragged back under a parent to be a child again. Only the table says Stand-alone: exports count it as a parent.',
      },
      {
        term: 'Moving a ticket',
        definition:
          'Put a child under a different parent by double-clicking its parent chip, in the Edit form, or by dragging its row onto the parent. Its time moves with it. Dragging a child out of its parent takes it out, still a child. Dropping a child onto a child spec under no parent, or dragging that child spec out, makes it a parent; dragging that parent out again, or onto a parent, makes it a child again (with nothing under it). A parent added in Input timeline stays a parent.',
      },
      {
        term: 'Issues',
        aka: 'N issues',
        definition: 'How many tickets sit under a spec, counting only tickets that exist.',
      },
      {
        term: 'Other issues',
        definition:
          'A line in a spec’s breakdown with the time of its tickets that aren’t in the table now (worked in another week, or hidden by the filter), so the breakdown still adds up.',
      },
    ],
  },
  {
    id: 'time',
    title: 'Time',
    terms: [
      {
        term: 'Timer',
        definition:
          'Times a phase while you work, from ▷. It ends and records its time when you click End or close the site (closing asks first); a reload carries on. Only one runs at a time: starting another asks first. Under a minute records nothing.',
      },
      {
        term: 'Check-in',
        definition:
          'One entry of time on a phase of a ticket, from a timer or typed in. Each is at least 1 minute and at most 8 hours (a working day); a timer left running longer is logged in 8-hour pieces. My check-ins lists them.',
      },
      {
        term: 'Add without timer',
        aka: 'Input timeline',
        definition:
          'Adds a ticket to My timeline with no time yet, as Not started, to time later. Time logged later takes it over. A parent ticket’s button says Add parent ticket.',
      },
      {
        term: 'Continue with timer',
        aka: 'Input timeline',
        definition:
          'Opens the ticket’s progress steps straight away, to time the step you’re on (▷ Start) or type the time it took.',
      },
      {
        term: 'Total',
        definition:
          'All the time logged on a ticket. For a spec it includes the tickets under it, with the split shown below, e.g. “1h own · from 2 issues”.',
      },
      {
        term: 'Own time',
        definition: 'A ticket’s time without the tickets under it.',
      },
      { term: 'Start', definition: 'The day work on the ticket started, or the start date set on it.' },
      { term: 'End', definition: 'The day the ticket was finished; “—” while it is open.' },
      {
        term: 'Time',
        aka: 'Started at',
        definition: 'The time of day the ticket’s first time was logged, in your time zone. The Start column has the day.',
      },
      {
        term: 'Week',
        definition:
          'My timeline shows one week, Monday to Sunday; the arrows at the top move between weeks. A ticket shows in the weeks it was worked on.',
      },
    ],
  },
  {
    id: 'table',
    title: 'In the table',
    terms: [
      {
        term: 'Double-click to edit',
        definition:
          'Double-click an hours cell, the title, the status or the parent chip to change it in place. Enter saves, Esc cancels.',
      },
      { term: '▷ Start', definition: 'Opens a ticket’s progress steps and its timer.' },
      {
        term: '+',
        definition: 'On a ticket, adds its next revision. On a spec, adds a ticket under it.',
      },
      { term: '✎ Edit', definition: 'Opens the ticket’s details, rounds and times in a form.' },
      {
        term: 'Delete',
        definition:
          'Removes a ticket, or one round, after asking. Deleting the last round with time deletes the ticket.',
      },
      {
        term: 'Sort and filter',
        definition:
          'Newest added comes first by default. The status filter, date filter and sort menu sit next to the search box. Pick a date to see only the tickets on the go that day (from their Start to their End, or today if still open); × clears it. Rows per page, under the table, shows 5, 10, 15 or 20 tickets to a page and is remembered in this browser.',
      },
    ],
  },
]
