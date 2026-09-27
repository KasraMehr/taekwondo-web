// src/stores/teamTournament.ts
import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import type {
    EncounterBout,
    EncounterLineup,
    MatchResult,
    MatchRound,
    ScheduleViolation,
    Team,
    TeamDrawOptions,
    TeamEncounter,
    TeamGroup,
    TeamMatch,
    TeamMember,
    TeamScoringConfig,
    TeamStandingEntry,
    TeamTieBreaker,
    TeamTournament,
    TeamWeightSlot,
    WinType,
} from '../types'
import {COURTS, isScheduled, MAX_COURTS_PER_GROUP, TIME_SLOTS} from '../types'
import type {AgeCategory, Gender} from '../data/categories'
import {WEIGHT_CATEGORIES} from '../data/categories'
import {slotCountFor} from '../utils/teamSlots'

export interface BoutSavePayload {
    winType: WinType
    /** یکی از این دو کافی است؛ winnerId اولویت دارد */
    winnerId?: string | null
    winnerSide?: 'home' | 'away' | null
    rounds?: MatchRound[]
    note?: string
}

const STORAGE_KEY = 'tkd_team_tournaments'
const now = () => new Date().toISOString()
const uid = () => crypto.randomUUID()
const otherCorner = (c: 'blue' | 'red'): 'blue' | 'red' => (c === 'blue' ? 'red' : 'blue')
function pairKey(x: string, y: string): string {
    return x < y ? `${x}|${y}` : `${y}|${x}`
}

const DRAFT_KEY = 'tkd_lineup_drafts'

/** پیش‌نویس‌های محلی: `${encounterId}:${teamId}` → entries */
type DraftMap = Record<string, Record<number, string | null>>

const loadDrafts = (): DraftMap => {
    try { return JSON.parse(localStorage.getItem(DRAFT_KEY) || '{}') } catch { return {} }
}
const draftKey = (encounterId: string, teamId: string) => `${encounterId}:${teamId}`

const ENC_PREFIX = 'enc:'
const encounterIdOf = (matchId: string) => `${ENC_PREFIX}${matchId}`


/* ═════════ پیش‌فرض‌ها ═════════ */

/** ترتیب پیش‌فرض تساوی‌شکن‌ها */
export const DEFAULT_TIE_BREAKERS = [
    'points', 'headToHead', 'boutDiff', 'roundDiff', 'boutsWon', 'fewestDsq', 'draw',
] as const satisfies readonly TeamTieBreaker[]

/** امتیاز یک مواجهه برای هر دو تیم */
function encounterPoints(e: TeamEncounter, sc: TeamScoringConfig) {
    if (e.status !== 'completed') return { home: 0, away: 0 }
    if (e.homeBouts > e.awayBouts) return { home: sc.pointsWin, away: sc.pointsLoss }
    if (e.awayBouts > e.homeBouts) return { home: sc.pointsLoss, away: sc.pointsWin }
    return { home: sc.pointsDraw, away: sc.pointsDraw }
}

/**
 * نگاشت رودررو: کلید `${A}|${B}` → اختلاف بافت A نسبت به B
 * مقدار مثبت یعنی A در رویارویی‌های مستقیم جلوتر است.
 */
function h2hMapOf(encs: TeamEncounter[]): Map<string, Record<string, number>> {
    const m = new Map<string, Record<string, number>>()
    for (const e of encs) {
        if (e.status !== 'completed') continue
        const k = pairKey(e.homeTeamId, e.awayTeamId)
        let rec = m.get(k)
        if (!rec) {
            rec = { [e.homeTeamId]: 0, [e.awayTeamId]: 0 }
            m.set(k, rec)
        }
        rec[e.homeTeamId] += e.homeBouts ?? 0
        rec[e.awayTeamId] += e.awayBouts ?? 0
    }
    return m
}

/**
 * مقایسهٔ دو ردیف بر اساس زنجیرهٔ تساوی‌شکن‌ها.
 * خروجی منفی = a بالاتر. صفر = واقعاً برابر (نیاز به پلی‌آف/قرعه).
 */
function compareByTieBreakers(
    a: TeamStandingEntry,
    b: TeamStandingEntry,
    order: readonly TeamTieBreaker[] | undefined,
    h2h: Map<string, Record<string, number>>,
): number {
    for (const rule of order ?? DEFAULT_TIE_BREAKERS) {
        let d = 0
        switch (rule) {
            case 'points':     d = b.points - a.points; break
            case 'boutDiff':   d = b.boutDiff - a.boutDiff; break
            case 'roundDiff':  d = b.roundDiff - a.roundDiff; break
            case 'boutsWon':   d = b.boutsWon - a.boutsWon; break
            case 'roundsWon':  d = b.roundsWon - a.roundsWon; break
            case 'headToHead': d = -(h2h.get(`${a.teamId}|${b.teamId}`) ?? 0); break
            case 'fewestDsq':  d = a.dsqAgainst - b.dsqAgainst; break // کمتر = بهتر
            case 'draw':       d = 0; break
            default: { void rule }
        }
        if (d !== 0) return d
    }
    return 0
}


/**
 * جدول رده‌بندی از مجموعهٔ مواجهه‌ها.
 * `only` برای محدود کردن به تیم‌های یک گروه.
 */
export function buildStandings(
    t: TeamTournament,
    encounters: TeamEncounter[],
    only?: Set<string>,
): TeamStandingEntry[] {
    const rows = new Map<string, TeamStandingEntry>()

    const row = (teamId: string): TeamStandingEntry | null => {
        if (only && !only.has(teamId)) return null
        const found = rows.get(teamId)
        if (found) return found
        const fresh: TeamStandingEntry = {
            teamId,
            teamName: t.teams.find((x) => x.id === teamId)?.name ?? '—',
            played: 0, wins: 0, draws: 0, losses: 0,
            boutsWon: 0, boutsLost: 0, boutDiff: 0,
            roundsWon: 0, roundsLost: 0, roundDiff: 0,
            dsqAgainst: 0,
            points: 0, rank: 0,
        }
        rows.set(teamId, fresh)
        return fresh
    }

    // همهٔ تیم‌های واجد شرایط حتی با صفر بازی در جدول بیایند
    for (const team of t.teams) row(team.id)

    for (const e of encounters) {
        if (e.status !== 'completed') continue
        const h = row(e.homeTeamId)
        const a = row(e.awayTeamId)
        if (!h || !a) continue

        const p = encounterPoints(e, t.scoring)
        h.played++; a.played++
        h.boutsWon += e.homeBouts; h.boutsLost += e.awayBouts
        a.boutsWon += e.awayBouts; a.boutsLost += e.homeBouts
        h.roundsWon += e.homeRounds; h.roundsLost += e.awayRounds
        a.roundsWon += e.awayRounds; a.roundsLost += e.homeRounds
        h.points += p.home; a.points += p.away

        if (e.homeBouts > e.awayBouts) { h.wins++; a.losses++ }
        else if (e.awayBouts > e.homeBouts) { a.wins++; h.losses++ }
        else { h.draws++; a.draws++ }

        // اخراج‌ها به‌عنوان جریمهٔ انضباطی
        for (const b of e.bouts) {
            const w = b.result?.winType
            if (w !== 'DSQ' && w !== 'PUN') continue
            if (b.winnerSide === 'home') a.dsqAgainst++
            else if (b.winnerSide === 'away') h.dsqAgainst++
        }
    }

    const table = [...rows.values()]
    for (const r of table) {
        r.boutDiff  = r.boutsWon  - r.boutsLost
        r.roundDiff = r.roundsWon - r.roundsLost
    }

    const h2h = h2hMapOf(encounters)
    table.sort((x, y) =>
        compareByTieBreakers(x, y, t.scoring.tieBreakers, h2h)
        || x.teamName.localeCompare(y.teamName, 'fa'))

    table.forEach((r, i) => { r.rank = i + 1 })
    return table
}

