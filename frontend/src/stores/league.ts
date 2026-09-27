// فایل جدید: stores/league.ts

import { defineStore } from "pinia"
import { ref, computed } from "vue"
import type {
    League, Season, Stage, Week,
    ScoringConfig, PlayerPointsRecord, TeamPointsRecord,
    TeamRankingEntry, PlayerRankingEntry,
    Athlete, Tournament, Match, TournamentFormat,
    LeagueClub, LeagueGroup, GroupTeamRankings, LeagueAthlete, AthleteDraft,
} from "../types"
import { useTournamentStore } from "./tournament"
import type { AgeCategory, Gender } from "../data/categories"
import * as XLSX from "xlsx"
import type { LeagueBundle } from "../types"
import {AthleteRef, buildGroupResolver} from "../utils/groupResolver.ts";
import {normalizeText} from "../utils/athleteImport.ts";
import { tournamentSeedsForGroup } from "../utils/leagueRoster";

const BUNDLE_VERSION = 1

const STORAGE_KEY = "tkd_leagues"

const GROUP_LETTERS = ["A", "B", "C", "D", "E", "F", "G", "H"] as const

export function normalizeClubName(name?: string | null): string {
    return (name ?? "").trim().replace(/\s+/g, " ")
}

export function makeGroups(count: number, names?: string[]): {
    id: `${string}-${string}-${string}-${string}-${string}`;
    name: string;
    order: number;
    clubs: any[]
}[] {
    return Array.from({ length: Math.max(0, count) }, (_, i) => ({
        id: crypto.randomUUID(),
        name: normalizeClubName(names?.[i]) || `گروه ${GROUP_LETTERS[i] ?? i + 1}`,
        order: i + 1,
        clubs: [],
    }))
}

/**
 * سازگاری با داده‌های قبلی localStorage و باندل‌های ایمپورت‌شده.
 * منبع حقیقتِ عضویت، آرایهٔ group.clubs است؛ club.groupId از آن ساخته می‌شود.
 */
function migrateLeague(league: League): League {
    if (!Array.isArray(league.clubs)) league.clubs = []
    if (!Array.isArray(league.season.groups)) league.season.groups = []

    const owner = new Map<string, string>()
    league.season.groups.forEach((g, i) => {
        if (!g.id) g.id = crypto.randomUUID()
        if (typeof g.order !== "number") g.order = i + 1
        g.clubs = Array.from(
            new Set((g.clubs ?? []).map(normalizeClubName).filter(Boolean))
        )
        for (const c of g.clubs) owner.set(c, g.id)
    })

    for (const c of league.clubs) {
        if (!c.id) c.id = crypto.randomUUID()
        c.name = normalizeClubName(c.name)
        c.groupId = owner.get(c.name) ?? null
        if (c.active === undefined) c.active = true
        if (!c.createdAt) c.createdAt = league.createdAt ?? new Date().toISOString()
    }

    return league
}

const deepClone = <T,>(v: T): T => JSON.parse(JSON.stringify(v)) as T

/** کلید یکتا برای تشخیص تکراری: نام نرمال‌شده + باشگاه + جنسیت */
function athleteKey(a: Pick<LeagueAthlete, 'name' | 'clubId' | 'gender'>): string {
    return [normalizeText(a.name), a.clubId ?? '', a.gender ?? ''].join('|')
}

