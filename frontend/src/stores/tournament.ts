import { defineStore } from "pinia"
import {ref, computed} from "vue"
import * as XLSX from "xlsx"
import type {
    Tournament,
    Athlete,
    Match,
    MatchResult,
    MatchRound,
    WinType,
    TournamentFormat,
    EliminationMatch,
    WeighIn
} from "../types"
import {
    emptyWeighIn, recordAttempt, sign, checkWeight,
    canWeigh, canSign, attemptsLeft, MAX_ATTEMPTS, migrateAthleteWeighIn,
} from "../utils/weighIn"
import {
    emptyRound, normalizeRounds, resolveMatch,
    isNonScoreWin, DEFAULT_RULES,
} from "../utils/scoring"
import {AgeCategory, Gender, weightOptions} from "../data/categories"
import { generateBracket, recordResult, hasLockedDownstream } from "../utils/bracket"
import { planCourts, assignCategoryCourts, syncOrders } from "../utils/scheduling"
import type { CategoryStat } from "../utils/scheduling"
import {useLeagueStore} from "./league.ts";
import { webApi } from "../webApi";

const STORAGE_KEY = "tkd_tournaments"

export const useTournamentStore = defineStore("tournament", () => {
    const tournaments = ref<Tournament[]>(
        (JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]") as Tournament[]).map((t) => ({
            ...t,
            athletes: t.athletes.map(migrateWeighIn),
        }))
    )

    const currentTournamentId = ref<string | null>(null)

    const save = () =>
        localStorage.setItem(STORAGE_KEY, JSON.stringify(tournaments.value))

    const currentTournament = computed(
        () => tournaments.value.find((t) => t.id === currentTournamentId.value) ?? null
    )

    function replaceFromServer(tournament: Tournament) {
        const index = tournaments.value.findIndex(item => item.id === tournament.id)
        if (index === -1) tournaments.value.push(tournament)
        else tournaments.value[index] = tournament
        save()
    }

    async function refreshFromServer(tournamentId: string) {
        replaceFromServer(await webApi().call<Tournament>(`/tournaments/${tournamentId}`))
    }

    function remote(task: () => Promise<unknown>, tournamentId?: string) {
        try {
            void task()
                .then(() => tournamentId ? refreshFromServer(tournamentId) : undefined)
                .catch(error => console.error('خطای همگام‌سازی با سرور:', error))
        } catch (error) {
            // Store tests and the desktop build intentionally run without a web session.
            if (savedWebSessionExists()) console.error('خطای همگام‌سازی با سرور:', error)
        }
    }

    function savedWebSessionExists() {
        return typeof localStorage !== 'undefined' && Boolean(localStorage.getItem('tkd_session'))
    }

    // ─────────────────────────── مسابقه ───────────────────────────

    function createTournament(
        name: string,
        date: string,
        courts: number,
        gender: Gender,
        ageCategory: AgeCategory,
        format: TournamentFormat = "grandPrix",
        leagueContext?: {
            leagueId: string;
            stageOrder: number;
            weekId: string;
        }
    ): Tournament {
        const now = new Date().toISOString();
        const tournament: Tournament = {
            id: crypto.randomUUID(),
            name,
            date,
            courts,
            gender,
            ageCategory,
            format,
            athletes: [],
            matches: [],
            courtAssignment: {},
            leagueId: leagueContext?.leagueId ?? null,
            stageOrder: leagueContext?.stageOrder ?? null,
            weekId: leagueContext?.weekId ?? null,
            teamTournamentId: null,
            createdAt: now,
            updatedAt: now,
        };
        tournaments.value.push(tournament);
        save();
        return tournament;
    }

    function selectTournament(id: string) { currentTournamentId.value = id }

    function deleteTournament(tournamentId: string) {
        const index = tournaments.value.findIndex((t) => t.id === tournamentId)
        if (index !== -1) {
            const tournament = tournaments.value[index]

            if (
                tournament.leagueId &&
                typeof tournament.stageOrder === 'number' &&
                tournament.weekId
            ) {
                const leagueStore = useLeagueStore()
                leagueStore.unlinkWeekFromTournament(
                    tournament.leagueId,
                    tournament.stageOrder,
                    tournament.weekId
                )
            }

            tournaments.value.splice(index, 1)

            if (currentTournamentId.value === tournamentId) {
                currentTournamentId.value = null
            }

            save()
            remote(() => webApi().call(`/tournaments/${tournamentId}`, 'DELETE'))
        }
    }
    // ─────────────────────────── ورزشکار ───────────────────────────

    function addAthlete(tournamentId: string, athlete: Omit<Athlete, "id">): { error?: string } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { error: 'مسابقه یافت نشد' }
        if (!athlete.name.trim() || !weightOptions(t.ageCategory, t.gender).includes(athlete.weightCategory))
            return { error: 'نام یا دسته وزنی نامعتبر است' }
        if (athlete.ranking != null && (!Number.isInteger(athlete.ranking) || athlete.ranking <= 0))
            return { error: 'رنک باید عدد صحیح مثبت باشد' }
        if (t.matches.some(m => m.weightCategory === athlete.weightCategory))
            return { error: 'ابتدا براکت این وزن را پاک کنید' }

        if (athlete.ranking != null) {
            const duplicate = t.athletes.find(
                (a) => a.weightCategory === athlete.weightCategory && a.ranking === athlete.ranking
            )
            if (duplicate) return { error: `رنک ${athlete.ranking} قبلاً برای ${duplicate.name} در این وزن ثبت شده` }
        }

        const created = {
            ...athlete,
            weighIn: athlete.weighIn ?? emptyWeighIn(),
            id: crypto.randomUUID(),
        }
        t.athletes.push(created)
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes`, 'POST', {
            id: created.id, name: created.name, club: created.club,
            weightCategory: created.weightCategory, ranking: created.ranking ?? null,
        }), tournamentId)
        return {}
    }

    function updateAthlete(tournamentId: string, athlete: Athlete): { error?: string } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { error: 'مسابقه یافت نشد' }

        if (athlete.ranking != null) {
            const duplicate = t.athletes.find(
                (a) => a.weightCategory === athlete.weightCategory &&
                    a.ranking === athlete.ranking &&
                    a.id !== athlete.id
            )
            if (duplicate) return { error: `رنک ${athlete.ranking} قبلاً برای ${duplicate.name} در این وزن ثبت شده` }
        }

        const idx = t.athletes.findIndex((a) => a.id === athlete.id)
        if (idx === -1) return { error: 'ورزشکار یافت نشد' }
        if (!athlete.name.trim() || !weightOptions(t.ageCategory, t.gender).includes(athlete.weightCategory))
            return { error: 'نام یا دسته وزنی نامعتبر است' }
        if (athlete.ranking != null && (!Number.isInteger(athlete.ranking) || athlete.ranking <= 0))
            return { error: 'رنک باید عدد صحیح مثبت باشد' }
        const previous = t.athletes[idx]
        if (previous.weightCategory !== athlete.weightCategory &&
            ((previous.weighIn?.attempts.length ?? 0) > 0 ||
             previous.weighIn?.status === 'passed' || previous.weighIn?.status === 'failed' ||
             t.matches.some(m => m.athlete1Id === athlete.id || m.athlete2Id === athlete.id || m.weightCategory === athlete.weightCategory)))
            return { error: 'پس از وزن‌کشی یا قرعه‌کشی تغییر وزن مجاز نیست' }
        if (idx !== -1) {
            t.athletes[idx] = {
                ...athlete,
                // وزن‌کشی از فرم ویرایش دست‌کاری نمی‌شود؛ فقط از متدهای اختصاصی
                weighIn: t.athletes[idx].weighIn ?? athlete.weighIn ?? emptyWeighIn(),
            }
        }
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes/${athlete.id}`, 'PUT', {
            name: athlete.name, club: athlete.club, weightCategory: athlete.weightCategory,
            ranking: athlete.ranking ?? null,
        }), tournamentId)
        return {}
    }

    function removeAthlete(tournamentId: string, athleteId: string): { error?: string } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { error: 'مسابقه یافت نشد' }
        if (t.matches.some(m => m.athlete1Id === athleteId || m.athlete2Id === athleteId || m.winnerId === athleteId))
            return { error: 'ورزشکار در براکت حضور دارد؛ ابتدا براکت را پاک کنید' }
        t.athletes = t.athletes.filter((a) => a.id !== athleteId)
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes/${athleteId}`, 'DELETE'), tournamentId)
        return {}
    }

    function resetRankings(tournamentId: string, weightCategory?: string): { count: number } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { count: 0 }

        let count = 0
        t.athletes = t.athletes.map((a) => {
            if (weightCategory && a.weightCategory !== weightCategory) return a
            if (a.ranking == null) return a
            count++
            const { ranking, ...rest } = a
            return rest as Athlete
        })

        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/rankings/reset`, 'POST', {
            weightCategory: weightCategory ?? '',
        }), tournamentId)
        return { count }
    }

    // ─────────────────────────── بازی‌ها ───────────────────────────

    function setMatches(tournamentId: string, matches: Match[]) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return
        t.matches = matches
        save()
    }

    function updateMatch(tournamentId: string, match: Match) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return
        const idx = t.matches.findIndex((m) => m.id === match.id)
        if (idx !== -1) t.matches[idx] = match
        save()
    }

    function updateMatchResult(tournamentId: string, matchId: string, winnerId: string) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return
        t.matches = recordResult(t.matches, matchId, winnerId)
        save()
    }

    function findNextMatch(
        matches: Match[], m: Match
    ): { match: Match; slot: 1 | 2 } | null {
        const link = (m as any).nextMatchId as string | undefined
        const bracketOrder = (x: Match) =>
            (x as any).bracketIndex ?? (x as any).slotIndex ?? x.order

        const sameCat = (x: Match) => x.weightCategory === m.weightCategory

        const siblings = matches
            .filter((x) => sameCat(x) && x.round === m.round)
            .sort((a, b) => bracketOrder(a) - bracketOrder(b))
        const idx = siblings.findIndex((x) => x.id === m.id)
        if (idx === -1) return null

        if (link) {
            const target = matches.find((x) => x.id === link)
            if (target) return { match: target, slot: idx % 2 === 0 ? 1 : 2 }
        }

        const nextRound = matches
            .filter((x) => sameCat(x) && x.round === m.round + 1)
            .sort((a, b) => bracketOrder(a) - bracketOrder(b))
        const target = nextRound[Math.floor(idx / 2)]
        if (!target) return null

        return { match: target, slot: idx % 2 === 0 ? 1 : 2 }
    }

    /** پاک‌سازی آبشاری: حذف ورزشکار از مراحل بعد و ریست نتایج آن‌ها */
    function clearDownstream(matches: Match[], from: Match, athleteId: string | null) {
        if (!athleteId) return
        let cursor: Match | null = from
        let removing = athleteId

        while (cursor) {
            const next = findNextMatch(matches, cursor)
            if (!next) return

            const nm = next.match
            const key = next.slot === 1 ? 'athlete1Id' : 'athlete2Id'
            if (nm[key] !== removing) return   // این شاخه دیگر آلوده نیست

            const wasWinner = nm.winnerId === removing
            nm[key] = null
            nm.winnerId = ""
            nm.status = 'pending'
            delete nm.result

            if (!wasWinner) return
            cursor = nm
            removing = removing   // همان ورزشکار در ادامه مسیر برنده بوده
        }
    }

    /** انتقال برنده به مرحله بعد */
    function propagateWinner(matches: Match[], m: Match) {
        if (!m.winnerId) return
        const next = findNextMatch(matches, m)
        if (!next) return
        const key = next.slot === 1 ? 'athlete1Id' : 'athlete2Id'
        next.match[key] = m.winnerId
    }

    /** ساخت نتیجه خالی برای فرم ورود امتیاز */
    function createEmptyResult(match: Match, roundCount = 2): MatchResult {
        return {
            winType: 'PTF',
            blueId: match.athlete1Id,
            redId: match.athlete2Id,
            winnerId: null,
            rounds: Array.from({ length: roundCount }, (_, i) => emptyRound(i + 1)),
            recordedAt: new Date().toISOString(),
        }
    }

    function startMatch(tournamentId: string, matchId: string) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        const m = t?.matches.find((x) => x.id === matchId)
        if (!t || !m || m.isBye) return
        if (m.status === 'completed' || m.status === 'ongoing' || m.winnerId) return
        if (!m.athlete1Id || !m.athlete2Id || m.athlete1Id === m.athlete2Id) return
        if ([m.athlete1Id, m.athlete2Id].some(id => !t.athletes.some(a => a.id === id && a.weightCategory === m.weightCategory && a.weighIn?.status === 'passed'))) return
        if (t.matches.some(other => other.nextMatchId === m.id && !other.winnerId)) return
        if (t.matches.some(other => other.id !== m.id && other.status === 'ongoing' &&
            (other.court === m.court || other.athlete1Id === m.athlete1Id || other.athlete1Id === m.athlete2Id ||
             other.athlete2Id === m.athlete1Id || other.athlete2Id === m.athlete2Id))) return
        m.status = 'ongoing'
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/matches/${matchId}/start`, 'POST'), tournamentId)
    }

    /**
     * ثبت یا ویرایش نتیجه یک بازی.
     * محاسبه‌ی برنده راند و مسابقه خودکار است، مگر winType غیرامتیازی باشد.
     */
    function setMatchResult(
        tournamentId: string,
        matchId: string,
        input: {
            winType: WinType
            rounds?: MatchRound[]
            /** برای WDR/DSQ/RSC که برنده صریح داده می‌شود */
            winnerId?: string | null
            note?: string
            refereeId?: string
            /** جابه‌جایی کرنر نسبت به athlete1/2 */
            swapCorners?: boolean
        },
        syncRemote = true,
    ): { error?: string } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { error: 'مسابقه یافت نشد' }

        const matches = t.matches.map((m) => ({ ...m }))
        const m = matches.find((x) => x.id === matchId)
        if (!m) return { error: 'بازی یافت نشد' }
        if (m.isBye) return { error: 'بازی استراحت نتیجه ندارد' }
        if (!m.athlete1Id || !m.athlete2Id) return { error: 'هر دو ورزشکار مشخص نشده‌اند' }

        const blueId = input.swapCorners ? m.athlete2Id : m.athlete1Id
        const redId = input.swapCorners ? m.athlete1Id : m.athlete2Id

        let winnerId: string | null = null
        let rounds: MatchRound[] = []
        let winType = input.winType
        let winnerCorner: 'blue' | 'red' | undefined

        if (isNonScoreWin(winType)) {
            if (!input.winnerId) return { error: `برای ${winType} انتخاب برنده الزامی است` }
            if (input.winnerId !== blueId && input.winnerId !== redId)
                return { error: 'برنده انتخاب‌شده در این بازی حاضر نیست' }
            winnerId = input.winnerId
            winnerCorner = winnerId === blueId ? 'blue' : 'red'
            rounds = normalizeRounds(input.rounds ?? [], DEFAULT_RULES)
        } else {
            if (!input.rounds?.length) return { error: 'امتیاز راندها وارد نشده است' }

            if (input.rounds.length > DEFAULT_RULES.maxRounds || input.rounds.some((r, i) => r.number !== i + 1))
                return { error: `حداکثر ${DEFAULT_RULES.maxRounds} راند با شماره‌های متوالی مجاز است` }
            const invalid = input.rounds.some((r) =>
                Object.values(r.blue).concat(Object.values(r.red))
                    .some((v) => typeof v === 'number' && (v < 0 || !Number.isInteger(v)))
            )
            if (invalid) return { error: 'امتیازها باید عدد صحیح و نامنفی باشند' }

            rounds = normalizeRounds(input.rounds, DEFAULT_RULES)
            const outcome = resolveMatch(rounds, DEFAULT_RULES)

            if (!outcome.decided || !outcome.winnerCorner)
                return { error: 'با این امتیازها برنده مشخص نمی‌شود' }

            winnerCorner = outcome.winnerCorner
            winnerId = winnerCorner === 'blue' ? blueId : redId
            winType = outcome.winType ?? winType
        }

        const previousWinner = m.winnerId ?? null
        if (previousWinner !== winnerId && hasLockedDownstream(matches, m))
            return { error: 'مرحله بعد شروع شده یا نتیجه دارد؛ ابتدا نتیجه مراحل بعد را اصلاح کنید' }
        if (previousWinner && previousWinner !== winnerId) {
            clearDownstream(matches, m, previousWinner)
        }

        m.result = {
            winType,
            blueId,
            redId,
            winnerId,
            winnerCorner,
            rounds,
            note: input.note,
            refereeId: input.refereeId,
            recordedAt: m.result?.recordedAt ?? new Date().toISOString(),
            updatedAt: m.result ? new Date().toISOString() : undefined,
        }
        m.winnerId = winnerId
        m.status = 'completed'

        propagateWinner(matches, m)
        setMatches(tournamentId, matches)
        const resultData = m.result!
        if (syncRemote) remote(() => webApi().call(`/tournaments/${tournamentId}/matches/${matchId}/result`, 'POST', {
            winType: resultData.winType, blueId: resultData.blueId, redId: resultData.redId,
            winnerId: resultData.winnerId, winnerCorner: resultData.winnerCorner,
            rounds: resultData.rounds, note: resultData.note ?? null, refereeId: resultData.refereeId ?? null,
        }), tournamentId)
        return {}
    }

    async function submitMatchResult(
        tournamentId: string,
        matchId: string,
        input: { winType: WinType; rounds?: MatchRound[]; winnerId?: string | null; note?: string; refereeId?: string; swapCorners?: boolean },
    ): Promise<{ error?: string }> {
        const validation = setMatchResult(tournamentId, matchId, input, false)
        if (validation.error) return validation
        const tournament = tournaments.value.find(t => t.id === tournamentId)
        const result = tournament?.matches.find(m => m.id === matchId)?.result
        if (!result) return { error: 'نتیجه بازی ساخته نشد' }
        try {
            await webApi().call(`/tournaments/${tournamentId}/matches/${matchId}/result`, 'POST', {
                winType: result.winType, blueId: result.blueId, redId: result.redId,
                winnerId: result.winnerId, winnerCorner: result.winnerCorner,
                rounds: result.rounds, note: result.note ?? null, refereeId: result.refereeId ?? null,
            })
            await refreshFromServer(tournamentId)
            return {}
        } catch (error: any) {
            try { await refreshFromServer(tournamentId) } catch { /* keep the server error */ }
            return { error: error?.message || 'ثبت نتیجه در سرور ناموفق بود' }
        }
    }

    /** حذف نتیجه یک بازی و پاک‌سازی آبشاری مراحل بعد */
    function clearMatchResult(tournamentId: string, matchId: string): { error?: string } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { error: 'مسابقه یافت نشد' }

        const matches = t.matches.map((m) => ({ ...m }))
        const m = matches.find((x) => x.id === matchId)
        if (!m) return { error: 'بازی یافت نشد' }

        if (m.isBye) return { error: 'نتیجه بازی استراحت قابل حذف نیست' }
        if (hasLockedDownstream(matches, m))
            return { error: 'مرحله بعد شروع شده یا نتیجه دارد؛ ابتدا نتیجه مراحل بعد را اصلاح کنید' }
        clearDownstream(matches, m, m.winnerId ?? null)

        m.winnerId = ""
        m.status = 'pending'
        delete m.result

        setMatches(tournamentId, matches)
        remote(() => webApi().call(`/tournaments/${tournamentId}/matches/${matchId}/result`, 'DELETE'), tournamentId)
        return {}
    }

    // ──────────────────── تخصیص زمین (هسته مشترک) ────────────────────

    /** آمار هر وزن برای موتور توزیع بار؛ وزن‌های شروع‌شده یا دستی‌جابه‌جا‌شده قفل می‌شوند */
    function buildCategoryStats(matches: Match[]): CategoryStat[] {
        const cats = [...new Set(matches.map((m) => m.weightCategory))]
        return cats.map((cat) => {
            const playable = matches.filter((m) => m.weightCategory === cat && !m.isBye)
            const started = playable.some((m) => m.winnerId || m.status === 'ongoing')
            const moved = playable.some((m) => m.movedAt)
            const pinnedCourts = [...new Set(playable.map((m) => m.court).filter((c) => c >= 1))]
                .sort((a, b) => a - b)

            return {
                cat,
                load: playable.length,
                firstRoundPlayable: playable.filter((m) => m.round === 1).length,
                pinned: (started || moved) && pinnedCourts.length > 0,
                pinnedCourts,
            }
        })
    }

    /** بازمحاسبه کامل تخصیص زمین + ترتیب برای همه وزن‌ها */
    function applyCourtPlan(t: Tournament, matches: Match[]) {
        const MAX_COURTS_PER_CATEGORY = 2

        const plan = planCourts(buildCategoryStats(matches), {
            courts: t.courts,
            maxCourtsPerCategory: MAX_COURTS_PER_CATEGORY,
        })


        const lockedSet = new Set(plan.locked)
        for (const [cat, courts] of Object.entries(plan.assignment)) {
            if (lockedSet.has(cat)) continue
            assignCategoryCourts(matches.filter((m) => m.weightCategory === cat), courts)
        }

        syncOrders(matches)
        t.courtAssignment = plan.assignment
    }

    /** بازچینش زمین‌ها بدون قرعه‌کشی مجدد (مثلاً بعد از تغییر تعداد زمین) */
    function rebalanceCourts(tournamentId: string) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t || !t.matches.length) return
        const matches = t.matches.map((m) => ({ ...m }))
        applyCourtPlan(t, matches)
        setMatches(tournamentId, matches)
        remote(() => webApi().call(`/tournaments/${tournamentId}/rebalance`, 'POST'), tournamentId)
    }

    // ─────────────────────────── قرعه‌کشی ───────────────────────────

    function drawBracket(tournamentId: string, drawType: 'random' | 'ranked' = 'random') {
        const id = tournamentId ?? currentTournamentId.value
        if (!id) return
        const t = tournaments.value.find((x) => x.id === id)
        if (!t) return

        if (t.matches.some(m => !m.isBye && (m.winnerId || m.result || m.status === 'ongoing')))
            return { error: 'بازی شروع شده یا نتیجه دارد؛ قرعه‌کشی مجدد مجاز نیست' }
        if (t.athletes.some(a => a.weighIn?.status !== 'passed' && a.weighIn?.status !== 'failed'))
            return { error: 'ابتدا وزن‌کشی همه ورزشکاران را تکمیل کنید' }
        const byCategory = t.athletes
            .filter((a) => a.weighIn?.status === 'passed')
            .reduce<Record<string, Athlete[]>>((acc, a) => {
                ;(acc[a.weightCategory] ??= []).push(a)
                return acc
            }, {})

        const allMatches: Match[] = []
        for (const [cat, group] of Object.entries(byCategory)) {
            allMatches.push(...generateBracket(group, cat, 0, drawType))
        }

        applyCourtPlan(t, allMatches)
        setMatches(id, allMatches)
        remote(() => webApi().call(`/tournaments/${id}/draw`, 'POST', { type: drawType }), id)
    }

    function drawBracketForCategory(
        tournamentId: string,
        weightCategory: string,
        drawType: 'random' | 'ranked' = 'random'
    ) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return

        if (t.matches.some(m => m.weightCategory === weightCategory && !m.isBye && (m.winnerId || m.result || m.status === 'ongoing')))
            return { error: 'این وزن بازی شروع‌شده یا نتیجه‌دار دارد؛ قرعه‌کشی مجدد مجاز نیست' }
        if (!isCategoryWeighInComplete(tournamentId, weightCategory))
            return { error: 'ابتدا وزن‌کشی این وزن را تکمیل کنید' }
        // فقط ورزشکاران قبول‌شده در وزن‌کشی
        const athletes = t.athletes.filter(
            (a) =>
                a.weightCategory === weightCategory &&
                a.weighIn?.status === 'passed'
        )

        // مسابقات سایر وزن‌ها بدون تغییر باقی می‌مانند
        const others = t.matches.filter(
            (m) => m.weightCategory !== weightCategory
        )

        // کمتر از دو ورزشکار مجاز: براکت جدید ساخته نمی‌شود.
        // براکت قبلی این وزن نیز در صورت وجود حذف می‌شود.
        const fresh = athletes.length >= 2
            ? generateBracket(athletes, weightCategory, 0, drawType)
            : []

        const allMatches = [...others, ...fresh]

        applyCourtPlan(t, allMatches)
        setMatches(tournamentId, allMatches)
        remote(() => webApi().call(`/tournaments/${tournamentId}/draw`, 'POST', { type: drawType, weightCategory }), tournamentId)
    }


    // ─────────────────────── جابه‌جایی دستی زمین ───────────────────────

    function reassignCourt(tournamentId: string, weightCategory: string, newCourt: number) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return
        if (!t.courtAssignment) t.courtAssignment = {}

        const oldCourts = t.courtAssignment[weightCategory] ?? []

        t.matches = t.matches.map((m) =>
            m.weightCategory === weightCategory && !m.winnerId && !m.isBye
                ? { ...m, court: newCourt, movedAt: new Date().toISOString() }
                : m
        )
        t.courtAssignment[weightCategory] = [newCourt]

        const affectedCourts = new Set([newCourt, ...oldCourts])
        for (const c of affectedCourts) {
            let ord = 1
            for (const m of t.matches
                .filter((m) => m.court === c && !m.isBye)
                .sort((a, b) => a.order - b.order)) {
                m.order = ord++
            }
        }
        save()
    }

    function swapCourtCategories(tournamentId: string, catA: string, catB: string): { warning: boolean } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { warning: false }
        if (!t.courtAssignment) t.courtAssignment = {}

        const countA = t.athletes.filter((a) => a.weightCategory === catA).length
        const countB = t.athletes.filter((a) => a.weightCategory === catB).length
        const warning = countA > 0 && countB > 0 &&
            Math.abs(countA - countB) / Math.max(countA, countB) > 0.35

        const courtsA = t.courtAssignment[catA] ?? []
        const courtsB = t.courtAssignment[catB] ?? []
        const now = new Date().toISOString()

        t.matches = t.matches.map((m) => {
            if (m.weightCategory === catA && !m.winnerId && !m.isBye) {
                return { ...m, court: courtsB[0] ?? m.court, movedAt: now }
            }
            if (m.weightCategory === catB && !m.winnerId && !m.isBye) {
                return { ...m, court: courtsA[0] ?? m.court, movedAt: now }
            }
            return m
        })

        t.courtAssignment[catA] = courtsB
        t.courtAssignment[catB] = courtsA

        syncOrders(t.matches)
        save()
        return { warning }
    }

    function moveMatch(tournamentId: string, matchId: string, newCourt: number, newOrder: number) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return

        const match = t.matches.find((m) => m.id === matchId)
        if (!match || match.winnerId || match.isBye) return
        if (!Number.isFinite(newCourt) || newCourt < 1 || newCourt > t.courts) return

        const oldCourt = match.court
        const oldOrder = match.order
        const sameCourt = oldCourt === newCourt

        const destinationMatches = t.matches
            .filter((m) => m.court === newCourt && !m.isBye && m.id !== match.id)
            .sort((a, b) => a.order - b.order)

        const safeOrder = Math.max(1, Math.min(Math.floor(newOrder || 1), destinationMatches.length + 1))

        if (!sameCourt) {
            match.sourceCourt = oldCourt
            match.sourceOrder = oldOrder
            match.sourceLabel = `زمین ${oldCourt} - بازی ${oldOrder}`
        }
        match.movedAt = new Date().toISOString()

        match.court = newCourt
        destinationMatches.splice(safeOrder - 1, 0, match)
        destinationMatches.forEach((m, i) => { m.order = i + 1 })

        if (!sameCourt) {
            t.matches
                .filter((m) => m.court === oldCourt && !m.isBye && m.id !== match.id)
                .sort((a, b) => a.order - b.order)
                .forEach((m, i) => { m.order = i + 1 })
        }

        const categoryCourts = [...new Set(
            t.matches
                .filter((m) => m.weightCategory === match.weightCategory && !m.isBye)
                .map((m) => m.court)
        )].sort((a, b) => a - b)

        if (!t.courtAssignment) t.courtAssignment = {}
        t.courtAssignment[match.weightCategory] = categoryCourts

        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/matches/${matchId}/move`, 'POST', {
            court: newCourt, order: safeOrder, sourceLabel: match.sourceLabel ?? null,
        }), tournamentId)
    }

    // ─────────────────────── جابه‌جایی ورزشکار ───────────────────────

    function swapAthletes(
        a: { matchId: string; slot: 1 | 2 },
        b: { matchId: string; slot: 1 | 2 }
    ) {
        const t = currentTournament.value
        if (!t) return

        const idxA = t.matches.findIndex((m) => m.id === a.matchId)
        const idxB = t.matches.findIndex((m) => m.id === b.matchId)
        if (idxA === -1 || idxB === -1) return

        const ma = t.matches[idxA]
        const mb = t.matches[idxB]

        if (ma.round !== 1 || mb.round !== 1) return
        if (ma.winnerId || mb.winnerId) return
        if (ma.weightCategory !== mb.weightCategory) return

        const keyA = a.slot === 1 ? 'athlete1Id' : 'athlete2Id'
        const keyB = b.slot === 1 ? 'athlete1Id' : 'athlete2Id'

        if (idxA === idxB && keyA === keyB) return
        if (!ma[keyA] || !mb[keyB]) return

        const athleteMap = new Map(t.athletes.map((x) => [x.id, x]))
        const getClub = (id: string | null | undefined) =>
            id ? athleteMap.get(id)?.club ?? null : null

        const valA = ma[keyA] as string | null
        const valB = mb[keyB] as string | null

        const otherA = keyA === 'athlete1Id' ? ma.athlete2Id : ma.athlete1Id
        const otherB = keyB === 'athlete1Id' ? mb.athlete2Id : mb.athlete1Id

        const clubA = getClub(valA)
        const clubB = getClub(valB)
        const clubOtherA = getClub(otherA)
        const clubOtherB = getClub(otherB)

        const willBreakA = clubA != null && clubA === clubOtherB
        const willBreakB = clubB != null && clubB === clubOtherA
        if (willBreakA || willBreakB) return

        if (idxA === idxB) {
            t.matches[idxA] = { ...ma, [keyA]: valB, [keyB]: valA }
        } else {
            t.matches[idxA] = { ...ma, [keyA]: valB }
            t.matches[idxB] = { ...mb, [keyB]: valA }
        }

        save()
    }

    // ─────────────────────── اکسپورت / ایمپورت اکسل ───────────────────────

    /** خروجی اکسل: شیت Data مرجع کامل (کل تورنمنت به‌صورت JSON)، بقیه شیت‌ها فقط برای مشاهده */
    function exportTournamentToExcel(tournamentId: string): { error?: string } {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return { error: "مسابقه یافت نشد" }

        try {
            const wb = XLSX.utils.book_new()

            // ── شیت Data: کل تورنمنت، تکه‌تکه‌شده تا از سقف ۳۲۷۶۷ کاراکتری سلول اکسل رد نشود
            const raw = JSON.stringify(t)
            const CHUNK = 30000
            const dataRows: { seq: number; part: string }[] = []
            for (let i = 0; i < raw.length; i += CHUNK) {
                dataRows.push({ seq: dataRows.length, part: raw.slice(i, i + CHUNK) })
            }
            XLSX.utils.book_append_sheet(wb, XLSX.utils.json_to_sheet(dataRows), "Data")

            // ── شیت Info: فقط برای خواندن توسط انسان
            XLSX.utils.book_append_sheet(wb, XLSX.utils.json_to_sheet([{
                name: t.name,
                date: t.date,
                courts: t.courts,
                gender: t.gender,
                ageCategory: t.ageCategory,
                athleteCount: t.athletes.length,
                matchCount: t.matches.length,
            }]), "Info")

            // ── شیت Athletes: خوانا
            XLSX.utils.book_append_sheet(wb, XLSX.utils.json_to_sheet(
                t.athletes.map((a) => ({
                    name: a.name,
                    club: a.club,
                    weightCategory: a.weightCategory,
                    ranking: a.ranking ?? "",
                    weighInStatus: a.weighIn?.status ?? "pending",
                    weightKg: a.weighIn?.weightKg ?? "",
                    withTolerance: a.weighIn?.withTolerance ? 1 : 0,
                    attempts: a.weighIn?.attempts.length ?? 0,
                    signedAt: a.weighIn?.signature?.signedAt ?? "",
                }))
            ), "Athletes")

            // ── شیت Matches: خوانا
            const athleteName = (id: string | null | undefined) =>
                t.athletes.find((a) => a.id === id)?.name ?? ""
            const sumScore = (s: any) =>
                (s?.punch ?? 0) + (s?.bodyKick ?? 0) + (s?.headKick ?? 0) +
                (s?.turningBodyKick ?? 0) + (s?.turningHeadKick ?? 0)

            XLSX.utils.book_append_sheet(wb, XLSX.utils.json_to_sheet(
                t.matches.map((m) => ({
                    weightCategory: m.weightCategory,
                    round: m.round,
                    side: m.side,
                    court: m.court,
                    order: m.order,
                    isBye: m.isBye ? 1 : 0,
                    status: m.status ?? "",
                    athlete1: athleteName(m.athlete1Id),
                    athlete2: athleteName(m.athlete2Id),
                    winner: athleteName(m.winnerId),
                    winType: m.result?.winType ?? "",
                    roundScores: (m.result?.rounds ?? [])
                        .map((r) => `${sumScore(r.blue)}-${sumScore(r.red)}`)
                        .join(" | "),
                }))
            ), "Matches")

            const safeName = t.name.replace(/[\\/:*?"<>|]/g, "-")
            XLSX.writeFile(wb, `${safeName}-${t.date}.xlsx`)
            return {}
        } catch {
            return { error: "ساخت فایل اکسل ناموفق بود" }
        }
    }

    /** ایمپورت از اکسل؛ داده‌ها را در تورنمنت جاری جایگزین می‌کند و همان را نمایش می‌دهد */
    async function importTournamentFromExcel(
        file: File,
        targetTournamentId?: string
    ): Promise<{ tournament?: Tournament; error?: string }> {
        let parsed: any = null

        try {
            const buf = await file.arrayBuffer()
            const wb = XLSX.read(buf, { type: "array" })

            if (wb.Sheets["Data"]) {
                // نسخه جدید: کل تورنمنت در شیت Data
                const rows = XLSX.utils.sheet_to_json<any>(wb.Sheets["Data"])
                if (!rows.length) return { error: "شیت Data خالی است" }
                const raw = rows
                    .sort((a, b) => Number(a.seq) - Number(b.seq))
                    .map((r) => String(r.part ?? ""))
                    .join("")
                parsed = JSON.parse(raw)
            } else if (wb.Sheets["Athletes"] && wb.Sheets["Matches"]) {
                // سازگاری با فایل‌های نسخه قبلی که athleteJson/matchJson داشتند
                const infoRows = XLSX.utils.sheet_to_json<any>(wb.Sheets["Info"] ?? {})
                const athleteRows = XLSX.utils.sheet_to_json<any>(wb.Sheets["Athletes"])
                const matchRows = XLSX.utils.sheet_to_json<any>(wb.Sheets["Matches"])

                if (athleteRows.some((r) => !r.athleteJson) || matchRows.some((r) => !r.matchJson)) {
                    return { error: "این فایل با نسخهٔ قدیمی اکسپورت شده و اطلاعات کامل براکت را ندارد؛ مسابقه را با نسخهٔ جدید دوباره اکسپورت بگیرید" }
                }

                const info = infoRows[0] ?? {}
                parsed = {
                    name: info.name,
                    date: info.date,
                    courts: info.courts,
                    gender: info.gender,
                    ageCategory: info.ageCategory,
                    athletes: athleteRows.map((r) => JSON.parse(String(r.athleteJson))),
                    matches: matchRows.map((r) => JSON.parse(String(r.matchJson))),
                    courtAssignment: info.courtAssignment ? JSON.parse(String(info.courtAssignment)) : {},
                }
            } else {
                return { error: "ساختار فایل شناخته نشد؛ این فایل خروجی برنامه نیست" }
            }
        } catch {
            return { error: "خواندن فایل اکسل ناموفق بود؛ فایل خراب یا نامعتبر است" }
        }

        if (!parsed || !Array.isArray(parsed.athletes) || !Array.isArray(parsed.matches)) {
            return { error: "اطلاعات ورزشکاران یا بازی‌ها در فایل یافت نشد" }
        }

        const targetId = targetTournamentId ?? currentTournamentId.value
        const existing = targetId ? tournaments.value.find((x) => x.id === targetId) : null

        if (existing) {
            // جایگزینی درجا: id تورنمنت و اتصال لیگ (leagueId/stageOrder/weekId) حفظ می‌شود.
            // idهای ورزشکاران و بازی‌ها عیناً از فایل می‌آیند تا ارجاع‌های براکت
            // (nextMatchId, nextSlot, winnerId, result و...) دقیقاً سالم بمانند.
            existing.name = String(parsed.name ?? existing.name)
            existing.date = String(parsed.date ?? existing.date)
            existing.courts = Number(parsed.courts) || existing.courts
            existing.gender = (parsed.gender ?? existing.gender) as Gender
            existing.ageCategory = (parsed.ageCategory ?? existing.ageCategory) as AgeCategory
            existing.athletes = (parsed.athletes as any[]).map(migrateWeighIn)
            existing.matches = parsed.matches
            existing.courtAssignment = parsed.courtAssignment ?? {}
            existing.updatedAt = new Date().toISOString()

            currentTournamentId.value = existing.id
            save()
            return { tournament: existing }
        }

        // تورنمنت جاری وجود نداشت: رکورد جدید ساخته و همان انتخاب می‌شود
        const created: Tournament = {
            ...parsed,
            athletes: (parsed.athletes as any[]).map(migrateWeighIn),
            id: crypto.randomUUID(),
            name: String(parsed.name ?? "مسابقه ایمپورت‌شده"),
            date: String(parsed.date ?? ""),
            courts: Number(parsed.courts) || 1,
            courtAssignment: parsed.courtAssignment ?? {},
            leagueId: null,
            stageOrder: null,
            weekId: null,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        }
        tournaments.value.push(created)
        currentTournamentId.value = created.id
        save()
        return { tournament: created }
    }

    // ─────────────────────── جابه‌جایی موقعیت بازی‌ها ───────────────────────

    function swapMatchPositions(tournamentId: string, matchIdA: string, matchIdB: string) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return
        const a = t.matches.find((m) => m.id === matchIdA)
        const b = t.matches.find((m) => m.id === matchIdB)
        if (!a || !b) return
        const tmpCourt = a.court
        const tmpOrder = a.order
        a.court = b.court
        a.order = b.order
        b.court = tmpCourt
        b.order = tmpOrder
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/matches/${matchIdA}/swap-position`, 'POST', {
            otherMatchId: matchIdB,
        }), tournamentId)
    }

// ─────────────────────── مرحله حذفی ───────────────────────

    function addEliminationMatch(tournamentId: string, match: EliminationMatch) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t) return
        if (!t.eliminationMatches) t.eliminationMatches = []
        t.eliminationMatches.push(match)
        save()
    }

    function removeEliminationMatch(tournamentId: string, matchId: string) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t || !t.eliminationMatches) return
        t.eliminationMatches = t.eliminationMatches.filter((m) => m.id !== matchId)
        save()
    }

    function setEliminationWinner(tournamentId: string, matchId: string, athleteLabel: string) {
        const t = tournaments.value.find((x) => x.id === tournamentId)
        if (!t || !t.eliminationMatches) return
        const m = t.eliminationMatches.find((m) => m.id === matchId)
        if (m) m.winnerId = athleteLabel
        save()
    }

    function migrateWeighIn(a: Athlete): Athlete { return migrateAthleteWeighIn(a) }


    // ─────────────────────────── وزن‌کشی ───────────────────────────

    function findAthlete(tournamentId: string, athleteId: string) {
        const t = tournaments.value.find((item) => item.id === tournamentId)

        if (!t) {
            return {
                error: 'مسابقه یافت نشد' as const,
            }
        }

        const a = t.athletes.find((item) => item.id === athleteId)

        if (!a) {
            return {
                error: 'ورزشکار یافت نشد' as const,
            }
        }

        if (!a.weighIn) {
            a.weighIn = emptyWeighIn()
        }

        return { t, a }
    }

    /**
     * فهرست دسته‌های وزنی معتبر برای ورزشکار.
     *
     * فرض شده است:
     * - سن روی مسابقه در t.ageCategory قرار دارد.
     * - جنسیت روی ورزشکار در a.gender قرار دارد.
     */
    function getAthleteWeightCategories(
        tournament: Tournament,
        _athlete: Athlete,
    ): readonly string[] {
        return weightOptions(
            tournament.ageCategory,
            tournament.gender,
        )
    }

    /** پیش‌نمایش قبل از ثبت: داور عدد را می‌بیند و بعد تأیید می‌کند. */
    function previewWeighIn(
        tournamentId: string,
        athleteId: string,
        weightKg: number,
    ) {
        const found = findAthlete(tournamentId, athleteId)

        if ('error' in found) {
            return { error: found.error }
        }

        const { t, a } = found
        const categories = getAthleteWeightCategories(t, a)

        if (categories.length === 0) {
            return {
                error: 'برای سن و جنسیت این ورزشکار دسته وزنی تعریف نشده است',
            }
        }

        if (!categories.includes(a.weightCategory)) {
            return {
                error: `دسته وزنی ${a.weightCategory} برای سن و جنسیت این ورزشکار معتبر نیست`,
            }
        }

        return checkWeight(
            weightKg,
            a.weightCategory,
            categories,
        )
    }

    /** ثبت یک نوبت وزن‌کشی */
    function recordWeighIn(
        tournamentId: string,
        athleteId: string,
        weightKg: number,
    ): { error?: string; weighIn?: WeighIn; message?: string } {
        const found = findAthlete(tournamentId, athleteId)

        if ('error' in found) {
            return { error: found.error }
        }

        const { t, a } = found
        const categories = getAthleteWeightCategories(t, a)

        if (!Number.isFinite(weightKg) || weightKg <= 0) {
            return {
                error: 'وزن نامعتبر است',
            }
        }

        if (categories.length === 0) {
            return {
                error: 'برای سن و جنسیت این ورزشکار دسته وزنی تعریف نشده است',
            }
        }

        if (!categories.includes(a.weightCategory)) {
            return {
                error: `دسته وزنی ${a.weightCategory} برای سن و جنسیت این ورزشکار معتبر نیست`,
            }
        }

        if (!canWeigh(a.weighIn!)) {
            return {
                error:
                    a.weighIn!.status === 'passed'
                        ? 'وزن این ورزشکار قبلاً تأیید شده'
                        : `هر ${MAX_ATTEMPTS} نوبت مصرف شده و ورزشکار مردود است`,
            }
        }

        const check = checkWeight(
            weightKg,
            a.weightCategory,
            categories,
        )

        a.weighIn = recordAttempt(
            a.weighIn!,
            weightKg,
            a.weightCategory,
            categories,
        )

        a.weighedIn = a.weighIn.status === 'passed'
        t.updatedAt = new Date().toISOString()
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes/${athleteId}/weigh-in`, 'POST', {
            weightKg,
        }), tournamentId)

        const left = attemptsLeft(a.weighIn)

        return {
            weighIn: a.weighIn,
            message: check.ok
                ? check.message
                : `${check.message}${
                    left > 0
                        ? ` — ${left} نوبت باقی مانده`
                        : ' — مردود شد'
                }`,
        }
    }

    /** امضای ورزشکار؛ imagePath مسیر فایلی است که در سمت Rust ذخیره شده است. */
    function signWeighIn(
        tournamentId: string,
        athleteId: string,
        imagePath: string,
    ): { error?: string; weighIn?: WeighIn } {
        const found = findAthlete(tournamentId, athleteId)

        if ('error' in found) {
            return { error: found.error }
        }

        const { t, a } = found

        if (!imagePath?.trim()) {
            return {
                error: 'فایل امضا ذخیره نشده است',
            }
        }

        if (!canSign(a.weighIn!)) {
            return {
                error: a.weighIn!.signature
                    ? 'امضا قبلاً ثبت شده'
                    : 'تا تأیید وزن، امضا امکان‌پذیر نیست',
            }
        }

        a.weighIn = sign(
            a.weighIn!,
            imagePath.trim(),
        )

        t.updatedAt = new Date().toISOString()
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes/${athleteId}/weigh-in/signature`, 'POST', {
            imagePath: imagePath.trim(),
        }), tournamentId)

        return {
            weighIn: a.weighIn,
        }
    }

    /**
     * حذف کامل براکت یک دسته وزنی و ریست وزن‌کشی ورزشکاران همان دسته.
     *
     * این متد برای زمانی است که براکت قبلی معتبر نیست یا باید
     * قرعه‌کشی یک وزن از ابتدا انجام شود.
     */
    function resetCategoryBracketAndWeighIns(
        tournamentId: string,
        weightCategory: string,
    ): {
        error?: string
        removedMatches: number
        resetAthletes: number
    } {
        const t = tournaments.value.find((item) => item.id === tournamentId)

        if (!t) {
            return {
                error: 'مسابقه یافت نشد',
                removedMatches: 0,
                resetAthletes: 0,
            }
        }

        const category = weightCategory?.trim()

        if (!category) {
            return {
                error: 'دسته وزنی نامعتبر است',
                removedMatches: 0,
                resetAthletes: 0,
            }
        }

        const categoryMatches = t.matches.filter(
            (match) => match.weightCategory === category,
        )

        // برای جلوگیری از ریست تصادفی، بدون براکت هیچ تغییری انجام نمی‌شود.
        if (categoryMatches.length === 0) {
            return {
                error: 'براکت این وزن یافت نشد',
                removedMatches: 0,
                resetAthletes: 0,
            }
        }

        const previousMatchCount = t.matches.length

        // حذف تمام مراحل براکت این وزن؛
        // شامل دور اول، مراحل بعدی، فینال و نتایج ذخیره‌شده.
        t.matches = t.matches.filter(
            (match) => match.weightCategory !== category,
        )

        const removedMatches = previousMatchCount - t.matches.length

        let resetAthletes = 0

        // ریست کامل وزن‌کشی همه ورزشکاران این دسته
        t.athletes = t.athletes.map((athlete) => {
            if (athlete.weightCategory !== category) {
                return athlete
            }

            resetAthletes++

            return {
                ...athlete,
                weighedIn: false,
                weighIn: emptyWeighIn(),
            }
        })

        // حذف تخصیص زمین مربوط به براکت حذف‌شده
        if (t.courtAssignment) {
            delete t.courtAssignment[category]
        }

        // بازتنظیم ترتیب بازی‌های باقی‌مانده روی زمین‌ها
        syncOrders(t.matches)

        t.updatedAt = new Date().toISOString()
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/reset-category`, 'POST', {
            weightCategory: category,
        }), tournamentId)

        return {
            removedMatches,
            resetAthletes,
        }
    }

    /**
     * ریست کامل وزن‌کشی با اختیار سرداور.
     * اگر قرعه‌کشی این دسته انجام شده باشد، ریست مجاز نیست.
     */
    function resetWeighIn(
        tournamentId: string,
        athleteId: string,
    ): { error?: string } {
        const found = findAthlete(tournamentId, athleteId)

        if ('error' in found) {
            return { error: found.error }
        }

        const { t, a } = found

        const drawn = t.matches.some(
            (match) =>
                match.weightCategory === a.weightCategory &&
                (
                    match.athlete1Id === a.id ||
                    match.athlete2Id === a.id
                ),
        )

        if (drawn) {
            return {
                error: 'قرعه‌کشی این وزن انجام شده؛ ابتدا براکت را پاک کنید',
            }
        }

        a.weighIn = emptyWeighIn()
        a.weighedIn = false
        t.updatedAt = new Date().toISOString()
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes/${athleteId}/weigh-in`, 'DELETE'), tournamentId)

        return {}
    }

    /** حذف امضا برای امضای مجدد، بدون از بین بردن وزن ثبت‌شده */
    function clearWeighInSignature(
        tournamentId: string,
        athleteId: string,
    ): { error?: string } {
        const found = findAthlete(tournamentId, athleteId)

        if ('error' in found) {
            return { error: found.error }
        }

        const { t, a } = found

        if (!a.weighIn!.signature) {
            return {
                error: 'امضایی ثبت نشده',
            }
        }

        a.weighIn = {
            ...a.weighIn!,
            signature: null,
        }

        t.updatedAt = new Date().toISOString()
        save()
        remote(() => webApi().call(`/tournaments/${tournamentId}/athletes/${athleteId}/weigh-in/signature`, 'DELETE'), tournamentId)

        return {}
    }

    /** آمار وزن‌کشی مسابقه جاری، به تفکیک دسته وزنی */
    const weighInStats = computed(() => {
        const t = currentTournament.value

        const blank = () => ({
            total: 0,
            passed: 0,
            failed: 0,
            pending: 0,
            signed: 0,
        })

        if (!t) {
            return {
                ...blank(),
                byCategory: {} as Record<
                    string,
                    ReturnType<typeof blank>
                >,
            }
        }

        const all = blank()

        const byCategory: Record<
            string,
            ReturnType<typeof blank>
        > = {}

        for (const athlete of t.athletes) {
            const weighIn = athlete.weighIn ?? emptyWeighIn()

            const categoryStats =
                byCategory[athlete.weightCategory] ??= blank()

            all.total++
            categoryStats.total++

            all[weighIn.status]++
            categoryStats[weighIn.status]++

            if (weighIn.signature) {
                all.signed++
                categoryStats.signed++
            }
        }

        return {
            ...all,
            byCategory,
        }
    })

    /**
     * ورزشکارانی که مجاز به حضور در براکت هستند.
     *
     * فقط ورزشکاری که وزنش قبول شده است وارد براکت می‌شود.
     * در نسخه قبلی، ورزشکار pending نیز مجاز شناخته می‌شد.
     */
    const eligibleAthletes = computed(() =>
        (currentTournament.value?.athletes ?? []).filter(
            (athlete) => athlete.weighIn?.status === 'passed',
        ),
    )

    /** آیا این دسته وزنی آماده قرعه‌کشی است؟ */
    function isCategoryWeighInComplete(
        tournamentId: string,
        weightCategory: string,
    ): boolean {
        const tournament = tournaments.value.find(
            (item) => item.id === tournamentId,
        )

        if (!tournament) return false

        const athletes = tournament.athletes.filter(
            (athlete) => athlete.weightCategory === weightCategory,
        )

        // دسته بدون ورزشکار نباید کامل در نظر گرفته شود.
        if (athletes.length === 0) return false

        return athletes.every(
            (athlete) =>
                athlete.weighIn?.status === 'passed' ||
                athlete.weighIn?.status === 'failed',
        )
    }



    return {
        tournaments,
        currentTournamentId,
        currentTournament,
        refreshFromServer,
        createTournament,
        deleteTournament,
        selectTournament,
        addAthlete,
        updateAthlete,
        removeAthlete,
        resetRankings,
        setMatches,
        updateMatch,
        updateMatchResult,
        drawBracket,
        drawBracketForCategory,
        rebalanceCourts,
        reassignCourt,
        swapCourtCategories,
        moveMatch,
        swapAthletes,
        startMatch,
        setMatchResult,
        submitMatchResult,
        clearMatchResult,
        createEmptyResult,
        save,
        exportTournamentToExcel,
        importTournamentFromExcel,
        swapMatchPositions,
        addEliminationMatch,
        removeEliminationMatch,
        setEliminationWinner,

        previewWeighIn,
        recordWeighIn,
        signWeighIn,
        resetWeighIn,
        resetCategoryBracketAndWeighIns,
        clearWeighInSignature,
        isCategoryWeighInComplete,
        weighInStats,
        eligibleAthletes,
        MAX_ATTEMPTS,
    }
})
