// src/utils/teamDraw.ts
import type {
    Team, TeamWeightSlot, TeamDrawOptions, TeamEncounter, EncounterBout,
} from '../types'

const uid = () => crypto.randomUUID()
const BYE = '__BYE__'

interface Pairing { roundNo: number; home: string; away: string }

function shuffle<T>(input: T[]): T[] {
    const a = [...input]
    for (let i = a.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1))
        ;[a[i], a[j]] = [a[j], a[i]]
    }
    return a
}

/** جفت‌های دور-رابین با روش دایره‌ای؛ تعداد فرد → یار خالی. میزبانی دور به دور جابه‌جا می‌شود */
function circlePairs(teamIds: string[]): Pairing[] {
    const arr = [...teamIds]
    if (arr.length % 2 === 1) arr.push(BYE)
    const n = arr.length
    const out: Pairing[] = []

    for (let r = 0; r < n - 1; r++) {
        for (let i = 0; i < n / 2; i++) {
            const a = arr[i], b = arr[n - 1 - i]
            if (a === BYE || b === BYE) continue
            out.push(r % 2 === 0
                ? { roundNo: r + 1, home: a, away: b }
                : { roundNo: r + 1, home: b, away: a })
        }
        arr.splice(1, 0, arr.pop()!)   // عنصر اول ثابت، بقیه می‌چرخند
    }
    return out
}

/* ═════════ ابزار زمان‌بندی ═════════ */

const push = <K, V>(map: Map<K, V[]>, key: K, value: V) => {
    const list = map.get(key)
    if (list) list.push(value); else map.set(key, [value])
}

const minOf = (a: number[], fallback = 0) => a.reduce((m, v) => (v < m ? v : m), a.length ? a[0] : fallback)
const maxOf = (a: number[], fallback = 0) => a.reduce((m, v) => (v > m ? v : m), a.length ? a[0] : fallback)

function restOk(
    teamSlots: Map<string, number[]>, teams: string[], slot: number, minRest: number,
): boolean {
    if (minRest <= 0) return true
    for (const t of teams) {
        for (const s of teamSlots.get(t) ?? []) {
            if (Math.abs(s - slot) < minRest) return false
        }
    }
    return true
}

/** ظرفیت آزاد بودن نوبت + نبودن تیم تکراری در همان نوبت */
function slotFree(
    load: Map<number, number>, teamsInSlot: Map<number, Set<string>>,
    slot: number, teams: string[], capacity: number,
): boolean {
    if ((load.get(slot) ?? 0) >= capacity) return false
    const set = teamsInSlot.get(slot)
    return !set || !teams.some((t) => set.has(t))
}

function occupy(
    load: Map<number, number>, teamsInSlot: Map<number, Set<string>>,
    teamSlots: Map<string, number[]>, slot: number, teams: string[],
) {
    load.set(slot, (load.get(slot) ?? 0) + 1)
    let set = teamsInSlot.get(slot)
    if (!set) { set = new Set(); teamsInSlot.set(slot, set) }
    for (const t of teams) { set.add(t); push(teamSlots, t, slot) }
}

/** هزینه: تداخل و سرریز ظرفیت سخت، کمبود استراحت نرم، طول برنامه جریمهٔ ناچیز */
function scoreSlotting(
    pairs: Pairing[], slotting: number[], capacity: number, minRest: number,
): number {
    if (!slotting.length) return 0

    let c = 0
    const load = new Map<number, number>()
    const teamsInSlot = new Map<number, Set<string>>()
    const teamSlots = new Map<string, number[]>()

    for (let i = 0; i < pairs.length; i++) {
        const s = slotting[i]
        load.set(s, (load.get(s) ?? 0) + 1)
        let set = teamsInSlot.get(s)
        if (!set) { set = new Set(); teamsInSlot.set(s, set) }
        for (const t of [pairs[i].home, pairs[i].away]) {
            if (set.has(t)) c += 1e6
            set.add(t)
            push(teamSlots, t, s)
        }
    }
    for (const count of load.values()) if (count > capacity) c += (count - capacity) * 1e6

    for (const slots of teamSlots.values()) {
        slots.sort((a, b) => a - b)
        for (let i = 1; i < slots.length; i++) {
            const gap = slots[i] - slots[i - 1]
            if (gap < minRest) { const d = minRest - gap; c += d * d * 100 }
        }
    }
    return c + (maxOf(slotting) - minOf(slotting) + 1)
}

