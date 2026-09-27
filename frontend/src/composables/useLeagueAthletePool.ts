// composables/useLeagueAthletePool.ts
import { computed, type Ref } from 'vue'
import type { Athlete } from '../types'
import { useTournamentStore } from '../stores/tournament'
import { useTeamTournamentStore } from '../stores/teamTournament'
import { useLeagueStore } from '../stores/league'

export function useLeagueAthletePool(leagueId: Ref<string>) {
    const tournamentStore = useTournamentStore()
    const teamStore = useTeamTournamentStore()
    const leagueStore = useLeagueStore()

    return computed<Athlete[]>(() => {
        const out = new Map<string, Athlete>()

        const push = (list: Athlete[] = []) => {
            for (const a of list) {
                const key = a.sourceLeagueAthleteId ?? a.id
                if (!out.has(key)) out.set(key, a)
            }
        }

        const league = leagueStore.leagues.find(item => item.id === leagueId.value)
        push((league?.athletes ?? []).filter(a => a.isActive !== false).map(a => ({
            id: a.id,
            sourceLeagueAthleteId: a.id,
            name: a.name,
            club: a.club ?? 'آزاد',
            weightCategory: a.weightCategory ?? '',
            ranking: (a as typeof a & { ranking?: number | null }).ranking ?? undefined,
        })))

        for (const t of tournamentStore.tournaments) {
            if (t.leagueId === leagueId.value) push(t.athletes)
        }
        for (const t of teamStore.items) {
            if (t.leagueId === leagueId.value) push(t.athletes)
        }

        return [...out.values()].sort(
            (a, b) =>
                (a.ranking ?? Number.MAX_SAFE_INTEGER) - (b.ranking ?? Number.MAX_SAFE_INTEGER) ||
                a.name.localeCompare(b.name, 'fa')
        )
    })
}
