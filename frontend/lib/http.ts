// Calls to the Go server, proxied by the rewrite in next.config.js. The
// session cookie is same-origin, so fetch sends it automatically. Every API
// module goes through request().
export const BASE_URL = '/api'

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isForm = init?.body instanceof FormData
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: isForm ? init?.headers : { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (res.status === 204) {
    return undefined as T
  }
  // The Go server sends {"error": "..."} on failure
  const body = await res.json().catch(() => null)
  if (!res.ok) {
    throw new ApiError(res.status, body?.error ?? `Request failed: ${res.status} ${res.statusText}`)
  }
  return body as T
}

// A request with a JSON body
export const json = (method: string, body: unknown): RequestInit => ({ method, body: JSON.stringify(body) })