/** چیدمان اولیه: به ترتیب دور، اولین نوبت بدون تداخل و با استراحت کافی */
function greedySlotting(
    pairs: Pairing[], capacity: number, minRest: number, slotCount: number,
): { slotting: number[]; slotCount: number } {
    const slotting = new Array<number>(pairs.length).fill(0)
    const load = new Map<number, number>()
    const teamsInSlot = new Map<number, Set<string>>()
    const teamSlots = new Map<string, number[]>()
    const order = pairs.map((_, i) => i).sort((a, b) => pairs[a].roundNo - pairs[b].roundNo)
    let span = slotCount

    for (const i of order) {
        const teams = [pairs[i].home, pairs[i].away]
        let chosen = -1

        // پاس اول با رعایت استراحت، پاس دوم فقط بدون تداخل
        for (let pass = 0; pass < 2 && chosen < 0; pass++) {
            for (let s = 0; s < span; s++) {
                if (!slotFree(load, teamsInSlot, s, teams, capacity)) continue
                if (pass === 0 && !restOk(teamSlots, teams, s, minRest)) continue
                chosen = s; break
            }
        }
        if (chosen < 0) chosen = span++   // ظرفیت تمام شد، نوبت تازه

        slotting[i] = chosen
        occupy(load, teamsInSlot, teamSlots, chosen, teams)
    }

    return { slotting, slotCount: span }
}

/** تپه‌نوردی سبک با دو عملگر: تعویض دو مواجهه و انتقال یک مواجهه به نوبت دیگر */
function optimizeSlotting(
    pairs: Pairing[], capacity: number, minRest: number, slotCount: number, iterations: number,
): number[] {
    if (pairs.length < 2) return new Array<number>(pairs.length).fill(0)

    const seed = greedySlotting(pairs, capacity, minRest, slotCount)
    let best = seed.slotting
    let bestCost = scoreSlotting(pairs, best, capacity, minRest)
    const span = Math.max(seed.slotCount, maxOf(best) + 1)

    for (let it = 0; it < iterations && bestCost > 0; it++) {
        const cand = best.slice()
        if (Math.random() < 0.5) {
            const i = Math.floor(Math.random() * cand.length)
            const j = Math.floor(Math.random() * cand.length)
            ;[cand[i], cand[j]] = [cand[j], cand[i]]
        } else {
            const i = Math.floor(Math.random() * cand.length)
            cand[i] = Math.floor(Math.random() * span)
        }
        const c = scoreSlotting(pairs, cand, capacity, minRest)
        if (c <= bestCost) { best = cand; bestCost = c }
    }
    return best
}

/**
 * پاس ترمیم: تضمین می‌کند هیچ تیمی در یک نوبت دو مواجهه ندارد و ظرفیت زمین‌ها
 * سرریز نمی‌شود. مواجهه‌ها به نزدیک‌ترین نوبت مجاز به محل بهینه‌شده منتقل می‌شوند.
 */
function repairSlotting(
    pairs: Pairing[], slotting: number[], capacity: number, minRest: number,
): number[] {
    if (!pairs.length) return []

    const out = new Array<number>(pairs.length).fill(0)
    const load = new Map<number, number>()
    const teamsInSlot = new Map<number, Set<string>>()
    const teamSlots = new Map<string, number[]>()

    const limit = maxOf(slotting) + pairs.length + 1
    const all = Array.from({ length: limit + 1 }, (_, s) => s)
    const order = pairs
        .map((_, i) => i)
        .sort((a, b) => slotting[a] - slotting[b] || pairs[a].roundNo - pairs[b].roundNo)

    for (const i of order) {
        const teams = [pairs[i].home, pairs[i].away]
        const desired = slotting[i]
        const ranked = [...all].sort(
            (a, b) => Math.abs(a - desired) - Math.abs(b - desired) || a - b,
        )

        let chosen = -1
        for (let pass = 0; pass < 2 && chosen < 0; pass++) {
            for (const s of ranked) {
                if (!slotFree(load, teamsInSlot, s, teams, capacity)) continue
                if (pass === 0 && !restOk(teamSlots, teams, s, minRest)) continue
                chosen = s; break
            }
        }
        if (chosen < 0) chosen = limit + 1 + i   // نباید رخ دهد، ولی خروجی سالم می‌ماند

        out[i] = chosen
        occupy(load, teamsInSlot, teamSlots, chosen, teams)
    }

    return out
}

