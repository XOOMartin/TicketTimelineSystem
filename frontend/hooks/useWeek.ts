import { useState } from 'react'

import { mondayOf, today } from 'lib/dates'

// The week a page shows, by its Monday: this week to start with.
export default function useWeek() {
  return useState(() => mondayOf(today()))
}
