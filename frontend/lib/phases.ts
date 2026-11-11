// Short phase names for tight spaces such as table headers and the progress
// steps; the full sheet name shows on hover
const SHORT: Record<string, string> = {
  investigation: 'Spec',
  confirmation: 'Confirm',
  planning: 'Planning',
  coding: 'Coding',
  testing: 'Testing',
  review: 'Review',
}

export const shortPhase = (value: string, label: string) => SHORT[value] ?? label