/* ═════════ خروجی ═════════ */

function newBout(slotNo: number, court: number, homeCorner: 'blue' | 'red'): EncounterBout {
    return {
        id: uid(), slotNo,
        homeAthleteId: null, awayAthleteId: null,
        court, status: 'pending', winnerSide: null,
        homeRoundsWon: 0, awayRoundsWon: 0, homeCorner,
    }
}

export function generateTeamDraw(
    teams: Team[],
    weightSlots: TeamWeightSlot[],
    options: TeamDrawOptions,
    courts: number,
): TeamEncounter[] {
    if (teams.length < 2) return []

    const matsPerEncounter = Math.max(1, options.matsPerEncounter)
    const matsCount = Math.max(1, courts)
    const minRest = Math.max(0, options.minRest)
    const slots = [...weightSlots].sort((a, b) => a.slotNo - b.slotNo)

    const ordered = options.drawType === 'seeded'
        ? [...teams].sort((a, b) =>
            (a.seed ?? Number.MAX_SAFE_INTEGER) - (b.seed ?? Number.MAX_SAFE_INTEGER))
        : shuffle(teams)

    const pairs = circlePairs(ordered.map((t) => t.id))
    if (!pairs.length) return []

    // ظرفیت هم‌زمانی: هم محدود به زمین‌ها، هم به حداکثر floor(N/2) مواجههٔ موازی
    const capacity = options.parallel
        ? Math.max(1, Math.min(
            Math.floor(teams.length / 2),
            Math.floor(matsCount / matsPerEncounter),
        ))
        : 1

    // کف تعداد نوبت‌ها: هم ظرفیت، هم امکان‌پذیر بودن استراحت برای تیمی که همه را بازی می‌کند
    const matchesPerTeam = teams.length - 1
    const slotCount = Math.max(
        Math.ceil(pairs.length / capacity),
        (matchesPerTeam - 1) * Math.max(1, minRest) + 1,
    )

    const optimized = optimizeSlotting(
        pairs, capacity, minRest, slotCount, Math.max(0, options.iterations ?? 3000),
    )
    const slotting = repairSlotting(pairs, optimized, capacity, minRest)

    // نوبت‌ها ۱-پایه می‌شوند؛ نوبت‌های خالی حفظ می‌شوند چون زمان انتظار واقعی‌اند
    const offset = 1 - minOf(slotting)
    const perSlot = new Map<number, number>()

    return pairs
        .map((p, i) => ({ p, slotIndex: slotting[i] + offset }))
        .sort((a, b) => a.slotIndex - b.slotIndex || a.p.roundNo - b.p.roundNo)
        .map(({ p, slotIndex }) => {
            const k = perSlot.get(slotIndex) ?? 0
            perSlot.set(slotIndex, k + 1)
            // زمین شروع مواجهه؛ ترمیم تضمین کرده k < capacity، پس court+mats-1 ≤ courts
            const court = k * matsPerEncounter + 1

            const encounter: TeamEncounter = {
                id: uid(),
                roundNo: p.roundNo,
                slotIndex,
                court,
                homeTeamId: p.home,
                awayTeamId: p.away,
                status: 'scheduled',
                lineups: [],
                bouts: slots.map((slot, idx) => newBout(
                    slot.slotNo,
                    court + (idx % matsPerEncounter),
                    idx % 2 === 0 ? 'blue' : 'red',   // گوشهٔ میزبان اسلات به اسلات جابه‌جا می‌شود
                )),
                homeBouts: 0, awayBouts: 0,
                homeRounds: 0, awayRounds: 0,
                homePoints: 0, awayPoints: 0,
            }
            return encounter
        })
}
