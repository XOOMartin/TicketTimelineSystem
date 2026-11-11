// Mirrors the DTOs in internal/interfaces/rest/activity_dto.go
export type ImportResult = {
  found: number
  added: number
}

export type DaySummary = {
  day: string // YYYY-MM-DD
  promptCount: number
}

export type Prompt = {
  id: string
  sessionId: string
  project: string
  branch: string
  text: string
  sentAt: string // ISO 8601
}

export type Improvement = {
  issue: string
  example: string
  better: string
}

export type Review = {
  day: string
  summary: string
  score: number // 1-10
  strengths: string[]
  improvements: Improvement[]
  focus: string
  createdAt: string
}
