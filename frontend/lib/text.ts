// A count with its word, plural unless it's one: "1 ticket", "3 tickets".
// Words with another plural pass it: plural(2, 'person', 'people').
export function plural(count: number, word: string, many = `${word}s`) {
  return `${count} ${count === 1 ? word : many}`
}
