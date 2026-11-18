/** Competition dates are calendar dates in Iran, independent of the viewer's timezone. */
export function tehranDate(now = new Date()): string {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'Asia/Tehran', year: 'numeric', month: '2-digit', day: '2-digit',
  }).formatToParts(now)
  const part = (type: string) => parts.find(p => p.type === type)!.value
  return `${part('year')}-${part('month')}-${part('day')}`
}

export function competitionPeriod(date: string, today = tehranDate()): 'today' | 'upcoming' | 'past' {
  const day = date.slice(0, 10)
  return day === today ? 'today' : day > today ? 'upcoming' : 'past'
}
