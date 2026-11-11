import { Movie, MovieInput } from 'types/movie'

// Proxied to the Go server by the rewrites in next.config.js
const BASE_URL = '/api/movies'

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status} ${res.statusText}`)
  }
  // The Go handlers return an empty body when an id isn't found
  const text = await res.text()
  if (!text) {
    throw new Error('Movie not found')
  }
  return JSON.parse(text) as T
}

export async function fetchMovies(): Promise<Movie[]> {
  // Go encodes a nil slice as `null`
  const movies = await request<Movie[] | null>(BASE_URL)
  return movies ?? []
}

export function createMovie(input: MovieInput): Promise<Movie> {
  return request<Movie>(BASE_URL, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateMovie(id: string, input: MovieInput): Promise<Movie> {
  return request<Movie>(`${BASE_URL}/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function deleteMovie(id: string): Promise<void> {
  await request<Movie[] | null>(`${BASE_URL}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}
