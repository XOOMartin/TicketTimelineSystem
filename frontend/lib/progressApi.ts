import { ApiError, request } from 'lib/http'
import { DaySummary, ImportResult, Prompt, Review } from 'types/progress'

// The Claude Code activity routes
const BASE_URL = '/activity'

export function importPrompts(): Promise<ImportResult> {
  return request<ImportResult>(`${BASE_URL}/import`, { method: 'POST' })
}

export function fetchDays(): Promise<DaySummary[]> {
  return request<DaySummary[]>(`${BASE_URL}/days`)
}

export function fetchPrompts(day: string): Promise<Prompt[]> {
  return request<Prompt[]>(`${BASE_URL}/days/${encodeURIComponent(day)}/prompts`)
}

// Resolves to null when the day hasn't been analyzed yet.
export async function fetchReview(day: string): Promise<Review | null> {
  try {
    return await request<Review>(`${BASE_URL}/days/${encodeURIComponent(day)}/review`)
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return null
    throw e
  }
}

export function generateReview(day: string): Promise<Review> {
  return request<Review>(`${BASE_URL}/days/${encodeURIComponent(day)}/review`, { method: 'POST' })
}