const defaultScoring = (blindLineup = false): TeamScoringConfig => ({
    pointsWin: 3,
    pointsDraw: 1,
    pointsLoss: 0,
    walkoverRounds: 2,
    tieBreakers: [...DEFAULT_TIE_BREAKERS],
    blindLineup,
})

const defaultDrawOptions = (): TeamDrawOptions => ({
    minRest: 2,
    matsPerEncounter: 1,
    parallel: false,
    drawType: 'random',
    iterations: 3000,
})

/**
 * اسلات‌های جای‌نگهدار. لیست واقعی اوزان تیمی در `data/categories` موجود نبود،
 * پس عنوان موقت ساخته می‌شود و در صفحهٔ مسابقه با defineWeightSlots جایگزین می‌شود.
 */
const buildDefaultWeightSlots = (
    ageCategory: AgeCategory,
    gender: Gender,
): TeamWeightSlot[] => {
    const count = slotCountFor(ageCategory)
    const slots: TeamWeightSlot[] = []
    for (let i = 1; i <= count; i++) {
        slots.push({ slotNo: i, title: `وزن ${i}`, gender })
    }
    return slots
}


/* ═════════ ورودی ساخت ═════════ */

export interface CreateTeamTournamentInput {
    name: string
    date: string
    gender: Gender
    ageCategory: AgeCategory
    courts?: number
    blindLineup?: boolean
    sourceTournamentId?: string | null
    leagueId?: string | null
    stageOrder?: number | null
    weekId?: string | null
}
/* ═════════ مهاجرت رکوردهای قدیمی ═════════ */

function migrate(raw: any): TeamTournament {
    const gender: Gender = raw?.gender ?? 'male'
    const ageCategory: AgeCategory = raw?.ageCategory ?? 'بزرگسالان'
    const { slotCount: _legacy, ...rawScoring } = raw?.scoring ?? {}
    const expected = slotCountFor(ageCategory)

    const weightSlots: TeamWeightSlot[] =
        raw?.weightSlots?.length === expected
            ? raw.weightSlots
            : buildDefaultWeightSlots(ageCategory, gender)

    return {
        ...raw,
        gender,
        ageCategory,
        courts: raw?.courts ?? 1,
        status: raw?.status ?? 'draft',
        scoring: { ...defaultScoring(!!rawScoring.blindLineup), ...rawScoring },
        drawOptions: { ...defaultDrawOptions(), ...(raw?.drawOptions ?? {}) },
        weightSlots,
        athletes: raw?.athletes ?? [],
        teams: (raw?.teams ?? []).map((t: any) => ({ ...t, roster: t?.roster ?? [] })),
        encounters: (raw?.encounters ?? []).map((e: any) => ({
            ...e,
            lineups: e?.lineups ?? [],
            bouts: e?.bouts ?? [],
        })),
        groups: raw?.groups ?? [],
        matches: raw?.matches ?? [],
        courtsPerGroup: raw?.courtsPerGroup ?? {},
        sourceTournamentId: raw?.sourceTournamentId ?? null,
        leagueId: raw?.leagueId ?? null,
        stageOrder: raw?.stageOrder ?? null,
        weekId: raw?.weekId ?? null,
        createdAt: raw?.createdAt ?? now(),
        updatedAt: raw?.updatedAt ?? raw?.createdAt ?? now(),
    } as TeamTournament
}

function load(): TeamTournament[] {
    try {
        const raw = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]')
        return Array.isArray(raw) ? raw.map(migrate) : []
    } catch {
        return []
    }
}

/* ═════════ کمک‌تابع‌های زمین و مبارزه ═════════ */

/** برچسب زمین: 1→A ... 4→D */
export const courtLabel = (c: number) => ['A', 'B', 'C', 'D'][c - 1] ?? String(c)

