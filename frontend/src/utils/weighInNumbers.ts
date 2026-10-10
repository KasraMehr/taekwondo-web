import type { Athlete } from '../types'

export function weighInNumbers(athletes: Pick<Athlete, 'id' | 'number' | 'weightCategory'>[], weights: string[]) {
  const indexes = new Map(weights.map((weight, index) => [weight, index]))
  const weightValue = (weight: string) => Number.parseFloat(weight.replace(/[^0-9.]/g, '')) || 0
  const ordered = [...athletes].sort((a, b) =>
    (indexes.get(a.weightCategory) ?? 999) - (indexes.get(b.weightCategory) ?? 999)
    || weightValue(a.weightCategory) - weightValue(b.weightCategory)
    || (a.number ?? 0) - (b.number ?? 0)
    || a.id.localeCompare(b.id))
  return new Map(ordered.map((athlete, index) => [athlete.id, index + 1]))
}
