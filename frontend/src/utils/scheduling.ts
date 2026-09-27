import type { Match } from '../types'

export interface CategoryStat {
    cat: string
    load: number              // بازی‌های قابل اجرا (بدون bye)
    firstRoundPlayable: number
    pinned: boolean           // نتیجه ثبت شده یا دستی جابه‌جا شده
    pinnedCourts: number[]
}

export interface PlanOptions {
    courts: number
    maxCourtsPerCategory?: number  // پیش‌فرض ۲
    minMatchesToSplit?: number     // پیش‌فرض ۶
}

export interface CourtPlan {
    assignment: Record<string, number[]>
    locked: string[]
}

export function planCourts(stats: CategoryStat[], opts: PlanOptions): CourtPlan {
    const totalCourts = Math.max(1, Math.floor(opts.courts || 1))
    const hardCap = Math.max(1, Math.min(totalCourts, opts.maxCourtsPerCategory ?? 2))
    const minSplit = opts.minMatchesToSplit ?? 6

    const assignment: Record<string, number[]> = {}
    const courtLoad = new Map<number, number>()
    for (let c = 1; c <= totalCourts; c++) courtLoad.set(c, 0)

    for (const s of stats) if (s.load === 0) assignment[s.cat] = []
    const active = stats.filter(s => s.load > 0)
    const locked: string[] = []
    if (!active.length) return { assignment, locked }

    // بیشترین زمینی که برای یک وزن واقعاً فایده دارد
    const maxUseful = (s: CategoryStat) =>
        s.load < minSplit ? 1 : Math.max(1, Math.min(hardCap, Math.floor(s.firstRoundPlayable / 2)))

    const addLoad = (courts: number[], load: number) => {
        const share = load / courts.length
        for (const c of courts) courtLoad.set(c, (courtLoad.get(c) ?? 0) + share)
    }

    // ۱) وزن‌های قفل‌شده: زمینشان دست‌نخورده می‌ماند، ولی بارشان حساب می‌شود
    for (const s of active) {
        if (!s.pinned) continue
        const courts = s.pinnedCourts.filter(c => c >= 1 && c <= totalCourts)
        if (!courts.length) { s.pinned = false; continue }
        assignment[s.cat] = courts
        locked.push(s.cat)
        addLoad(courts, s.load)
    }

    const free = active.filter(s => !s.pinned).sort((a, b) => b.load - a.load)
    if (!free.length) return { assignment, locked }

    const totalLoad = Math.max(1, free.reduce((a, s) => a + s.load, 0))

    // ۲) سهم متناسب هر وزن، سپس کم‌بارترین زمین‌ها
    for (const s of free) {
        const ideal = Math.round((totalCourts * s.load) / totalLoad)
        const want = Math.max(1, Math.min(ideal, maxUseful(s)))

        const chosen = Array.from({ length: totalCourts }, (_, i) => i + 1)
            .sort((a, b) => (courtLoad.get(a)! - courtLoad.get(b)!) || a - b)
            .slice(0, want)
            .sort((a, b) => a - b)

        assignment[s.cat] = chosen
        addLoad(chosen, s.load)
    }

    // ۳) زمین‌های خالی‌مانده را به وزن‌هایی بده که هنوز ظرفیت گسترش دارند
    for (;;) {
        const empty = Array.from({ length: totalCourts }, (_, i) => i + 1)
            .filter(c => (courtLoad.get(c) ?? 0) === 0)
        if (!empty.length) break

        const candidate = free
            .filter(s => assignment[s.cat].length < maxUseful(s))
            .sort((a, b) =>
                (b.load / assignment[b.cat].length) - (a.load / assignment[a.cat].length))[0]
        if (!candidate) break

        const c = empty[0]
        const courts = [...assignment[candidate.cat], c].sort((a, b) => a - b)
        // بار قبلی این وزن را برگردان و با تقسیم جدید دوباره حساب کن
        addLoad(assignment[candidate.cat], -candidate.load)
        assignment[candidate.cat] = courts
        addLoad(courts, candidate.load)
    }

    return { assignment, locked }
}

/**
 * توزیع بازی‌های یک وزن روی زمین‌های تخصیص‌یافته.
 * جدول به k قطعه پیوسته تقسیم می‌شود (نیمه بالا / نیمه پایین)،
 * فینال همیشه روی زمین اصلی وزن.
 */
export function assignCategoryCourts(categoryMatches: Match[], courts: number[]) {
    for (const m of categoryMatches) if (m.isBye) { m.court = 0; m.order = 0 }

    const playable = categoryMatches.filter(m => !m.isBye)
    if (!playable.length) return

    const pool = (courts.length ? courts : [1]).slice().sort((a, b) => a - b)
    if (pool.length === 1) {
        for (const m of playable) m.court = pool[0]
        return
    }

    const maxRound = Math.max(...categoryMatches.map(m => m.round))
    const k = pool.length

    const rounds = new Map<number, Match[]>()
    for (const m of categoryMatches) {
        const arr = rounds.get(m.round) ?? []
        arr.push(m)
        rounds.set(m.round, arr)
    }

    for (const [round, arr] of rounds) {
        const len = arr.length
        arr.forEach((m, i) => {
            if (m.isBye) return
            if (round === maxRound) { m.court = pool[0]; return }
            m.court = pool[Math.min(k - 1, Math.floor((i * k) / len))]
        })
    }
}

/** همگام‌سازی فازها: دورهای پایانی وزن‌های مختلف نزدیک هم تمام شوند. */
export function syncOrders(matches: Match[]) {
    const active = matches.filter(m => m.court >= 1 && !m.isBye)
    if (!active.length) return

    const maxRound: Record<string, number> = {}
    for (const m of active) {
        maxRound[m.weightCategory] = Math.max(maxRound[m.weightCategory] ?? 0, m.round)
    }
    const phaseOf = (m: Match) => (maxRound[m.weightCategory] ?? 1) - m.round
    const maxPhase = Math.max(...active.map(phaseOf))

    const queues = new Map<number, Match[]>()

    for (let phase = maxPhase; phase >= 0; phase--) {
        const inPhase = active.filter(m => phaseOf(m) === phase)
        if (!inPhase.length) continue

        const catSize: Record<string, number> = {}
        for (const m of inPhase) catSize[m.weightCategory] = (catSize[m.weightCategory] ?? 0) + 1

        const ordered = [...inPhase].sort((a, b) => {
            if (a.court !== b.court) return a.court - b.court
            const d = (catSize[b.weightCategory] ?? 0) - (catSize[a.weightCategory] ?? 0)
            if (d !== 0) return d
            if (a.weightCategory !== b.weightCategory) {
                return a.weightCategory.localeCompare(b.weightCategory, 'fa')
            }
            return a.side === b.side ? 0 : a.side < b.side ? -1 : 1
        })

        for (const m of ordered) {
            const q = queues.get(m.court) ?? []
            q.push(m)
            queues.set(m.court, q)
        }
    }

    for (const [, q] of queues) {
        const done = q.filter(m => m.winnerId).sort((a, b) => (a.order || 0) - (b.order || 0))
        const rest = q.filter(m => !m.winnerId)
        ;[...done, ...rest].forEach((m, i) => { m.order = i + 1 })
    }
}
