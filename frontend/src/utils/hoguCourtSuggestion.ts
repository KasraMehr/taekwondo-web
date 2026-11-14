export interface HoguCourtWeight {
  category: string
  size: string
  matches: number
}

// Keep a size with one pair on one court; spread busy sizes only when
// enough pairs exist. The server remains responsible for exact start times.
export function suggestHoguCourts(
  weights: HoguCourtWeight[],
  courtCount: number,
  pairCounts: Record<string, number>,
  limitHogu: boolean,
): Record<string, number[]> {
  const result: Record<string, number[]> = {}
  if (courtCount < 1) return result

  const loads = Array.from({ length: courtCount }, () => 0)
  const bySize = new Map<string, HoguCourtWeight[]>()
  for (const weight of weights) {
    const group = bySize.get(weight.size) ?? []
    group.push(weight)
    bySize.set(weight.size, group)
  }

  const groups = [...bySize.entries()].sort((a, b) =>
    b[1].reduce((total, weight) => total + weight.matches, 0) -
    a[1].reduce((total, weight) => total + weight.matches, 0),
  )
  for (const [size, group] of groups) {
    const pairs = limitHogu ? Math.min(courtCount, pairCounts[size] ?? 0) : courtCount
    if (pairs < 1) continue
    const preferred = Array.from({ length: courtCount }, (_, index) => index + 1)
      .sort((a, b) => loads[a - 1] - loads[b - 1] || a - b)
      .slice(0, pairs)

    for (const weight of [...group].sort((a, b) => b.matches - a.matches || a.category.localeCompare(b.category))) {
      const available = [...preferred].sort((a, b) => loads[a - 1] - loads[b - 1] || a - b)
      const selected = weight.matches >= 16 && available.length > 1 ? available.slice(0, 2) : available.slice(0, 1)
      result[weight.category] = selected
      for (const court of selected) loads[court - 1] += weight.matches / selected.length
    }
  }
  return result
}
