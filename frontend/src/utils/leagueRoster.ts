import type { Athlete, League } from '../types'

/** Builds a tournament snapshot from the league roster, which is the source of truth. */
export function tournamentSeedsForGroup(league: League, groupId: string | null): Omit<Athlete, 'id'>[] {
  const clubs = new Map((league.clubs ?? []).map(c => [c.id, c]))
  return (league.athletes ?? [])
    .filter(a => a.isActive !== false)
    .filter(a => {
      const inherited = a.clubId ? clubs.get(a.clubId)?.groupId ?? null : null
      return groupId == null || a.groupId === groupId || (a.groupId == null && inherited === groupId)
    })
    .filter(a => Boolean(a.weightCategory))
    .map(a => ({
      sourceLeagueAthleteId: a.id,
      name: a.name.trim(),
      club: a.clubId ? clubs.get(a.clubId)?.name ?? a.club ?? '' : a.club ?? '',
      weightCategory: a.weightCategory!,
      ranking: undefined,
    }))
    .sort((a, b) => a.weightCategory.localeCompare(b.weightCategory) || a.name.localeCompare(b.name, 'fa'))
}
