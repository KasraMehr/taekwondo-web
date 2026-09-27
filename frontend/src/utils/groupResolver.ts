// utils/groupResolver.ts
import type { Athlete, LeagueGroup } from '../types'

export type AthleteRef = Pick<Athlete, 'id' | 'club'>

export interface GroupResolver {
    groupIdOf(a: AthleteRef): string | null
    groupOf(a: AthleteRef): LeagueGroup | null
    isExplicit(a: AthleteRef): boolean
    /** گروهی که فقط بر اساس باشگاه به آن تعلق دارد (پیش از اعمال استثناها) */
    clubGroupIdOf(a: AthleteRef): string | null
}

export function buildGroupResolver(groups: LeagueGroup[]): GroupResolver {
    const explicit = new Map<string, string>()
    const excluded = new Set<string>()
    const clubToGroup = new Map<string, string>()

    for (const g of groups) {
        for (const id of g.athleteIds ?? []) explicit.set(id, g.id)
        for (const id of g.excludedAthleteIds ?? []) excluded.add(`${g.id}::${id}`)
        for (const club of g.clubs ?? []) clubToGroup.set(club, g.id)
    }

    const byId = new Map(groups.map((g) => [g.id, g]))

    const clubGroupIdOf = (a: AthleteRef) => (a.club ? clubToGroup.get(a.club) ?? null : null)

    const groupIdOf = (a: AthleteRef): string | null => {
        const ex = explicit.get(a.id)
        if (ex) return ex
        const viaClub = clubGroupIdOf(a)
        if (viaClub && !excluded.has(`${viaClub}::${a.id}`)) return viaClub
        return null
    }

    return {
        groupIdOf,
        clubGroupIdOf,
        groupOf: (a) => {
            const id = groupIdOf(a)
            return id ? byId.get(id) ?? null : null
        },
        isExplicit: (a) => explicit.has(a.id),
    }
}
