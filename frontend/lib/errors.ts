// What went wrong, as a message to show: an Error's message, without the
// server's "invalid check-in: " (or "invalid user: ", ...) prefix, which
// only says which kind of thing was refused.
export function errorMessage(e: unknown): string {
  const message = e instanceof Error ? e.message : String(e)
  return message.replace(/^invalid [a-z-]+: /, '')
}