export const useLeagueStore = defineStore("league", () => {
    const leagues = ref<League[]>(
        (JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]") as League[]).map(migrateLeague)
    )
    const currentLeagueId = ref<string | null>(null)
    const webStandings = ref<Record<string, { players: PlayerRankingEntry[]; teams: TeamRankingEntry[] }>>({})

    function replaceFromBackend(items: Array<{ league: any; standings: any }>) {
        leagues.value = items.map(({ league: raw, standings }) => {
            webStandings.value[raw.id] = {
                players: (standings?.players ?? []).map((row: any) => ({
                    rank: row.rank, athleteId: row.id, name: row.name, club: row.clubName,
                    weightCategory: row.weightCategory, totalPoints: row.totalPoints,
                    gold: row.gold, silver: row.silver, bronze: row.bronze,
                    weighInPoints: row.weighInPoints, winPoints: row.winPoints,
                    roundDiff: row.roundDiff, wins2_0: row.wins2_0, wins2_1: row.wins2_1,
                    losses0_2: row.losses0_2, losses1_2: row.losses1_2,
                    totalWins: row.totalWins, totalLosses: row.totalLosses,
                    groupId: row.groupId, groupName: row.groupName,
                })),
                teams: (standings?.teams ?? []).map((row: any) => ({
                    rank: row.rank, club: row.name, totalPoints: row.totalPoints,
                    gold: row.gold, silver: row.silver, bronze: row.bronze,
                    weighInPoints: row.weighInPoints, winPoints: row.winPoints,
                    roundDiff: row.roundDiff, wins2_0: row.wins2_0, wins2_1: row.wins2_1,
                    losses0_2: row.losses0_2, losses1_2: row.losses1_2,
                    totalWins: row.totalWins, totalLosses: row.totalLosses,
                    groupId: row.groupId, groupName: row.groupName,
                    athleteCount: (raw.athletes ?? []).filter((a: any) => a.teamId === row.id).length,
                })),
            }
            const groups = (raw.groups ?? []).map((g: any) => ({
                id: g.id, name: g.name, order: g.position,
                clubs: (raw.teams ?? []).filter((t: any) => t.groupId === g.id).map((t: any) => t.clubName),
                athleteIds: (raw.athletes ?? []).filter((a: any) => a.active && a.groupId === g.id).map((a: any) => a.id),
            }))
            return {
                id: raw.id, name: raw.name, gender: raw.gender, ageCategory: raw.ageCategory,
                format: 'grandPrix', createdAt: raw.createdAt, updatedAt: raw.updatedAt,
                promotionRules: { teamsToPromote: raw.settings?.teamsToPromote ?? 2, autoPromote: raw.settings?.autoPromote ?? true },
                clubs: (raw.teams ?? []).map((t: any) => ({ id: t.clubId, teamId: t.id, name: t.clubName, groupId: t.groupId, active: t.active, createdAt: raw.createdAt })),
                athletes: (raw.athletes ?? []).map((a: any) => {
                    const team = (raw.teams ?? []).find((t: any) => t.id === a.teamId)
                    return { id: a.id, profileId: a.profileId, teamId: a.teamId, coachName: a.coachName, ranking: a.ranking, name: a.name, nationalId: null, clubId: team?.clubId ?? null, club: a.clubName || null, isActive: a.active, weight: null, weightCategory: a.weightCategory, gender: raw.gender, birthDate: null, birthYear: null, beltDegree: null, memberCode: null, groupId: a.groupId, createdAt: a.createdAt, updatedAt: a.updatedAt }
                }),
                season: {
                    id: raw.id, name: 'فصل جاری', groups,
                    scoringConfig: { ...raw.settings?.scoring, keepPlayerRankingBetweenStages: false },
                    stages: (raw.stages ?? []).map((s: any) => ({ id: s.id, name: s.name, order: s.position, completed: s.status === 'completed', promotedClubs: s.promotedTeamIds ?? [], playerPoints: {}, teamPoints: {}, weeks: (s.weeks ?? []).map((w: any) => ({ id: w.id, name: w.name, order: w.position, dateRange: w.dateRange, locked: w.locked })) })),
                    seasonPlayerPoints: {}, seasonTeamPoints: {}, completed: raw.status === 'completed',
                },
            } as League
        })
        save()
    }

    const save = () =>
        localStorage.setItem(STORAGE_KEY, JSON.stringify(leagues.value))

    const touch = (league: League) => { league.updatedAt = new Date().toISOString() }

    const findLeague = (id: string) => leagues.value.find(l => l.id === id) ?? null

    const currentLeague = computed(() => {
        if (!currentLeagueId.value) return null
        return leagues.value.find(l => l.id === currentLeagueId.value) || null
    })

    function selectLeague(id: string) {
        currentLeagueId.value = id
    }

    function clearSelectedLeague() {
        currentLeagueId.value = null
    }

    // ─── ساخت لیگ ───

    function createLeague(config: {
        name: string
        gender: Gender
        ageCategory: AgeCategory
        seasonName: string
        stageCount: number
        weeksPerStage: number
        format?: TournamentFormat
        teamDefaults?: { slotCount: 8 | 10; blindLineup: boolean }
        groupCount?: number
        groupNames?: string[]
        initialClubs?: string[]
    }) {
        const stages: Stage[] = []
        for (let i = 0; i < config.stageCount; i++) {
            const weeks: Week[] = []
            for (let j = 0; j < config.weeksPerStage; j++) {
                weeks.push({
                    id: crypto.randomUUID(),
                    name: `هفته ${j + 1}`,
                    order: j + 1,
                    locked: false,
                })
            }

            stages.push({
                id: crypto.randomUUID(),
                name: `مرحله ${i + 1}`,
                order: i + 1,
                weeks,
                playerPoints: {},
                teamPoints: {},
                completed: false,
                promotedClubs: [],
            })
        }

        const season: Season = {
            id: crypto.randomUUID(),
            name: config.seasonName,
            scoringConfig: {
                gold: 10,
                silver: 6,
                bronze: 3,
                weighInPoint: 1,
                winPoint: 2,
                countWeighIn: true,
                countWin: true,
                keepPlayerRankingBetweenStages: false,
            },
            stages,
            seasonPlayerPoints: {},
            seasonTeamPoints: {},
            groups: makeGroups(config.groupCount ?? 0, config.groupNames),
            completed: false,
        }

        const format: TournamentFormat = config.format ?? "grandPrix"
        const now = new Date().toISOString()

        const league: League = {
            athletes: [],
            id: crypto.randomUUID(),
            name: config.name,
            gender: config.gender,
            ageCategory: config.ageCategory,
            season,
            clubs: [],
            promotionRules: {
                teamsToPromote: 2,
                autoPromote: true,
            },
            format,
            // فقط برای لیگ تیمی معنا دارد
            ...(format === "team" && config.teamDefaults
                ? { teamDefaults: config.teamDefaults }
                : {}),
            createdAt: now,
            updatedAt: now
        }

        leagues.value.push(league)
        currentLeagueId.value = league.id
        save()

        // باشگاه‌های اولیه؛ در صورت وجود گروه، به‌ترتیب پخش می‌شوند
        for (const name of config.initialClubs ?? []) {
            try { addClub(league.id, { name }) } catch { /* تکراری یا خالی */ }
        }
        if (season.groups.length) distributeClubs(league.id)

        return league
    }

    // ─── مدیریت هفته ───

    function addWeek(leagueId: string, stageOrder: number, week: Omit<Week, "id" | "order">): Week | null {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        if (!stage) return null

        const maxOrder = stage.weeks.reduce((max, w) => Math.max(max, w.order), 0)
        const newWeek: Week = {
            id: crypto.randomUUID(),
            ...week,
            order: maxOrder + 1,
        }
        stage.weeks.push(newWeek)
        save()
        return newWeek
    }

    function createTournamentForWeek(
        leagueId: string,
        stageOrder: number,
        weekId: string,
        tournamentConfig: { name: string; courts: number }
    ): Tournament | null {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        const week = stage?.weeks.find((w) => w.id === weekId)
        if (!league || !stage || !week) return null
        if (week.tournamentId) return null   // جلوگیری از ساخت تکراری روی یک هفته

        const tournamentStore = useTournamentStore()

        const newTournament = tournamentStore.createTournament(
            tournamentConfig.name,
            todayLocal(),
            tournamentConfig.courts,
            league.gender,
            league.ageCategory,
            league.format ?? "grandPrix",
            { leagueId, stageOrder, weekId },
        )

        week.tournamentId = newTournament.id
        autoCarryAthletesToTournament(leagueId, stageOrder, weekId)
        save()

        return newTournament
    }

    function linkWeekToTournament(leagueId: string, stageOrder: number, weekId: string, tournamentId: string) {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        const week = stage?.weeks.find((w) => w.id === weekId)
        if (!week) return
        week.tournamentId = tournamentId
        save()
    }

    function lockWeek(leagueId: string, stageOrder: number, weekId: string, locked = true) {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        const week = stage?.weeks.find((w) => w.id === weekId)
        if (!week) return
        week.locked = locked
        save()
    }

    // ─── هسته محاسبه امتیاز ───

    function getOrCreatePlayerRecord(stage: Stage, athlete: Athlete): PlayerPointsRecord {
        const stableId = athlete.sourceLeagueAthleteId ?? athlete.id
        const existing = stage.playerPoints[stableId]
        if (existing) return existing

        const record: PlayerPointsRecord = {
            athleteId: stableId,
            name: athlete.name,
            club: athlete.club,
            weightCategory: athlete.weightCategory,
            totalPoints: 0,
            gold: 0,
            silver: 0,
            bronze: 0,
            weighInPoints: 0,
            winPoints: 0,
            roundDiff: 0,
            wins2_0: 0,
            wins2_1: 0,
            losses0_2: 0,
            losses1_2: 0,
            totalWins: 0,
            totalLosses: 0,
            ranking: athlete.ranking,
        }
        stage.playerPoints[stableId] = record
        return record
    }

    function savePlayerRecord(stage: Stage, record: PlayerPointsRecord) {
        stage.playerPoints[record.athleteId] = record
    }

    function applyWeighInToStage(stage: Stage, league: League, tournament: Tournament) {
        const config = league.season.scoringConfig
        if (!config.countWeighIn) return

        for (const athlete of tournament.athletes) {
            if (!athlete.weighedIn) continue

            const record = getOrCreatePlayerRecord(stage, athlete)
            record.weighInPoints += config.weighInPoint
            record.totalPoints += config.weighInPoint
        }
    }

    function applyMatchToStage(stage: Stage, league: League, tournament: Tournament, match: Match) {
        if (match.status !== 'completed' || !match.result || !match.winnerId) return
        if (!match.athlete1Id || !match.athlete2Id) return

        const config = league.season.scoringConfig
        const athletes = new Map(tournament.athletes.map((a) => [a.id, a]))

        const a1 = athletes.get(match.athlete1Id)
        const a2 = athletes.get(match.athlete2Id)
        if (!a1 || !a2) return

        const winnerAthlete = match.winnerId === a1.id ? a1 : a2
        const loserAthlete = match.winnerId === a1.id ? a2 : a1

        // ─── محاسبه تفاضل راند ───

        let winnerRounds = 0
        let loserRounds = 0

        const rounds = match.result.rounds ?? []
        for (const round of rounds) {
            if (round.winner === 'blue') {
                if (match.result.winnerCorner === 'blue') winnerRounds++
                else loserRounds++
            } else if (round.winner === 'red') {
                if (match.result.winnerCorner === 'red') winnerRounds++
                else loserRounds++
            }
        }

        let winnerRoundDiff = 0
        let loserRoundDiff = 0

        if (rounds.length > 0) {
            if (winnerRounds === 2 && loserRounds === 0) {
                winnerRoundDiff = 2
                loserRoundDiff = -2
            } else if (winnerRounds === 2 && loserRounds === 1) {
                winnerRoundDiff = 1
                loserRoundDiff = -1
            }
        }

        // ─── رکوردها ───

        const winnerRecord = getOrCreatePlayerRecord(stage, winnerAthlete)
        const loserRecord = getOrCreatePlayerRecord(stage, loserAthlete)

        // ─── امتیاز برد (دلتا مستقیم به totalPoints) ───

        if (config.countWin) {
            winnerRecord.winPoints += config.winPoint
            winnerRecord.totalPoints += config.winPoint
        }

        // ─── آمار برد/باخت و راند ───

        winnerRecord.totalWins++
        winnerRecord.roundDiff += winnerRoundDiff
        if (winnerRoundDiff === 2) winnerRecord.wins2_0++
        else if (winnerRoundDiff === 1) winnerRecord.wins2_1++

        loserRecord.totalLosses++
        loserRecord.roundDiff += loserRoundDiff
        if (loserRoundDiff === -2) loserRecord.losses0_2++
        else if (loserRoundDiff === -1) loserRecord.losses1_2++

        savePlayerRecord(stage, winnerRecord)
        savePlayerRecord(stage, loserRecord)
    }

    function awardMedals(stage: Stage, league: League, tournament: Tournament) {
        const config = league.season.scoringConfig
        const athletes = new Map(tournament.athletes.map((a) => [a.id, a]))

        // گروه‌بندی بر اساس وزن (بدون بازی‌های استراحت)
        const byCategory: Record<string, Match[]> = {}
        for (const match of tournament.matches) {
            if (match.isBye) continue
            if (!byCategory[match.weightCategory]) byCategory[match.weightCategory] = []
            byCategory[match.weightCategory].push(match)
        }

        for (const matches of Object.values(byCategory)) {
            if (matches.length === 0) continue

            const allCompleted = matches.every(
                (m) => m.status === 'completed' && m.winnerId
            )
            if (!allCompleted) continue

            // گراف براکت منبع مطمئن‌تری از شماره راند است: فینال مقصد بعدی ندارد.
            const finals = matches.filter((m) => !m.nextMatchId)
            const finalMatch = finals.length === 1 ? finals[0] : undefined
            if (!finalMatch || !finalMatch.winnerId) continue

            // ─── طلا: برنده فینال ───
            const goldAthlete = athletes.get(finalMatch.winnerId)
            if (goldAthlete) {
                const rec = getOrCreatePlayerRecord(stage, goldAthlete)
                rec.gold++
                rec.totalPoints += config.gold
                savePlayerRecord(stage, rec)
            }

            // ─── نقره: بازنده فینال ───
            const silverLoserId = finalMatch.winnerId === finalMatch.athlete1Id
                ? finalMatch.athlete2Id
                : finalMatch.athlete1Id
            if (silverLoserId) {
                const silverAthlete = athletes.get(silverLoserId)
                if (silverAthlete) {
                    const rec = getOrCreatePlayerRecord(stage, silverAthlete)
                    rec.silver++
                    rec.totalPoints += config.silver
                    savePlayerRecord(stage, rec)
                }
            }

            // ─── برنز: بازنده‌های نیمه‌نهایی (راندِ قبل از فینال) ───
            // اگر نیمه‌نهایی bye بوده باشه، بازنده‌ای نداره و طبیعتاً برنز نمی‌گیره
            const semifinalMatches = matches.filter((m) => m.nextMatchId === finalMatch.id)
            for (const semi of semifinalMatches) {
                if (!semi.winnerId) continue
                const loserId = semi.winnerId === semi.athlete1Id
                    ? semi.athlete2Id
                    : semi.athlete1Id
                if (!loserId) continue

                const bronzeAthlete = athletes.get(loserId)
                if (bronzeAthlete) {
                    const rec = getOrCreatePlayerRecord(stage, bronzeAthlete)
                    rec.bronze++
                    rec.totalPoints += config.bronze
                    savePlayerRecord(stage, rec)
                }
            }
        }
    }

    function computeTeamPoints(playerPoints: Record<string, PlayerPointsRecord>): Record<string, TeamPointsRecord> {
        const teams: Record<string, TeamPointsRecord> = {}

        for (const record of Object.values(playerPoints)) {
            if (!teams[record.club]) {
                teams[record.club] = {
                    club: record.club,
                    totalPoints: 0,
                    gold: 0,
                    silver: 0,
                    bronze: 0,
                    weighInPoints: 0,
                    winPoints: 0,
                    roundDiff: 0,
                    wins2_0: 0,
                    wins2_1: 0,
                    losses0_2: 0,
                    losses1_2: 0,
                    totalWins: 0,
                    totalLosses: 0,
                    athleteCount: 0,
                }
            }

            const team = teams[record.club]
            team.totalPoints += record.totalPoints
            team.gold += record.gold
            team.silver += record.silver
            team.bronze += record.bronze
            team.weighInPoints += record.weighInPoints
            team.winPoints += record.winPoints
            team.roundDiff += record.roundDiff
            team.wins2_0 += record.wins2_0
            team.wins2_1 += record.wins2_1
            team.losses0_2 += record.losses0_2
            team.losses1_2 += record.losses1_2
            team.totalWins += record.totalWins
            team.totalLosses += record.totalLosses
            team.athleteCount++
        }

        return teams
    }

    function mergeRecords<T extends { totalPoints: number }>(a: T | undefined, b: T): T {
        if (!a) return { ...b }
        const merged = { ...a }
        for (const key of Object.keys(b) as (keyof T)[]) {
            if (typeof b[key] === 'number') {
                ;(merged as any)[key] = (a[key] as number) + (b[key] as number)
            }
        }
        return merged
    }

    function recalculateSeasonPoints(league: League) {
        league.season.seasonPlayerPoints = {}
        league.season.seasonTeamPoints = {}

        for (const stage of league.season.stages) {
            for (const [key, record] of Object.entries(stage.playerPoints)) {
                const existing = league.season.seasonPlayerPoints[key]
                league.season.seasonPlayerPoints[key] = mergeRecords(existing, record)
            }
            for (const [key, record] of Object.entries(stage.teamPoints)) {
                const existing = league.season.seasonTeamPoints[key]
                league.season.seasonTeamPoints[key] = mergeRecords(existing, record)
            }
        }
    }

    // ─── محاسبه کامل یک مرحله ───

    function recalculateStagePoints(leagueId: string, stageOrder: number) {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        if (!league || !stage) return

        // ❗❗ **اینجا قبل از هر چیز پاکش کن**
        stage.playerPoints = {}
        stage.teamPoints = {}

        const tournamentStore = useTournamentStore()

        for (const week of stage.weeks) {
            if (!week.tournamentId) continue
            const tournament = tournamentStore.tournaments.find((t) => t.id === week.tournamentId)
            if (!tournament) continue

            applyWeighInToStage(stage, league, tournament)

            // اول امتیاز هر بازی
            for (const match of tournament.matches) {
                applyMatchToStage(stage, league, tournament, match)
            }

            // بعد مدال‌ها (بعد از همه بازی‌ها)
            awardMedals(stage, league, tournament)
        }

        // محاسبه امتیاز تیمی از امتیاز بازیکنان
        stage.teamPoints = computeTeamPoints(stage.playerPoints)

        // به‌روزرسانی امتیاز فصل
        recalculateSeasonPoints(league)
        save()
    }

    // ─── صعود خودکار ───

    function promoteTeams(leagueId: string): Stage | null {
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return null

        const currentStage = league.season.stages[league.season.stages.length - 1]
        if (!currentStage || !currentStage.completed) return null

        const promoted = getPromotionCandidates(leagueId, currentStage.order)

        const newStage: Stage = {
            id: crypto.randomUUID(),
            name: `مرحله ${currentStage.order + 1}`,
            order: currentStage.order + 1,
            weeks: [],
            playerPoints: {},
            teamPoints: {},
            completed: false,
            promotedClubs: promoted,
        }

        league.season.stages.push(newStage)
        save()
        return newStage
    }

    // ─── رتبه‌بندی ───

    function getTeamRankings(leagueId: string, stageOrder?: number): TeamRankingEntry[] {
        if (webStandings.value[leagueId]) return webStandings.value[leagueId].teams
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return []

        if (stageOrder != null) {
            const stage = league.season.stages.find((s) => s.order === stageOrder)
            return stage ? rankTeams(stage.teamPoints) : []
        }
        return rankTeams(league.season.seasonTeamPoints)
    }

    function getPlayerRankings(leagueId: string, stageOrder?: number): PlayerRankingEntry[] {
        if (webStandings.value[leagueId]) return webStandings.value[leagueId].players
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return []

        if (stageOrder != null) {
            const stage = league.season.stages.find((s) => s.order === stageOrder)
            return stage ? rankPlayers(stage.playerPoints) : []
        }
        return rankPlayers(league.season.seasonPlayerPoints)
    }

    function rankPlayers(
        playerPoints: Record<string, PlayerPointsRecord>
    ): PlayerRankingEntry[] {
        return Object.values(playerPoints)
            .sort((a, b) => {
                // اول امتیاز کل
                if (b.totalPoints !== a.totalPoints) return b.totalPoints - a.totalPoints
                // بعد تعداد طلا
                if (b.gold !== a.gold) return b.gold - a.gold
                // بعد نقره
                if (b.silver !== a.silver) return b.silver - a.silver
                // بعد برنز
                if (b.bronze !== a.bronze) return b.bronze - a.bronze
                // بعد تفاضل راند
                if (b.roundDiff !== a.roundDiff) return b.roundDiff - a.roundDiff
                // بعد تعداد برد
                return b.totalWins - a.totalWins
            })
            .map((record, index) => ({
                rank: index + 1,
                athleteId: record.athleteId,
                name: record.name,
                club: record.club,
                weightCategory: record.weightCategory,
                totalPoints: record.totalPoints,
                gold: record.gold,
                silver: record.silver,
                bronze: record.bronze,
                weighInPoints: record.weighInPoints,
                winPoints: record.winPoints,
                roundDiff: record.roundDiff,
                wins2_0: record.wins2_0,
                wins2_1: record.wins2_1,
                losses0_2: record.losses0_2,
                losses1_2: record.losses1_2,
                totalWins: record.totalWins,
                totalLosses: record.totalLosses,
            }))
    }

    function rankTeams(
        teamPoints: Record<string, TeamPointsRecord>
    ): TeamRankingEntry[] {
        return Object.values(teamPoints)
            .sort((a, b) => {
                // اول امتیاز کل
                if (b.totalPoints !== a.totalPoints) return b.totalPoints - a.totalPoints
                // بعد تعداد طلا
                if (b.gold !== a.gold) return b.gold - a.gold
                // بعد نقره
                if (b.silver !== a.silver) return b.silver - a.silver
                // بعد برنز
                if (b.bronze !== a.bronze) return b.bronze - a.bronze
                // بعد تفاضل راند
                if (b.roundDiff !== a.roundDiff) return b.roundDiff - a.roundDiff
                // بعد تعداد برد
                return b.totalWins - a.totalWins
            })
            .map((record, index) => ({
                rank: index + 1,
                club: record.club,
                totalPoints: record.totalPoints,
                gold: record.gold,
                silver: record.silver,
                bronze: record.bronze,
                weighInPoints: record.weighInPoints,
                winPoints: record.winPoints,
                roundDiff: record.roundDiff,
                wins2_0: record.wins2_0,
                wins2_1: record.wins2_1,
                losses0_2: record.losses0_2,
                losses1_2: record.losses1_2,
                totalWins: record.totalWins,
                totalLosses: record.totalLosses,
                athleteCount: record.athleteCount,
            }))
    }

    // ─── تنظیمات امتیاز ───

    function updateScoringConfig(leagueId: string, config: Partial<ScoringConfig>) {
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return
        Object.assign(league.season.scoringConfig, config)
        save()
    }

    function autoCarryAthletesToTournament(
        leagueId: string,
        stageOrder: number,
        newWeekId: string
    ) {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        if (!league || !stage) return

        const tournamentStore = useTournamentStore()

        // پیدا کردن هفته جدید
        const newWeek = stage.weeks.find((w) => w.id === newWeekId)
        if (!newWeek || !newWeek.tournamentId) return

        // پیدا کردن تورنمنت جدید
        const newTournament = tournamentStore.tournaments.find(
            (t) => t.id === newWeek.tournamentId
        )
        if (!newTournament) return

        // روستر خود لیگ منبع حقیقت است. وزن‌کشی هر تورنمنت مستقل شروع می‌شود.
        newTournament.athletes = []
        for (const seed of tournamentSeedsForGroup(league, newTournament.groupId ?? null))
            tournamentStore.addAthlete(newTournament.id, seed)

        tournamentStore.save()
        save()
    }


    // ─── حذف ───

    function deleteLeague(id: string) {
        leagues.value = leagues.value.filter((l) => l.id !== id)
        if (currentLeagueId.value === id) currentLeagueId.value = null
        save()
    }

    function carryAthletesFromPreviousTournament(
        leagueId: string,
        stageOrder: number,
        targetWeekId: string
    ) {
        autoCarryAthletesToTournament(leagueId, stageOrder, targetWeekId)
    }

    function unlinkWeekFromTournament(
        leagueId: string,
        stageOrder: number,
        weekId: string
    ) {
        const league = leagues.value.find((l) => l.id === leagueId)
        const stage = league?.season.stages.find((s) => s.order === stageOrder)
        const week = stage?.weeks.find((w) => w.id === weekId)

        if (week) {
            week.tournamentId = undefined
            save()
        }
    }

// ❗️ رنکینگ کل (همه وزن‌ها با هم) - برای جدول کلی
    function getOverallPlayerRankings(
        leagueId: string,
        stageOrder?: number
    ): PlayerRankingEntry[] {
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return []

        if (stageOrder != null) {
            const stage = league.season.stages.find((s) => s.order === stageOrder)
            return stage ? rankPlayers(stage.playerPoints) : []
        }
        return rankPlayers(league.season.seasonPlayerPoints)
    }

    function todayLocal(): string {
        const d = new Date();
        const pad = (n: number) => String(n).padStart(2, "0");
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    }

    // ─────────────────────── اکسپورت / ایمپورت لیگ ───────────────────────

    /** ارجاع‌های زندهٔ تورنمنت‌های لیگ (قابل ویرایش) */
    function liveLeagueTournaments(league: League): Tournament[] {
        const tournamentStore = useTournamentStore()
        const ids = new Set<string>()

        for (const stage of league.season.stages)
            for (const week of stage.weeks)
                if (week.tournamentId) ids.add(week.tournamentId)

        for (const t of tournamentStore.tournaments)
            if (t.leagueId === league.id) ids.add(t.id)

        return tournamentStore.tournaments.filter(t => ids.has(t.id))
    }

    /** نسخهٔ کلون‌شده برای اکسپورت (رفتار قبلی) */
    function collectLeagueTournaments(league: League): Tournament[] {
        return liveLeagueTournaments(league).map(t => deepClone(t))
    }


    function buildLeagueBundle(leagueId: string): LeagueBundle | null {
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return null

        return {
            kind: "tkd-league-bundle",
            version: BUNDLE_VERSION,
            exportedAt: new Date().toISOString(),
            league: deepClone(league),
            tournaments: collectLeagueTournaments(league),
        }
    }

    const safeFileName = (s: string) => s.replace(/[\\/:*?"<>|]/g, "-")

    /** خروجی JSON: سبک‌ترین و بی‌نقص‌ترین حالت انتقال */
    function exportLeagueToJson(leagueId: string): { error?: string } {
        const bundle = buildLeagueBundle(leagueId)
        if (!bundle) return { error: "لیگ یافت نشد" }

        try {
            const blob = new Blob([JSON.stringify(bundle)], { type: "application/json" })
            const url = URL.createObjectURL(blob)
            const a = document.createElement("a")
            a.href = url
            a.download = `league-${safeFileName(bundle.league.name)}-${safeFileName(bundle.league.season.name)}.json`
            a.click()
            URL.revokeObjectURL(url)
            return {}
        } catch {
            return { error: "ساخت فایل JSON ناموفق بود" }
        }
    }

    /**
     * خروجی اکسل:
     *  - شیت Data  → مرجع کامل (لیگ + همهٔ تورنمنت‌ها به‌صورت JSON تکه‌شده)
     *  - بقیهٔ شیت‌ها فقط برای خواندن انسان؛ در ایمپورت نادیده گرفته می‌شوند
     */
    function exportLeagueToExcel(leagueId: string): { error?: string } {
        const bundle = buildLeagueBundle(leagueId)
        if (!bundle) return { error: "لیگ یافت نشد" }

        try {
            const league = bundle.league
            const tById = new Map(bundle.tournaments.map((t) => [t.id, t]))
            const wb = XLSX.utils.book_new()
            const addSheet = (rows: any[], name: string) => {
                if (!rows.length) rows = [{}]
                XLSX.utils.book_append_sheet(wb, XLSX.utils.json_to_sheet(rows), name)
            }

            // ── Data: مرجع بازیابی (سقف ۳۲۷۶۷ کاراکتر هر سلول اکسل)
            const raw = JSON.stringify(bundle)
            const CHUNK = 30000
            const dataRows: { seq: number; part: string }[] = []
            for (let i = 0; i < raw.length; i += CHUNK)
                dataRows.push({ seq: dataRows.length, part: raw.slice(i, i + CHUNK) })
            addSheet(dataRows, "Data")

            // ── Info
            const cfg = league.season.scoringConfig
            addSheet([{
                name: league.name,
                gender: league.gender,
                ageCategory: league.ageCategory,
                format: league.format ?? "grandPrix",
                season: league.season.name,
                stageCount: league.season.stages.length,
                tournamentCount: bundle.tournaments.length,
                gold: cfg.gold,
                silver: cfg.silver,
                bronze: cfg.bronze,
                weighInPoint: cfg.weighInPoint,
                winPoint: cfg.winPoint,
                countWeighIn: cfg.countWeighIn ? 1 : 0,
                countWin: cfg.countWin ? 1 : 0,
                teamsToPromote: league.promotionRules.teamsToPromote,
                autoPromote: league.promotionRules.autoPromote ? 1 : 0,
                exportedAt: bundle.exportedAt,
            }], "Info")

            // ── Weeks: نقشهٔ مرحله/هفته/تورنمنت
            addSheet(league.season.stages.flatMap((s) =>
                s.weeks.map((w) => {
                    const t = w.tournamentId ? tById.get(w.tournamentId) : undefined
                    return {
                        stage: s.name,
                        stageOrder: s.order,
                        stageCompleted: s.completed ? 1 : 0,
                        promotedClubs: (s.promotedClubs ?? []).join("، "),
                        week: w.name,
                        weekOrder: w.order,
                        locked: w.locked ? 1 : 0,
                        tournament: t?.name ?? "",
                        date: t?.date ?? "",
                        athletes: t?.athletes.length ?? 0,
                        matches: t?.matches.length ?? 0,
                        completedMatches: t?.matches.filter((m) => m.status === "completed").length ?? 0,
                    }
                })
            ), "Weeks")

            // ── رتبه‌بندی بازیکنان و تیم‌ها (هر مرحله + کل فصل)
            const playerRows = [
                ...league.season.stages.flatMap((s) =>
                    rankPlayers(s.playerPoints).map((r) => ({ scope: s.name, ...r }))
                ),
                ...rankPlayers(league.season.seasonPlayerPoints).map((r) => ({ scope: "کل فصل", ...r })),
            ]
            addSheet(playerRows, "PlayerRanks")

            const teamRows = [
                ...league.season.stages.flatMap((s) =>
                    rankTeams(s.teamPoints).map((r) => ({ scope: s.name, ...r }))
                ),
                ...rankTeams(league.season.seasonTeamPoints).map((r) => ({ scope: "کل فصل", ...r })),
            ]
            addSheet(teamRows, "TeamRanks")

            // ── بازی‌های همهٔ تورنمنت‌ها (خوانا)
            const sumScore = (s: any) =>
                (s?.punch ?? 0) + (s?.bodyKick ?? 0) + (s?.headKick ?? 0) +
                (s?.turningBodyKick ?? 0) + (s?.turningHeadKick ?? 0)

            addSheet(league.season.stages.flatMap((s) =>
                s.weeks.flatMap((w) => {
                    const t = w.tournamentId ? tById.get(w.tournamentId) : undefined
                    if (!t) return []
                    const nameOf = (id?: string | null) =>
                        t.athletes.find((a) => a.id === id)?.name ?? ""
                    return t.matches.map((m) => ({
                        stage: s.name,
                        week: w.name,
                        tournament: t.name,
                        weightCategory: m.weightCategory,
                        round: m.round,
                        side: m.side,
                        court: m.court,
                        order: m.order,
                        isBye: m.isBye ? 1 : 0,
                        status: m.status ?? "",
                        athlete1: nameOf(m.athlete1Id),
                        athlete2: nameOf(m.athlete2Id),
                        winner: nameOf(m.winnerId),
                        winType: m.result?.winType ?? "",
                        roundScores: (m.result?.rounds ?? [])
                            .map((r) => `${sumScore(r.blue)}-${sumScore(r.red)}`)
                            .join(" | "),
                    }))
                })
            ), "Matches")

            // ── ورزشکاران همهٔ تورنمنت‌ها (خوانا)
            addSheet(bundle.tournaments.flatMap((t) =>
                t.athletes.map((a) => ({
                    tournament: t.name,
                    name: a.name,
                    club: a.club,
                    weightCategory: a.weightCategory,
                    ranking: a.ranking ?? "",
                    weighedIn: a.weighedIn ? 1 : 0,
                }))
            ), "Athletes")

            XLSX.writeFile(wb, `league-${safeFileName(league.name)}-${safeFileName(league.season.name)}.xlsx`)
            return {}
        } catch {
            return { error: "ساخت فایل اکسل ناموفق بود" }
        }
    }

    // ─── خواندن و اعتبارسنجی باندل ───

    async function readBundleFromFile(file: File): Promise<{ bundle?: LeagueBundle; error?: string }> {
        let parsed: any = null
        try {
            if (/\.json$/i.test(file.name)) {
                parsed = JSON.parse(await file.text())
            } else {
                const wb = XLSX.read(await file.arrayBuffer(), { type: "array" })
                if (!wb.Sheets["Data"]) return { error: "شیت Data پیدا نشد؛ این فایل خروجی لیگ نیست" }
                const rows = XLSX.utils.sheet_to_json<any>(wb.Sheets["Data"])
                if (!rows.length) return { error: "شیت Data خالی است" }
                parsed = JSON.parse(
                    rows.sort((a, b) => Number(a.seq) - Number(b.seq))
                        .map((r) => String(r.part ?? ""))
                        .join("")
                )
            }
        } catch {
            return { error: "خواندن فایل ناموفق بود؛ فایل خراب یا نامعتبر است" }
        }

        if (!parsed?.league?.season || !Array.isArray(parsed.league.season.stages))
            return { error: "ساختار لیگ در فایل یافت نشد" }
        if (Number(parsed.version) > BUNDLE_VERSION)
            return { error: "این فایل با نسخهٔ جدیدتری از برنامه ساخته شده؛ برنامه را به‌روزرسانی کنید" }
        if (!Array.isArray(parsed.tournaments)) parsed.tournaments = []

        return { bundle: parsed as LeagueBundle }
    }

    /** ساخت idهای تازه برای حالت «اضافه کردن»؛ idهای ورزشکار/بازی دست‌نخورده می‌مانند
     *  چون کلیدهای playerPoints و ارجاع‌های براکت (nextMatchId, winnerId, ...) به آن‌ها وابسته‌اند */
    function remapBundleIds(bundle: LeagueBundle, nameSuffix?: string): LeagueBundle {
        const b = deepClone(bundle)
        const tournamentIdMap = new Map<string, string>()
        const weekIdMap = new Map<string, string>()

        for (const t of b.tournaments) {
            const next = crypto.randomUUID()
            tournamentIdMap.set(t.id, next)
            t.id = next
        }

        b.league.id = crypto.randomUUID()
        b.league.season.id = crypto.randomUUID()
        if (nameSuffix) b.league.name = `${b.league.name}${nameSuffix}`

        for (const stage of b.league.season.stages) {
            stage.id = crypto.randomUUID()
            for (const week of stage.weeks) {
                const next = crypto.randomUUID()
                weekIdMap.set(week.id, next)
                week.id = next
                if (week.tournamentId)
                    week.tournamentId = tournamentIdMap.get(week.tournamentId) ?? undefined
            }
        }

        for (const t of b.tournaments) {
            t.leagueId = b.league.id
            if (t.weekId) t.weekId = weekIdMap.get(t.weekId) ?? null
        }

        return b
    }

    /**
     * ایمپورت لیگ از فایل اکسل/JSON.
     * mode:
     *  - "replace" → اگر لیگی با همان id موجود بود، کامل جایگزین می‌شود (پیش‌فرض وقتی موجود است)
     *  - "add"     → همیشه به‌عنوان لیگ جدید با idهای تازه اضافه می‌شود
     * recalculate: بازمحاسبهٔ امتیاز مراحلی که تورنمنت‌شان در فایل بوده
     */
    async function importLeagueFromFile(
        file: File,
        options: { mode?: "replace" | "add"; recalculate?: boolean } = {}
    ): Promise<{ league?: League; error?: string; replaced?: boolean }> {
        const { bundle, error } = await readBundleFromFile(file)
        if (!bundle) return { error }

        const tournamentStore = useTournamentStore()
        const existingIndex = leagues.value.findIndex((l) => l.id === bundle.league.id)
        const mode = options.mode ?? (existingIndex >= 0 ? "replace" : "add")
        const recalculate = options.recalculate ?? true

        const payload = mode === "add"
            ? remapBundleIds(bundle, existingIndex >= 0 ? " (کپی)" : undefined)
            : deepClone(bundle)

        // ── تورنمنت‌ها: upsert بر اساس id تا اتصال هفته‌ها سالم بماند
        for (const t of payload.tournaments) {
            const idx = tournamentStore.tournaments.findIndex((x) => x.id === t.id)
            if (idx >= 0) tournamentStore.tournaments[idx] = t
            else tournamentStore.tournaments.push(t)
        }
        tournamentStore.save()

        // ── لیگ
        const league = payload.league
        league.updatedAt = new Date().toISOString()

        migrateLeague(league)

        let replaced = false
        if (mode === "replace" && existingIndex >= 0) {
            leagues.value[existingIndex] = league
            replaced = true
        } else {
            leagues.value.push(league)
        }

        currentLeagueId.value = league.id
        save()

        // ── بازمحاسبه: فقط مراحلی که تورنمنت‌شان واقعاً در دسترس است،
        //    تا امتیازهای ذخیره‌شدهٔ مراحل بدون تورنمنت پاک نشوند
        if (recalculate) {
            for (const stage of league.season.stages) {
                const hasTournament = stage.weeks.some(
                    (w) => w.tournamentId && tournamentStore.tournaments.some((t) => t.id === w.tournamentId)
                )
                if (hasTournament) recalculateStagePoints(league.id, stage.order)
            }
        }

        return { league, replaced }
    }

    // ─── باشگاه‌ها ───

    const clubByName = (league: League, name: string) =>
        league.clubs.find(c => c.name === name) ?? null

    function addClub(
        leagueId: string,
        input: { name: string; groupId?: string | null; active?: boolean }
    ): LeagueClub | null {
        const league = findLeague(leagueId)
        if (!league) return null

        const name = normalizeClubName(input.name)
        if (!name) return null
        if (clubByName(league, name)) return null


        const club: LeagueClub = {
            id: crypto.randomUUID(),
            name,
            groupId: null,
            active: input.active ?? true,
            createdAt: new Date().toISOString(),
        }
        league.clubs.push(club)

        if (input.groupId) assignClubToGroup(leagueId, club.id, input.groupId)

        touch(league)
        save()
        return club
    }

    /** کلید teamPoints نام باشگاه است، پس تغییر نام باید در امتیازها هم منتشر شود */
    function renameTeamPointsKey(
        table: Record<string, TeamPointsRecord>,
        prev: string,
        next: string
    ) {
        const rec = table[prev]
        if (!rec) return
        delete table[prev]
        rec.club = next
        const existing = table[next]
        table[next] = existing ? mergeRecords(existing, rec) : rec
        table[next].club = next
    }

    function renameClub(leagueId: string, clubId: string, name: string) {
        const league = findLeague(leagueId)
        const club = league?.clubs.find(c => c.id === clubId)
        if (!league || !club) return

        const next = normalizeClubName(name)
        if (!next) throw new Error("نام باشگاه نمی‌تواند خالی باشد")
        if (league.clubs.some(c => c.id !== clubId && c.name === next))
            throw new Error("باشگاهی با این نام وجود دارد")

        const prev = club.name
        if (prev === next) return
        club.name = next

        for (const g of league.season.groups)
            g.clubs = g.clubs.map(c => (c === prev ? next : c))

        for (const stage of league.season.stages) {
            renameTeamPointsKey(stage.teamPoints, prev, next)
            for (const rec of Object.values(stage.playerPoints))
                if (rec.club === prev) rec.club = next
            stage.promotedClubs = (stage.promotedClubs ?? []).map(c => (c === prev ? next : c))
        }

        renameTeamPointsKey(league.season.seasonTeamPoints, prev, next)
        for (const rec of Object.values(league.season.seasonPlayerPoints))
            if (rec.club === prev) rec.club = next

        touch(league)
        save()
    }

    function setClubActive(leagueId: string, clubId: string, active: boolean) {
        const league = findLeague(leagueId)
        const club = league?.clubs.find(c => c.id === clubId)
        if (!league || !club) return
        club.active = active
        touch(league)
        save()
    }

    /** باشگاهِ دارای امتیاز ثبت‌شده حذف نمی‌شود، فقط غیرفعال می‌شود تا تاریخچه نشکند */
    function removeClub(leagueId: string, clubId: string): "deleted" | "deactivated" | null {
        const league = findLeague(leagueId)
        const club = league?.clubs.find(c => c.id === clubId)
        if (!league || !club) return null

        const name = club.name
        const hasHistory =
            league.season.stages.some(s => name in s.teamPoints) ||
            name in league.season.seasonTeamPoints

        for (const g of league.season.groups)
            g.clubs = g.clubs.filter(c => c !== name)
        club.groupId = null

        if (hasHistory) {
            club.active = false
            touch(league)
            save()
            return "deactivated"
        }

        league.clubs = league.clubs.filter(c => c.id !== clubId)
        touch(league)
        save()
        return "deleted"
    }

    // ─── گروه‌ها ───

    const sortedGroups = (league: League) =>
        [...league.season.groups].sort((a, b) => a.order - b.order)

    function addGroup(leagueId: string, name?: string): LeagueGroup | null {
        const league = findLeague(leagueId)
        if (!league) return null

        const i = league.season.groups.length
        const group: LeagueGroup = {
            id: crypto.randomUUID(),
            name: normalizeClubName(name) || `گروه ${GROUP_LETTERS[i] ?? i + 1}`,
            order: i + 1,
            clubs: [],
        }

        league.season.groups.push(group)
        touch(league)
        save()
        return group
    }

    function renameGroup(leagueId: string, groupId: string, name: string) {
        const league = findLeague(leagueId)
        const group = league?.season.groups.find(g => g.id === groupId)
        if (!league || !group) return

        const next = normalizeClubName(name)
        if (!next) throw new Error("نام گروه نمی‌تواند خالی باشد")

        group.name = next
        touch(league)
        save()
    }

    function removeGroup(leagueId: string, groupId: string) {
        const league = findLeague(leagueId)
        if (!league) return
        if (!league.season.groups.some(g => g.id === groupId)) return

        for (const c of league.clubs)
            if (c.groupId === groupId) c.groupId = null

        league.season.groups = sortedGroups(league)
            .filter(g => g.id !== groupId)
            .map((g, i) => ({ ...g, order: i + 1 }))

        touch(league)
        save()
    }

    /** groupId = null یعنی باشگاه بی‌گروه شود */
    function assignClubToGroup(leagueId: string, clubId: string, groupId: string | null) {
        const league = findLeague(leagueId)
        const club = league?.clubs.find(c => c.id === clubId)
        if (!league || !club) return

        const target = groupId
            ? league.season.groups.find(g => g.id === groupId) ?? null
            : null
        if (groupId && !target) return

        // خروج از گروه فعلی
        for (const g of league.season.groups)
            g.clubs = g.clubs.filter(c => c !== club.name)

        if (target && !target.clubs.includes(club.name))
            target.clubs.push(club.name)

        club.groupId = target?.id ?? null

        touch(league)
        save()
    }

    /**
     * پخش متوازن باشگاه‌ها بین گروه‌ها.
     *  - mode "all" → همه از صفر بازتوزیع می‌شوند
     *  - mode "unassigned" → فقط باشگاه‌های بی‌گروه اضافه می‌شوند
     * باشگاه غیرفعال به‌صورت پیش‌فرض پخش نمی‌شود.
     */
    function distributeClubs(
        leagueId: string,
        options: { mode?: "all" | "unassigned"; includeInactive?: boolean } = {}
    ) {
        const league = findLeague(leagueId)
        if (!league) return

        const groups = sortedGroups(league)
        if (groups.length === 0) return

        const mode = options.mode ?? "all"
        const includeInactive = options.includeInactive ?? false

        const eligible = league.clubs.filter(c => includeInactive || c.active !== false)

        if (mode === "all") {
            for (const g of league.season.groups) g.clubs = []
            for (const c of league.clubs) c.groupId = null
        }

        const pool = mode === "all" ? eligible : eligible.filter(c => !c.groupId)

        // همیشه کم‌جمعیت‌ترین گروه اول پر می‌شود
        for (const club of pool) {
            const target = groups.reduce(
                (min, g) => (g.clubs.length < min.clubs.length ? g : min),
                groups[0]
            )
            target.clubs.push(club.name)
            club.groupId = target.id
        }

        touch(league)
        save()
    }

    /** برای هشدار در UI */
    const ungroupedClubs = (leagueId: string) =>
        findLeague(leagueId)?.clubs.filter(c => !c.groupId && c.active !== false) ?? []

    // ─── ایمپورت باشگاه ───

    function importClubs(leagueId: string, names: string[]) {
        const league = findLeague(leagueId)
        if (!league) return { added: [] as string[], skipped: [] as string[] }

        const added: string[] = []
        const skipped: string[] = []
        const seen = new Set(league.clubs.map(c => c.name))
        const now = new Date().toISOString()

        for (const raw of names) {
            const name = normalizeClubName(raw)
            if (!name) continue
            if (seen.has(name)) {
                skipped.push(name)
                continue
            }
            seen.add(name)
            league.clubs.push({
                id: crypto.randomUUID(),
                name,
                groupId: null,
                active: true,
                createdAt: now,
            })
            added.push(name)
        }

        if (added.length) {
            touch(league)
            save()
        }
        return { added, skipped }
    }

    /** متن چندخطی یا جداشده با کاما / ویرگول فارسی / نقطه‌ویرگول / تب */
    function importClubsFromText(leagueId: string, text: string) {
        return importClubs(
            leagueId,
            text.split(/\r?\n|[,،;\t]/).map(s => s.trim()).filter(Boolean)
        )
    }

    /** استخراج باشگاه‌ها از ورزشکاران تورنمنت‌های همین لیگ */
    function syncClubsFromTournaments(leagueId: string) {
        const league = findLeague(leagueId)
        if (!league) return { added: [] as string[], skipped: [] as string[] }

        const names = new Set<string>()
        for (const t of liveLeagueTournaments(league))
            for (const a of t.athletes) {
                const n = normalizeClubName(a.club)
                if (n) names.add(n)
            }

        return importClubs(leagueId, [...names])
    }

    // ─── رتبه‌بندی گروهی ───

    // ─── رتبه‌بندی گروهی ───

    function emptyTeamRecord(club: string): TeamPointsRecord {
        return {
            club,
            totalPoints: 0,
            gold: 0,
            silver: 0,
            bronze: 0,
            weighInPoints: 0,
            winPoints: 0,
            roundDiff: 0,
            wins2_0: 0,
            wins2_1: 0,
            losses0_2: 0,
            losses1_2: 0,
            totalWins: 0,
            totalLosses: 0,
            athleteCount: 0,
        }
    }

    function teamPointsSource(
        league: League,
        stageOrder?: number
    ): Record<string, TeamPointsRecord> {
        if (stageOrder == null) return league.season.seasonTeamPoints
        return league.season.stages.find((s) => s.order === stageOrder)?.teamPoints ?? {}
    }

    /** جدول هر گروه؛ باشگاه بدون امتیاز با رکورد صفر می‌آید تا از جدول حذف نشود */
    function getGroupTeamRankings(
        leagueId: string,
        stageOrder?: number
    ): GroupTeamRankings[] {
        const league = findLeague(leagueId)
        if (!league) return []

        const source = teamPointsSource(league, stageOrder)

        return sortedGroups(league).map((g) => {
            const subset: Record<string, TeamPointsRecord> = {}
            for (const name of g.clubs) {
                const club = clubByName(league, name)
                if (club && club.active === false) continue
                subset[name] = source[name] ?? emptyTeamRecord(name)
            }
            return {
                groupId: g.id,
                groupName: g.name,
                order: g.order,
                rows: rankTeams(subset),
            }
        })
    }

    /** n تیم اول هر گروه؛ اگر گروهی وجود نداشت، n تیم اول جدول کل */
    function getPromotionCandidates(
        leagueId: string,
        stageOrder?: number,
        perGroup?: number
    ): string[] {
        const league = findLeague(leagueId)
        if (!league) return []

        const count = perGroup ?? league.promotionRules.teamsToPromote

        if (league.season.groups.length === 0) {
            return getTeamRankings(leagueId, stageOrder)
                .slice(0, count)
                .map((r) => r.club)
        }

        return getGroupTeamRankings(leagueId, stageOrder).flatMap((g) =>
            g.rows.slice(0, count).map((r) => r.club)
        )
    }

    function moveAthletes(
        leagueId: string,
        targetGroupId: string | null,
        athletes: AthleteRef[]
    ) {
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return

        const groups = league.season.groups
        if (targetGroupId && !groups.some((g) => g.id === targetGroupId)) return

        const resolver = buildGroupResolver(groups)

        for (const g of groups) {
            g.athleteIds ??= []
            g.excludedAthleteIds ??= []
        }

        for (const a of athletes) {
            // ۱) پاک‌کردن هر تخصیص صریح قبلی
            for (const g of groups) {
                g.athleteIds = g.athleteIds!.filter((id) => id !== a.id)
            }

            const clubGroupId = resolver.clubGroupIdOf(a)

            if (targetGroupId) {
                const target = groups.find((g) => g.id === targetGroupId)!
                if (clubGroupId === targetGroupId) {
                    // به گروه باشگاهی خودش برمی‌گردد: استثنا را بردار، تخصیص صریح لازم نیست
                    target.excludedAthleteIds = target.excludedAthleteIds!.filter((id) => id !== a.id)
                } else {
                    target.athleteIds!.push(a.id)
                    // از گروه باشگاهی‌اش مستثنا شود تا در دو گروه نباشد
                    if (clubGroupId) {
                        const origin = groups.find((g) => g.id === clubGroupId)!
                        if (!origin.excludedAthleteIds!.includes(a.id)) {
                            origin.excludedAthleteIds!.push(a.id)
                        }
                    }
                }
            } else if (clubGroupId) {
                // بازگشت به استخر: باید از گروه باشگاهی هم مستثنا شود
                const origin = groups.find((g) => g.id === clubGroupId)!
                if (!origin.excludedAthleteIds!.includes(a.id)) {
                    origin.excludedAthleteIds!.push(a.id)
                }
            }
        }

        league.updatedAt = new Date().toISOString()
    }

    /** پاک‌کردن همه‌ی دخالت‌های دستی برای یک ورزشکار (بازگشت به رفتار خودکار باشگاهی) */
    function resetAthleteAssignment(leagueId: string, athleteIds: string[]) {
        const league = leagues.value.find((l) => l.id === leagueId)
        if (!league) return
        const ids = new Set(athleteIds)
        for (const g of league.season.groups) {
            g.athleteIds = (g.athleteIds ?? []).filter((id) => !ids.has(id))
            g.excludedAthleteIds = (g.excludedAthleteIds ?? []).filter((id) => !ids.has(id))
        }
        league.updatedAt = new Date().toISOString()
    }

    // ---------------------------------------------------------------
    // Helpers داخلی
    // ---------------------------------------------------------------

    /** لیگ را پیدا می‌کند و در صورت نبود خطا می‌دهد (از undefined-check تکراری جلوگیری می‌کند) */
    function requireLeague(leagueId: string): League {
        const league = leagues.value.find(l => l.id === leagueId)
        if (!league) throw new Error(`لیگ با شناسه ${leagueId} یافت نشد`)
        if (!league.athletes) league.athletes = []
        if (!league.clubs) league.clubs = []
        return league
    }

    // ---------------------------------------------------------------
// مدیریت ورزشکار
// ---------------------------------------------------------------

    /**
     * افزودن ورزشکار به لیگ.
     * اگر ورزشکاری با همان نام/باشگاه/جنسیت موجود باشد، رکورد موجود برگردانده می‌شود
     * (به‌جای ساخت رکورد تکراری که بعداً در تورنمنت باگ ارجاعی می‌سازد).
     */
    function addAthlete(leagueId: string, payload: AthleteDraft): LeagueAthlete {
        const league = requireLeague(leagueId)

        const name = normalizeText(payload.name)
        if (!name) throw new Error('نام ورزشکار الزامی است')

        const clubId = payload.clubId ?? null
        const club = clubId ? league.clubs.find(c => c.id === clubId) : null
        if (clubId && !club) throw new Error(`باشگاه با شناسه ${clubId} در این لیگ ثبت نشده است`)

        const gender = payload.gender ?? league.gender ?? null
        const key = athleteKey({ name, clubId, gender })
        const existing = league.athletes.find(a => athleteKey(a) === key)
        if (existing) {
            if (existing.isActive === false) {
                existing.isActive = true
                existing.updatedAt = new Date().toISOString()
                touch(league)
            }
            return existing
        }

        const now = new Date().toISOString()
        const athlete: LeagueAthlete = {
            id: crypto.randomUUID(),
            name,
            gender,
            clubId,
            club: club?.name ?? null,
            nationalId: payload.nationalId ?? null,
            birthDate: payload.birthDate ?? null,
            birthYear: payload.birthYear ?? null,
            beltDegree: payload.beltDegree ?? null,
            memberCode: payload.memberCode ?? null,
            groupId: payload.groupId ?? null,
            weight: null,
            weightCategory: payload.weightCategory ?? null,
            isActive: payload.isActive ?? true,
            createdAt: now,
            updatedAt: now,
        }


        league.athletes.push(athlete)
        touch(league)
        return athlete
    }

    /**
     * ویرایش ورزشکار. فیلدهای id/createdAt قابل تغییر نیستند.
     * تغییر clubId، اسنپ‌شات club را هم همگام می‌کند.
     * تمام اعتبارسنجی‌ها قبل از نوشتن انجام می‌شود تا در صورت خطا رکورد نیمه‌کاره نماند.
     */
    function updateAthlete(
        leagueId: string,
        athleteId: string,
        patch: Partial<AthleteDraft>,
    ): LeagueAthlete {
        const league = requireLeague(leagueId)
        const athlete = league.athletes.find(a => a.id === athleteId)
        if (!athlete) throw new Error(`ورزشکار با شناسه ${athleteId} یافت نشد`)

        // --- ۱) محاسبهٔ حالت بعدی (بدون تغییر رکورد) ---
        const next: Partial<LeagueAthlete> = {}

        if (patch.name !== undefined) {
            const name = normalizeText(patch.name)
            if (!name) throw new Error('نام ورزشکار نمی‌تواند خالی باشد')
            next.name = name
        }

        if (patch.clubId !== undefined) {
            const clubId = patch.clubId ?? null
            const club = clubId ? league.clubs.find(c => c.id === clubId) : null
            if (clubId && !club) {
                throw new Error(`باشگاه با شناسه ${clubId} در این لیگ ثبت نشده است`)
            }
            next.clubId = clubId
            next.club = club?.name ?? null
        }

        if (patch.gender !== undefined) {
            next.gender = patch.gender ?? null
        }

        // --- ۲) چک تکراری روی حالت بعدی ---
        const key = athleteKey({
            name: next.name ?? athlete.name,
            clubId: next.clubId ?? athlete.clubId,
            gender: next.gender ?? athlete.gender,
        })
        const clash = league.athletes.find(a => a.id !== athlete.id && athleteKey(a) === key)
        if (clash) {
            throw new Error('ورزشکار دیگری با همین نام و باشگاه در این لیگ وجود دارد')
        }

        // --- ۳) commit ---
        if (patch.gender !== undefined && patch.gender) {
            next.gender = patch.gender
        }

        if (patch.weightCategory !== undefined) next.weightCategory = patch.weightCategory ?? null
        if (patch.nationalId !== undefined) next.nationalId = patch.nationalId ?? null
        if (patch.birthDate !== undefined) next.birthDate = patch.birthDate ?? null
        if (patch.birthYear !== undefined) next.birthYear = patch.birthYear ?? null
        if (patch.beltDegree !== undefined) next.beltDegree = patch.beltDegree ?? null
        if (patch.memberCode !== undefined) next.memberCode = patch.memberCode ?? null
        if (patch.groupId !== undefined) next.groupId = patch.groupId ?? null
        if (patch.isActive !== undefined) next.isActive = patch.isActive

        Object.assign(athlete, next)
        athlete.updatedAt = new Date().toISOString()
        touch(league)
        return athlete
    }

    /** حذف نرم: رکورد برای حفظ ارجاع‌های تورنمنت باقی می‌ماند و فقط غیرفعال می‌شود. */
    function removeAthlete(leagueId: string, athleteId: string): void {
        const league = requireLeague(leagueId)
        const athlete = league.athletes.find(a => a.id === athleteId)
        if (!athlete) throw new Error('ورزشکار یافت نشد')

        if (athlete.isActive === false) return

        // پاک‌کردن تخصیص دستی تا رکورد غیرفعال در گروه باقی نماند
        // resetAthleteAssignment(leagueId, athleteId)

        athlete.isActive = false
        athlete.updatedAt = new Date().toISOString()
        touch(league)
    }

    return {
        leagues,
        webStandings,
        replaceFromBackend,
        currentLeagueId,
        currentLeague,
        selectLeague,
        clearSelectedLeague,
        createLeague,
        addWeek,
        createTournamentForWeek,
        linkWeekToTournament,
        lockWeek,
        recalculateStagePoints,
        promoteTeams,
        getTeamRankings,
        getPlayerRankings,
        updateScoringConfig,
        autoCarryAthletesToTournament,
        deleteLeague,
        carryAthletesFromPreviousTournament,
        unlinkWeekFromTournament,
        getOverallPlayerRankings,
        buildLeagueBundle,
        exportLeagueToExcel,
        exportLeagueToJson,
        importLeagueFromFile,

        // باشگاه‌ها
        findLeague,
        addClub,
        renameClub,
        setClubActive,
        removeClub,
        importClubs,
        importClubsFromText,
        syncClubsFromTournaments,

        // گروه‌ها
        addGroup,
        renameGroup,
        removeGroup,
        assignClubToGroup,
        distributeClubs,
        ungroupedClubs,

        // رتبه‌بندی گروهی
        getGroupTeamRankings,
        getPromotionCandidates,
        moveAthletes, resetAthleteAssignment,

        // ورزشکاران
        requireLeague,   // این را لازم نیست export کنی، داخلی است
        addAthlete,
        updateAthlete,
        removeAthlete,
    }
})