export const useTeamTournamentStore = defineStore('teamTournament', () => {
    const items = ref<TeamTournament[]>(load())
    const currentId = ref<string | null>(null)

    const save = () => localStorage.setItem(STORAGE_KEY, JSON.stringify(items.value))
    const current = computed(() => items.value.find((t) => t.id === currentId.value) ?? null)
    const find = (id: string) => items.value.find((t) => t.id === id)
    const touch = (t: TeamTournament) => { t.updatedAt = now() }

    /* ═════════ CRUD رویداد ═════════ */

    function create(input: CreateTeamTournamentInput): TeamTournament {
        const stamp = now()

        const t: TeamTournament = {
            courtsPerGroup: {}, groups: [], matches: [],
            id: uid(),
            name: input.name,
            date: input.date,
            courts: input.courts ?? 1,
            gender: input.gender,
            ageCategory: input.ageCategory,
            status: 'draft',
            scoring: defaultScoring(input.blindLineup ?? false),
            drawOptions: defaultDrawOptions(),
            weightSlots: buildDefaultWeightSlots(input.ageCategory, input.gender),
            athletes: [],
            teams: [],
            encounters: [],
            sourceTournamentId: input.sourceTournamentId ?? null,
            leagueId: input.leagueId ?? null,
            stageOrder: input.stageOrder ?? null,
            weekId: input.weekId ?? null,
            createdAt: stamp,
            updatedAt: stamp
        }

        items.value.push(t)
        currentId.value = t.id
        save()
        return t
    }

    const byId = (id?: string | null): TeamTournament | undefined =>
        id ? items.value.find((t) => t.id === id) : undefined

    const bySourceTournament = (tournamentId: string): TeamTournament | undefined =>
        items.value.find((t) => t.sourceTournamentId === tournamentId)

    function select(id: string) {
        currentId.value = byId(id) ? id : null
    }

    function clearSelection() {
        currentId.value = null
    }

    function remove(id: string) {
        items.value = items.value.filter((t) => t.id !== id)
        if (currentId.value === id) currentId.value = null
        save()
    }

    /* ═════════ تیم و روستر ═════════ */

    function addTeam(id: string, team: Omit<Team, 'id' | 'roster'>): { error?: string } {
        const t = find(id)
        if (!t) return { error: `تورنمنت یافت نشد (${String(id)})` }

        const name = (team.name ?? '').trim()
        if (!name) return { error: 'نام تیم الزامی است' }
        // if (t.teams.length >= 5) return { error: 'حداکثر ۵ تیم' }
        if (t.teams.some((x) => x.name.trim() === name)) return { error: 'نام تیم تکراری است' }

        t.teams.push({ ...team, name, id: uid(), roster: [] })
        if (t.status === 'draft') t.status = 'teamsRegistered'
        touch(t); save(); return {}
    }

    function updateTeam(
        id: string,
        teamId: string,
        patch: Partial<Omit<Team, 'id' | 'roster'>>,
    ): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const team = t.teams.find((x) => x.id === teamId); if (!team) return { error: 'تیم یافت نشد' }

        if (patch.name !== undefined) {
            const name = patch.name.trim()
            if (!name) return { error: 'نام تیم الزامی است' }
            if (t.teams.some((x) => x.id !== teamId && x.name.trim() === name))
                return { error: 'نام تیم تکراری است' }
            patch = { ...patch, name }
        }

        Object.assign(team, patch)
        touch(t); save(); return {}
    }

    function clearTeams(id: string): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        if (t.encounters.length) return { error: 'پس از قرعه‌کشی پاک‌سازی مجاز نیست' }
        t.teams = []
        t.status = 'draft'
        touch(t); save(); return {}
    }

    function removeTeam(id: string, teamId: string): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        if (t.encounters.length) return { error: 'پس از قرعه‌کشی حذف تیم مجاز نیست' }
        t.teams = t.teams.filter((x) => x.id !== teamId)
        if (!t.teams.length) t.status = 'draft'
        touch(t); save(); return {}
    }

    function addMember(id: string, teamId: string, member: TeamMember): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const team = t.teams.find((x) => x.id === teamId); if (!team) return { error: 'تیم یافت نشد' }
        if (team.roster.some((m) => m.athleteId === member.athleteId))
            return { error: 'این بازیکن قبلاً در روستر تیم است' }
        team.roster.push({ ...member, status: member.status ?? 'active' })
        touch(t); save(); return {}
    }

    function removeMember(id: string, teamId: string, athleteId: string) {
        const t = find(id); if (!t) return
        const team = t.teams.find((x) => x.id === teamId); if (!team) return
        team.roster = team.roster.filter((m) => m.athleteId !== athleteId)
        touch(t); save()
    }

    /* ═════════ ترکیب (Lineup) ═════════ */

    const lineupOf = (e: TeamEncounter, teamId: string) => e.lineups.find((l) => l.teamId === teamId)

    function submitLineup(
        id: string, encounterId: string, teamId: string, entries: Record<number, string | null>,
    ): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const e = t.encounters.find((x) => x.id === encounterId); if (!e) return { error: 'مواجهه یافت نشد' }
        if (e.status === 'completed') return { error: 'مواجهه پایان یافته' }
                // if (teamId !== e.homeTeamId && teamId !== e.awayTeamId) return { error: 'تیم در این مواجهه نیست' }
        if (lineupOf(e, teamId)?.lockedAt) return { error: 'ترکیب این تیم قفل شده' }

        const team = t.teams.find((x) => x.id === teamId); if (!team) return { error: 'تیم یافت نشد' }
        const roster = new Map(team.roster.map((m) => [m.athleteId, m]))
        const used = new Set<string>()
        const clean: Record<number, string | null> = {}

        for (const slot of t.weightSlots) {
            const athleteId = entries[slot.slotNo] ?? null
            clean[slot.slotNo] = athleteId
            if (!athleteId) continue

            const member = roster.get(athleteId)
            if (!member) return { error: 'بازیکن در روستر این تیم نیست' }
            if (member.status && member.status !== 'active')
                return { error: `بازیکن اسلات ${slot.slotNo} در دسترس نیست` }
            if (used.has(athleteId)) return { error: 'یک بازیکن در دو اسلات' }
            used.add(athleteId)

            if (member.weighInKg != null) {
                if (slot.minKg != null && member.weighInKg < slot.minKg)
                    return { error: `وزن اسلات ${slot.slotNo} کمتر از حد` }
                if (slot.maxKg != null && member.weighInKg > slot.maxKg)
                    return { error: `وزن اسلات ${slot.slotNo} بیشتر از حد` }
            }
        }

        const lineup: EncounterLineup = { teamId, entries: clean, submittedAt: now() }
        const idx = e.lineups.findIndex((l) => l.teamId === teamId)
        if (idx > -1) e.lineups[idx] = lineup; else e.lineups.push(lineup)

        const home = lineupOf(e, e.homeTeamId)
        const away = lineupOf(e, e.awayTeamId)
        if (home?.submittedAt && away?.submittedAt) {
            home.lockedAt ??= now()
            away.lockedAt ??= now()
            e.revealedAt ??= now()
            buildBouts(t, e, home, away)
        }

        touch(t); save(); return {}
    }

    /** مبارزه‌ها را از weightSlots می‌سازد؛ id و زمین مبارزه‌های موجود حفظ می‌شود */
    function buildBouts(
        t: TeamTournament, e: TeamEncounter, home: EncounterLineup, away: EncounterLineup,
    ) {
        const prev = new Map(e.bouts.map((b) => [b.slotNo, b]))
        const mats = Math.max(1, t.drawOptions.matsPerEncounter)

        e.bouts = t.weightSlots.map((slot, i) => {
            const old = prev.get(slot.slotNo)
            const bout: EncounterBout = {
                id: old?.id ?? uid(),
                slotNo: slot.slotNo,
                homeAthleteId: home.entries[slot.slotNo] ?? null,
                awayAthleteId: away.entries[slot.slotNo] ?? null,
                court: old?.court ?? e.court + (i % mats),
                status: 'pending',
                winnerSide: null,
                homeRoundsWon: 0,
                awayRoundsWon: 0,
                homeCorner: old?.homeCorner ?? 'blue',
            }
            if (!bout.homeAthleteId && !bout.awayAthleteId) {
                bout.status = 'completed'   // هر دو اسلات خالی: بدون برنده و بدون راند
            } else if (!bout.homeAthleteId) {
                applyWalkover(bout, 'away', t.scoring.walkoverRounds)
            } else if (!bout.awayAthleteId) {
                applyWalkover(bout, 'home', t.scoring.walkoverRounds)
            }
            return bout
        })

        recomputeEncounter(e, t.scoring)
    }

    function applyWalkover(b: EncounterBout, winner: 'home' | 'away', walkoverRounds: number) {
        b.winnerSide = winner
        b.status = 'completed'
        b.homeRoundsWon = winner === 'home' ? walkoverRounds : 0
        b.awayRoundsWon = winner === 'away' ? walkoverRounds : 0
        b.result = buildResult(b, 'WO', winner, [])
    }

    function buildResult(
        b: EncounterBout, winType: WinType, winnerSide: 'home' | 'away' | null,
        rounds: MatchRound[], note?: string,
    ): MatchResult {
        const homeIsBlue = b.homeCorner === 'blue'
        return {
            winType,
            blueId: homeIsBlue ? b.homeAthleteId : b.awayAthleteId,
            redId: homeIsBlue ? b.awayAthleteId : b.homeAthleteId,
            winnerId: winnerSide === 'home' ? b.homeAthleteId : winnerSide === 'away' ? b.awayAthleteId : null,
            winnerCorner: winnerSide === 'home' ? b.homeCorner
                : winnerSide === 'away' ? otherCorner(b.homeCorner) : undefined,
            rounds,
            note,
            recordedAt: b.result?.recordedAt ?? now(),
            updatedAt: b.result ? now() : undefined,
        }
    }

    /** مجموع‌ها و وضعیت را همیشه از صفر روی bouts بازمی‌سازد */
    function recomputeEncounter(e: TeamEncounter, s: TeamScoringConfig) {
        let hb = 0, ab = 0, hr = 0, ar = 0
        for (const b of e.bouts) {
            if (b.status !== 'completed') continue
            hr += b.homeRoundsWon
            ar += b.awayRoundsWon
            if (b.winnerSide === 'home') hb++
            else if (b.winnerSide === 'away') ab++
        }
        e.homeBouts = hb; e.awayBouts = ab
        e.homeRounds = hr; e.awayRounds = ar

        if (hb > ab) { e.homePoints = s.pointsWin; e.awayPoints = s.pointsLoss }
        else if (hb < ab) { e.homePoints = s.pointsLoss; e.awayPoints = s.pointsWin }
        else { e.homePoints = s.pointsDraw; e.awayPoints = s.pointsDraw }

        const hasLineups = e.lineups.length === 2 && e.bouts.length > 0
        if (hasLineups && e.bouts.every((b) => b.status === 'completed')) {
            e.status = 'completed'
            e.finalizedAt ??= now()
        } else if (e.bouts.some((b) => b.status !== 'pending')) {
            e.status = 'ongoing'
        } else if (hasLineups) {
            e.status = 'lineupLocked'
        }
    }

    /* ═════════ اعتبارسنجی زمان‌بندی ═════════ */

    const scheduleViolations = computed<ScheduleViolation[]>(() => {
        const t = current.value; if (!t) return []
        const out: ScheduleViolation[] = []
        const mats = Math.max(1, t.drawOptions.matsPerEncounter)

        const bySlot = new Map<number, TeamEncounter[]>()
        const byTeam = new Map<string, TeamEncounter[]>()
        for (const e of t.encounters) {
            bySlot.set(e.slotIndex, [...(bySlot.get(e.slotIndex) ?? []), e])
            byTeam.set(e.homeTeamId, [...(byTeam.get(e.homeTeamId) ?? []), e])
            byTeam.set(e.awayTeamId, [...(byTeam.get(e.awayTeamId) ?? []), e])
        }

        for (const [slotIndex, list] of bySlot) {
            const seen = new Map<string, string>()
            for (const e of list) {
                for (const teamId of [e.homeTeamId, e.awayTeamId]) {
                    if (seen.has(teamId)) {
                        out.push({
                            encounterId: e.id, kind: 'conflict',
                            message: `تیم در نوبت ${slotIndex} دو مواجههٔ هم‌زمان دارد`,
                        })
                    } else seen.set(teamId, e.id)
                }
                if (e.court < 1 || e.court + mats - 1 > t.courts) {
                    out.push({
                        encounterId: e.id, kind: 'courtOverflow',
                        message: `زمین ${e.court} با ${mats} تخته در دسترس نیست (کل زمین‌ها: ${t.courts})`,
                    })
                }
            }
            if (!t.drawOptions.parallel && list.length > 1) {
                for (const e of list.slice(1)) out.push({
                    encounterId: e.id, kind: 'conflict',
                    message: `اجرای موازی غیرفعال است اما نوبت ${slotIndex} چند مواجهه دارد`,
                })
            }
            if (list.length * mats > t.courts) {
                for (const e of list) out.push({
                    encounterId: e.id, kind: 'courtOverflow',
                    message: `نوبت ${slotIndex} به ${list.length * mats} تخته نیاز دارد`,
                })
            }
        }

        for (const [, list] of byTeam) {
            const sorted = [...list].sort((a, b) => a.slotIndex - b.slotIndex)
            for (let i = 1; i < sorted.length; i++) {
                const gap = sorted[i].slotIndex - sorted[i - 1].slotIndex
                if (gap > 0 && gap < t.drawOptions.minRest) out.push({
                    encounterId: sorted[i].id, kind: 'rest',
                    message: `فاصلهٔ استراحت ${gap} نوبت، کمتر از حداقل ${t.drawOptions.minRest} است`,
                })
            }
        }

        return out
    })

    function addAthlete(
        id: string,
        input: {
            name: string
            nationalId?: string
            birthYear?: number | null
            club?: string
            weight?: string | null
        },
    ): { error?: string; id?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }

        const name = (input.name ?? '').trim()
        if (!name) return { error: 'نام ورزشکار الزامی است' }

        // اعتبارسنجی وزن
        const weights = allowedWeights(id)
        const weight = (input.weight ?? '').toString().trim()
        if (weight && weights.length && !weights.includes(weight))
            return { error: `وزن «${weight}» برای ردهٔ ${t.ageCategory} مجاز نیست` }

        t.athletes ??= []

        const nid = (input.nationalId ?? '').trim()
        const dup = t.athletes.find((a: any) =>
            nid ? String(a.nationalId ?? '') === nid : String(a.name ?? '').trim() === name,
        )
        if (dup) {
            // اگر ورزشکار موجود وزن نداشت و حالا وزن آمده، تکمیلش کن
            if (weight && !dup.weightCategory) { dup.weightCategory = weight; touch(t); save() }
            return { id: dup.id }
        }

        const athlete: any = {
            id: uid(),
            name,
            nationalId: nid || undefined,
            birthYear: input.birthYear ?? null,
            club: input.club ?? '',
            weight: weight || null,
            gender: t.gender,
            weighedIn: true
        }
        t.athletes.push(athlete)
        touch(t); save()
        return { id: athlete.id }
    }


    function updateAthlete(
        id: string,
        athleteId: string,
        patch: {
            name?: string
            nationalId?: string
            birthYear?: number | null
            club?: string
            weight?: string | null
            weighedIn: true
        },
    ): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const a: any = (t.athletes ?? []).find((x: any) => x.id === athleteId)
        if (!a) return { error: 'ورزشکار یافت نشد' }

        if (patch.name !== undefined) {
            const name = patch.name.trim()
            if (!name) return { error: 'نام ورزشکار الزامی است' }
            a.name = name
        }
        if (patch.nationalId !== undefined) a.nationalId = patch.nationalId.trim() || undefined
        if (patch.birthYear !== undefined) a.birthYear = patch.birthYear
        if (patch.club !== undefined) a.club = patch.club

        if (patch.weight !== undefined) {
            const weights = allowedWeights(id)
            const w = (patch.weight ?? '').toString().trim()
            if (w && weights.length && !weights.includes(w))
                return { error: `وزن «${w}» برای ردهٔ ${t.ageCategory} مجاز نیست` }
            a.weight = w || null
        }

        touch(t); save(); return {}
    }


    function removeAthlete(id: string, athleteId: string): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        if (t.encounters.length) return { error: 'پس از قرعه‌کشی حذف ورزشکار مجاز نیست' }
        t.athletes = (t.athletes ?? []).filter((a: any) => a.id !== athleteId)
        for (const team of t.teams) team.roster = team.roster.filter((m) => m.athleteId !== athleteId)
        touch(t); save(); return {}
    }

    const athleteById = (id: string, athleteId: string) =>
        (find(id)?.athletes ?? []).find((a: any) => a.id === athleteId) ?? null

    /* ═════════ گروه‌ها ═════════ */

    function addGroup(id: string, name: string): { error?: string; id?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const g: TeamGroup = { id: uid(), name: name.trim() || `گروه ${t.groups.length + 1}`, teamIds: [] }
        t.groups.push(g)
        t.courtsPerGroup[g.id] = []
        touch(t); save(); return { id: g.id }
    }

    function removeGroup(id: string, groupId: string) {
        const t = find(id); if (!t) return
        t.groups = t.groups.filter(g => g.id !== groupId)
        t.matches = t.matches.filter(m => m.groupId !== groupId)
        delete t.courtsPerGroup[groupId]
        touch(t); save()
    }

    function assignTeamToGroup(id: string, groupId: string, teamId: string): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const g = t.groups.find(x => x.id === groupId); if (!g) return { error: 'گروه یافت نشد' }
        for (const other of t.groups) other.teamIds = other.teamIds.filter(x => x !== teamId)
        g.teamIds.push(teamId)
        touch(t); save(); return {}
    }

    /** تعیین زمین‌های مجاز یک گروه (حداکثر ۲) */
    function setGroupCourts(id: string, groupId: string, courts: number[]): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        if (courts.length > MAX_COURTS_PER_GROUP)
            return { error: `هر گروه حداکثر ${MAX_COURTS_PER_GROUP} زمین` }
        if (courts.some(c => !COURTS.includes(c as any)))
            return { error: 'شماره زمین نامعتبر' }
        t.courtsPerGroup[groupId] = [...courts]
        // بازی‌هایی که روی زمین غیرمجاز مانده‌اند به استخر برگردند
        for (const m of t.matches) {
            if (m.groupId === groupId && m.courtId !== null && !courts.includes(m.courtId)) {
                m.courtId = null; m.slotId = null
            }
        }
        touch(t); save(); return {}
    }

    /* ═════════ تولید دوره‌ای بازی‌ها ═════════ */

    function generateGroupMatches(id: string, groupId: string): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const g = t.groups.find(x => x.id === groupId); if (!g) return { error: 'گروه یافت نشد' }
        if (g.teamIds.length < 2) return { error: 'حداقل ۲ تیم لازم است' }

        // الگوریتم دایره‌ای (circle method) با BYE برای تعداد فرد
        const teams: (string | null)[] = [...g.teamIds]
        if (teams.length % 2 === 1) teams.push(null)
        const n = teams.length
        const rounds = n - 1
        const half = n / 2

        const out: TeamMatch[] = []
        let arr = [...teams]
        for (let r = 0; r < rounds; r++) {
            for (let i = 0; i < half; i++) {
                const a = arr[i], b = arr[n - 1 - i]
                if (!a || !b) continue
                out.push({
                    id: uid(),
                    groupId,
                    teamAId: r % 2 === 0 ? a : b,
                    teamBId: r % 2 === 0 ? b : a,
                    round: r + 1,
                    courtId: null,
                    slotId: null,
                    scoreA: null,
                    scoreB: null,
                })
            }
            // چرخش: عنصر اول ثابت
            arr = [arr[0], arr[n - 1], ...arr.slice(1, n - 1)]
        }

        t.matches = [...t.matches.filter(m => m.groupId !== groupId), ...out]
        touch(t); save(); return {}
    }

    /* ═════════ اعتبارسنجی چیدمان ═════════ */

    /** آیا می‌توان بازی را در (courtId, slotId) گذاشت؟ */
    function validatePlacement(
        t: TeamTournament, match: TeamMatch, courtId: number, slotId: number,
    ): string | null {
        const allowed = t.courtsPerGroup[match.groupId] ?? []
        if (allowed.length && !allowed.includes(courtId))
            return 'این زمین برای این گروه مجاز نیست'

        for (const m of t.matches) {
            if (m.id === match.id) continue
            if (m.slotId !== slotId) continue

            // خانهٔ اشغال‌شده
            if (m.courtId === courtId) return 'این خانه اشغال است'

            // تداخل تیمی در همان سانس
            const ids = [m.teamAId, m.teamBId]
            if (ids.includes(match.teamAId) || ids.includes(match.teamBId))
                return 'یکی از تیم‌ها در همین سانس بازی دارد'
        }
        return null
    }

    /** جای‌گذاری/جابه‌جایی دستی؛ اگر خانه پر باشد swap می‌کند */
    function placeMatch(
        id: string, matchId: string, courtId: number | null, slotId: number | null,
    ): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const m = t.matches.find(x => x.id === matchId); if (!m) return { error: 'بازی یافت نشد' }

        // بازگرداندن به استخر
        if (courtId === null || slotId === null) {
            m.courtId = null; m.slotId = null
            touch(t); save(); return {}
        }

        const occupant = t.matches.find(x => x.id !== m.id && x.courtId === courtId && x.slotId === slotId)

        if (occupant) {
            // swap: اعتبارسنجی دوطرفه با فرض تعویض
            const from = { c: m.courtId, s: m.slotId }
            occupant.courtId = from.c; occupant.slotId = from.s
            m.courtId = courtId; m.slotId = slotId

            const e1 = m.courtId !== null && m.slotId !== null ? validatePlacement(t, m, m.courtId, m.slotId) : null
            const e2 = occupant.courtId !== null && occupant.slotId !== null
                ? validatePlacement(t, occupant, occupant.courtId, occupant.slotId) : null

            if (e1 || e2) {   // rollback
                m.courtId = from.c; m.slotId = from.s
                occupant.courtId = courtId; occupant.slotId = slotId
                return { error: e1 ?? e2 ?? 'جابه‌جایی ممکن نیست' }
            }
            touch(t); save(); return {}
        }

        const err = validatePlacement(t, m, courtId, slotId)
        if (err) return { error: err }

        m.courtId = courtId; m.slotId = slotId
        touch(t); save(); return {}
    }

    /* ═════════ چیدمان خودکار ═════════ */

    function autoSchedule(id: string): { error?: string; placed?: number; unplaced?: number } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }

        for (const m of t.matches) { m.courtId = null; m.slotId = null }

        // اول گروه‌های بزرگ‌تر، سپس ترتیب دور
        const sizeOf = (gid: string) => t.groups.find(g => g.id === gid)?.teamIds.length ?? 0
        const queue = [...t.matches].sort((a, b) =>
            sizeOf(b.groupId) - sizeOf(a.groupId) || a.round - b.round)

        let placed = 0
        for (const m of queue) {
            const courts = (t.courtsPerGroup[m.groupId]?.length ? t.courtsPerGroup[m.groupId] : [...COURTS])
            let done = false
            for (const slot of TIME_SLOTS) {
                for (const c of courts) {
                    if (!validatePlacement(t, m, c, slot.id)) {
                        m.courtId = c; m.slotId = slot.id
                        placed++; done = true; break
                    }
                }
                if (done) break
            }
        }

        touch(t); save()
        return { placed, unplaced: t.matches.length - placed }
    }

    /* ═════════ ثبت امتیاز بازی ═════════ */

    function setMatchScore(
        id: string, matchId: string, scoreA: number | null, scoreB: number | null,
    ): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const m = t.matches.find(x => x.id === matchId); if (!m) return { error: 'بازی یافت نشد' }
        m.scoreA = scoreA; m.scoreB = scoreB
        touch(t); save(); return {}
    }

    /* ═════════ سلکتورها ═════════ */

    /** بازی‌های چیده‌نشده (استخر) */
    const matchPool = computed(() =>
        (current.value?.matches ?? []).filter(m => !isScheduled(m)))

    /** جدول: [slotId][courtId] → TeamMatch | null */
    const scheduleGrid = computed(() => {
        const t = current.value
        const grid: Record<number, Record<number, TeamMatch | null>> = {}
        for (const s of TIME_SLOTS) {
            grid[s.id] = {}
            for (const c of COURTS) grid[s.id][c] = null
        }
        for (const m of t?.matches ?? []) {
            if (m.slotId !== null && m.courtId !== null) grid[m.slotId]![m.courtId] = m
        }
        return grid
    })

    const groupMatches = (groupId: string) =>
        (current.value?.matches ?? []).filter(m => m.groupId === groupId)

    /* ═════════ ترکیب: پیش‌نویس و ویرایش ═════════ */

    const drafts = ref<DraftMap>(loadDrafts())
    const saveDrafts = () => localStorage.setItem(DRAFT_KEY, JSON.stringify(drafts.value))

    /** ترکیب فعلی (ثبت‌شده یا پیش‌نویس) برای یک تیم در یک مواجهه */
    function getLineup(
        id: string, encounterId: string, teamId: string,
    ): Record<number, string | null> {
        const t = find(id)
        const e = t?.encounters.find((x) => x.id === encounterId)
        if (!t || !e) return {}

        const submitted = lineupOf(e, teamId)
        if (submitted) return { ...submitted.entries }

        const d = drafts.value[draftKey(encounterId, teamId)]
        if (d) return { ...d }

        // مقدار اولیه: همهٔ اسلات‌ها خالی
        const empty: Record<number, string | null> = {}
        for (const s of t.weightSlots) empty[s.slotNo] = null
        return empty
    }

    /** ترکیب قفل است؟ (ثبت نهایی شده یا مواجهه تمام شده) */
    function isLineupEditable(id: string, encounterId: string, teamId: string): boolean {
        const t = find(id)
        const e = t?.encounters.find((x) => x.id === encounterId)
        if (!t || !e) return false
        if (e.status === 'completed') return false
        return !lineupOf(e, teamId)?.lockedAt && !lineupOf(e, teamId)?.submittedAt
    }

    /** اختصاص یا پاک کردن یک اسلات در پیش‌نویس */
    function assignSlot(
        id: string, encounterId: string, teamId: string,
        slotNo: number, athleteId: string | null,
    ): { error?: string } {
        if (!isLineupEditable(id, encounterId, teamId)) return { error: 'ترکیب قابل ویرایش نیست' }
        const t = find(id)!
        const key = draftKey(encounterId, teamId)
        const entries = { ...getLineup(id, encounterId, teamId) }

        if (!t.weightSlots.some((s) => s.slotNo === slotNo)) return { error: 'اسلات نامعتبر' }

        if (athleteId) {
            const team = t.teams.find((x) => x.id === teamId)
            if (!team?.roster.some((m) => m.athleteId === athleteId))
                return { error: 'بازیکن در روستر این تیم نیست' }
            // اگر جای دیگری استفاده شده، از آنجا برداشته شود
            for (const s of t.weightSlots) {
                if (s.slotNo !== slotNo && entries[s.slotNo] === athleteId) entries[s.slotNo] = null
            }
        }

        entries[slotNo] = athleteId
        drafts.value[key] = entries
        saveDrafts()
        return {}
    }

    /** پاک کردن یک اسلات */
    const clearSlot = (id: string, encounterId: string, teamId: string, slotNo: number) =>
        assignSlot(id, encounterId, teamId, slotNo, null)

    /** پاک کردن کل ترکیب پیش‌نویس */
    function resetLineup(id: string, encounterId: string, teamId: string): { error?: string } {
        if (!isLineupEditable(id, encounterId, teamId)) return { error: 'ترکیب قابل ویرایش نیست' }
        const t = find(id)!
        const empty: Record<number, string | null> = {}
        for (const s of t.weightSlots) empty[s.slotNo] = null
        drafts.value[draftKey(encounterId, teamId)] = empty
        saveDrafts()
        return {}
    }

    /* ═════════ ترکیب: روستر و انتخاب‌پذیری ═════════ */

    /** آیا وزن ورزشکار با اسلات می‌خواند */
    function fitsSlot(member: TeamMember, slot: TeamWeightSlot): boolean {
        if (member.weighInKg == null) return true            // بدون وزن‌کشی: محدود نمی‌کنیم
        if (slot.minKg != null && member.weighInKg < slot.minKg) return false
        if (slot.maxKg != null && member.weighInKg > slot.maxKg) return false
        return true
    }

    /**
     * روستر تیم با وضعیت انتخاب‌پذیری نسبت به یک اسلات مشخص.
     * اگر slotNo داده نشود، فقط وضعیت کلی (استفاده‌شده/غیرفعال) برمی‌گردد.
     */
    function rosterFor(
        id: string, encounterId: string, teamId: string, slotNo?: number,
    ): Array<{
        athleteId: string
        name: string
        weightCategory?: string | null
        weighInKg?: number
        member: TeamMember
        usedInSlot: number | null
        isEligible: boolean
        reason?: string
    }> {
        const t = find(id)
        const team = t?.teams.find((x) => x.id === teamId)
        if (!t || !team) return []

        const entries = getLineup(id, encounterId, teamId)
        const slot = slotNo != null ? t.weightSlots.find((s) => s.slotNo === slotNo) : undefined
        const usedBy = new Map<string, number>()
        for (const [k, v] of Object.entries(entries)) if (v) usedBy.set(v, Number(k))

        return team.roster.map((member) => {
            const a: any = (t.athletes ?? []).find((x: any) => x.id === member.athleteId)
            const usedInSlot = usedBy.get(member.athleteId) ?? null

            let isEligible = true
            let reason: string | undefined

            if (member.status && member.status !== 'active') {
                isEligible = false
                reason = member.status === 'injured' ? 'مصدوم' : 'محروم'
            } else if (usedInSlot != null && usedInSlot !== slotNo) {
                isEligible = false
                reason = `در اسلات ${usedInSlot} انتخاب شده`
            } else if (slot && !fitsSlot(member, slot)) {
                isEligible = false
                reason = 'وزن با این اسلات نمی‌خواند'
            }

            return {
                athleteId: member.athleteId,
                name: a?.name ?? '—',
                weightCategory: a?.weight ?? a?.weightCategory ?? null,
                weighInKg: member.weighInKg,
                member,
                usedInSlot,
                isEligible,
                reason,
            }
        })
    }

    /** فقط بازیکنان مجاز برای یک اسلات */
    const eligibleForSlot = (id: string, encounterId: string, teamId: string, slotNo: number) =>
        rosterFor(id, encounterId, teamId, slotNo).filter((r) => r.isEligible)

    /* ═════════ ترکیب: اعتبارسنجی ═════════ */

    interface LineupIssueLite {
        level: 'error' | 'warning'
        code: string
        message: string
        slotNo?: number
        athleteId?: string
    }

    function validateLineup(
        id: string, encounterId: string, teamId: string,
    ): {
        isValid: boolean
        canSubmit: boolean
        issues: LineupIssueLite[]
        filledCount: number
        totalSlots: number
        emptySlots: number[]
    } {
        const t = find(id)
        const team = t?.teams.find((x) => x.id === teamId)
        const issues: LineupIssueLite[] = []
        const empty: number[] = []

        if (!t || !team) {
            return { isValid: false, canSubmit: false, issues, filledCount: 0, totalSlots: 0, emptySlots: [] }
        }

        const entries = getLineup(id, encounterId, teamId)
        const roster = new Map(team.roster.map((m) => [m.athleteId, m]))
        const seen = new Map<string, number>()
        let filled = 0

        for (const slot of t.weightSlots) {
            const athleteId = entries[slot.slotNo] ?? null

            if (!athleteId) {
                empty.push(slot.slotNo)
                issues.push({
                    level: 'warning', code: 'EMPTY_SLOT', slotNo: slot.slotNo,
                    message: `اسلات ${slot.slotNo} خالی است — باخت با WO`,
                })
                continue
            }

            filled++
            const member = roster.get(athleteId)

            if (!member) {
                issues.push({
                    level: 'error', code: 'NOT_IN_ROSTER', slotNo: slot.slotNo, athleteId,
                    message: `بازیکن اسلات ${slot.slotNo} در روستر تیم نیست`,
                })
                continue
            }

            if (member.status && member.status !== 'active') {
                issues.push({
                    level: 'error', code: 'INACTIVE_ATHLETE', slotNo: slot.slotNo, athleteId,
                    message: `بازیکن اسلات ${slot.slotNo} ${member.status === 'injured' ? 'مصدوم' : 'محروم'} است`,
                })
            }

            const dup = seen.get(athleteId)
            if (dup != null) {
                issues.push({
                    level: 'error', code: 'DUPLICATE_ATHLETE', slotNo: slot.slotNo, athleteId,
                    message: `یک بازیکن در اسلات‌های ${dup} و ${slot.slotNo}`,
                })
            } else seen.set(athleteId, slot.slotNo)

            if (member.weighInKg == null) {
                issues.push({
                    level: 'warning', code: 'NO_WEIGH_IN', slotNo: slot.slotNo, athleteId,
                    message: `وزن‌کشی اسلات ${slot.slotNo} ثبت نشده`,
                })
            } else if (!fitsSlot(member, slot)) {
                issues.push({
                    level: 'error', code: 'WRONG_SLOT', slotNo: slot.slotNo, athleteId,
                    message: `وزن ${member.weighInKg} با اسلات ${slot.slotNo} نمی‌خواند`,
                })
            }
        }

        if (filled === 0) {
            issues.push({ level: 'error', code: 'MIN_ATHLETES', message: 'حداقل یک بازیکن لازم است' })
        }

        const hasError = issues.some((i) => i.level === 'error')
        return {
            isValid: !hasError && empty.length === 0,
            canSubmit: !hasError,
            issues,
            filledCount: filled,
            totalSlots: t.weightSlots.length,
            emptySlots: empty,
        }
    }

    /* ═════════ ترکیب: ثبت نهایی ═════════ */

    /** پیش‌نویس را با submitLineup ثبت می‌کند */
    function submitDraftLineup(
        id: string, encounterId: string, teamId: string, submittedBy?: string,
    ): { error?: string } {
        const v = validateLineup(id, encounterId, teamId)
        if (!v.canSubmit) {
            return { error: v.issues.find((i) => i.level === 'error')?.message ?? 'ترکیب نامعتبر' }
        }

        const entries = getLineup(id, encounterId, teamId)
        const res = submitLineup(id, encounterId, teamId, entries)
        if (res.error) return res

        // متادیتای ثبت‌کننده
        const e = find(id)?.encounters.find((x) => x.id === encounterId)
        const l = e && lineupOf(e, teamId)
        if (l && submittedBy) (l as any).submittedBy = submittedBy

        delete drafts.value[draftKey(encounterId, teamId)]
        saveDrafts(); save()
        return {}
    }

    /** لغو ثبت پیش از قفل شدن (مثلاً مربی اشتباه ثبت کرده) */
    function unsubmitLineup(
        id: string, encounterId: string, teamId: string,
    ): { error?: string } {
        const t = find(id); if (!t) return { error: 'تورنمنت یافت نشد' }
        const e = t.encounters.find((x) => x.id === encounterId); if (!e) return { error: 'مواجهه یافت نشد' }
        const l = lineupOf(e, teamId); if (!l) return { error: 'ترکیبی ثبت نشده' }
        if (l.lockedAt) return { error: 'ترکیب قفل شده و قابل بازگشت نیست' }
        if (e.bouts.some((b) => b.status !== 'pending')) return { error: 'مبارزه‌ها شروع شده' }

        drafts.value[draftKey(encounterId, teamId)] = { ...l.entries }
        e.lineups = e.lineups.filter((x) => x.teamId !== teamId)
        e.bouts = []
        e.revealedAt = undefined
        e.status = 'scheduled'
        recomputeEncounter(e, t.scoring)

        saveDrafts(); touch(t); save(); return {}
    }



    function allowedWeights(id: string): string[] {
        const t = find(id)
        if (!t) return []
        return WEIGHT_CATEGORIES[t.ageCategory as keyof typeof WEIGHT_CATEGORIES]?.[t.gender] ?? []
    }

    /** اسلات‌های وزنی یک تورنمنت */
    function weightSlotsOf(tournamentId: string): readonly TeamWeightSlot[] {
        return find(tournamentId)?.weightSlots ?? []
    }

    /* ═════════ مواجهه: ساخت از بازی ═════════ */

    // matchId روی TeamEncounter اختیاری است
    const matchOf = (t: TeamTournament, matchId?: string) =>
        matchId ? t.matches.find((m) => m.id === matchId) : undefined

    /** فیلدهای مشتق‌شده از بازی را روی مواجهه می‌نشاند (زمین/سانس/تیم‌ها) */
    function syncEncounterFromMatch(t: TeamTournament, e: TeamEncounter, m?: TeamMatch) {
        if (!m) return
        const started = e.bouts.some((b) => b.status !== 'pending')
        // تیم‌ها فقط تا پیش از شروع مبارزه‌ها قابل بازنشانی‌اند
        if (!started) {
            e.homeTeamId = m.teamAId
            e.awayTeamId = m.teamBId
        }
        e.court = m.courtId ?? e.court ?? COURTS[0]
        e.slotIndex = m.slotId ?? e.slotIndex ?? 0
        e.groupId = m.groupId
        e.roundNo = m.round ?? e.roundNo ?? 0
        // زمین مبارزه‌هایی که دستی جابه‌جا نشده‌اند را هم بکش
        const mats = Math.max(1, t.drawOptions.matsPerEncounter)
        e.bouts.forEach((b, i) => {
            if (b.status === 'pending') b.court = e.court + (i % mats)
        })
    }

    function ensureEncounter(id: string, matchId: string): string | null {
        const t = find(id)
        if (!t || !matchId) return null
        if (!Array.isArray(t.encounters)) t.encounters = []

        const m = matchOf(t, matchId)
        if (!m) return null
        if (!isScheduled(m)) return null   // بازیِ بی‌زمین/بی‌سانس مواجهه نمی‌گیرد

        const eid = encounterIdOf(matchId)
        let e = t.encounters.find((x) => x.id === eid)

        if (!e) {
            e = {
                id: eid,
                matchId,
                roundNo: m.round ?? 0,
                groupId: m.groupId,
                homeTeamId: m.teamAId,
                awayTeamId: m.teamBId,
                court: m.courtId ?? COURTS[0],
                slotIndex: m.slotId ?? 0,
                status: 'scheduled',
                lineups: [],
                bouts: [],
                homeBouts: 0, awayBouts: 0,
                homeRounds: 0, awayRounds: 0,
                homePoints: 0, awayPoints: 0,
            }
            t.encounters.push(e)
        }

        syncEncounterFromMatch(t, e, m)
        touch(t); save()
        return eid
    }

    /** مواجههٔ یک بازی (اگر ساخته شده باشد) */
    const encounterOf = (id: string, matchId: string): TeamEncounter | null =>
        find(id)?.encounters.find((x) => x.matchId === matchId) ?? null

    /** مواجهه + مبارزه از روی boutId */

    const athleteMapOf = (id: string) =>
        new Map((find(id)?.athletes ?? []).map((a) => [a.id, a]))

    /* ═════════ رده‌بندی ═════════ */

    const standings = computed<TeamStandingEntry[]>(() =>
        current.value ? buildStandings(current.value, current.value.encounters) : [])

    /** برندهٔ کل رویداد (وقتی همهٔ بازی‌ها ثبت شده‌اند) */
    const champion = computed(() => {
        const t = current.value; if (!t) return null
        const scheduled = t.matches.filter(isScheduled)
        if (!scheduled.length) return null
        const done = t.encounters.filter((e) => e.status === 'completed').length
        if (done < scheduled.length) return null
        const top = standings.value[0]
        return top ? { teamId: top.teamId, teamName: top.teamName, points: top.points } : null
    })

    /* ═════════ تختهٔ اجرا ═════════ */

    /** جدول اجرای روز: هر سانس × هر زمین با وضعیت زندهٔ مواجهه */
    const board = computed(() => {
        const t = current.value
        return TIME_SLOTS.map((slot) => ({
            slotId: slot.id,
            label: slot.label,
            cells: COURTS.map((courtId) => {
                const m = t?.matches.find((x) => x.slotId === slot.id && x.courtId === courtId)
                const e = m && t ? t.encounters.find((x) => x.matchId === m.id) : null
                return {
                    courtId, courtName: courtLabel(courtId),
                    match: m ?? null,
                    encounterId: e?.id ?? null,
                    status: e?.status ?? (m ? 'scheduled' : 'empty'),
                    teamAName: m ? t?.teams.find((x) => x.id === m.teamAId)?.name ?? '—' : null,
                    teamBName: m ? t?.teams.find((x) => x.id === m.teamBId)?.name ?? '—' : null,
                    scoreA: m?.scoreA ?? null, scoreB: m?.scoreB ?? null,
                }
            }),
        }))
    })

    return {
        items, currentId, current, standings, scheduleViolations, save,
        // CRUD
        create, byId, bySourceTournament, select, clearSelection, remove,
        // تیم و روستر
        addTeam, updateTeam, clearTeams, removeTeam, addMember, removeMember,
        // ورزشکاران
        addAthlete, updateAthlete, removeAthlete, athleteById,
        // گروه‌ها و زمان‌بندی
        addGroup, removeGroup, assignTeamToGroup, setGroupCourts,
        generateGroupMatches, placeMatch, autoSchedule, setMatchScore,
        matchPool, scheduleGrid, groupMatches,
        // ترکیب (Lineup)
        getLineup, isLineupEditable, assignSlot, clearSlot, resetLineup,
        rosterFor, eligibleForSlot, validateLineup,
        submitDraftLineup, unsubmitLineup,
        // اوزان و اسلات‌ها
        weightSlotsOf, ensureEncounter,
        // مواجهه
        encounterOf, athleteMapOf, champion,
        // تختهٔ اجرا
        board, courtLabel

    }
})
