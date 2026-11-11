// Chart and badge colours, shared so a status or task looks the same on
// every screen. They match the palette in styles/globals.css.

export const STATUS_COLORS: Record<string, string> = {
  not_started: '#6b7089',
  ongoing: '#7b61ff',
  waiting: '#f6a75c',
  blocked: '#ef4f8b',
  done: '#3ccf91',
}

export const PHASE_COLORS: Record<string, string> = {
  investigation: '#8f7cf8',
  confirmation: '#ef4f8b',
  planning: '#f6a75c',
  coding: '#5b9dff',
  testing: '#3ccf91',
  review: '#a0a6c0',
}

export const SUBMISSION_COLORS: Record<string, string> = {
  app: '#3ccf91',
  imported: '#5b9dff',
  missing: '#ef4f8b',
}

// For series without a fixed meaning, such as projects.
const SERIES = ['#7b61ff', '#ef4f8b', '#f6a75c', '#5b9dff', '#3ccf91', '#a0a6c0']

export const seriesColor = (i: number) => SERIES[i % SERIES.length]
