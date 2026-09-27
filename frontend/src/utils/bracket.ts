import type { Athlete, Match } from '../types'

function nextPowerOf2(n: number): number {
    let p = 1
    while (p < n) p <<= 1
    return p
}

/** ترتیب استاندارد ریاضی: ۸ → [1,8,5,4,3,6,7,2] */
function standardSeedOrder(size: number): number[] {
    if (size <= 1) return [1]
    const half = standardSeedOrder(size / 2)
    const top = half.map(r => (r % 2 === 1 ? 2 * r - 1 : 2 * r))
    const bottom = [...half].reverse().map(r => (r % 2 === 1 ? 2 * r : 2 * r - 1))
    return [...top, ...bottom]
}

/**
 * جای دستیِ چهار رنک اول.
 * ایندکس اسلات = شماره‌ی سطر بصری؛ نیمه‌ی چپ 0..size/2-1 و نیمه‌ی راست size/2..size-1
 *   رنک ۱ → اولین سطر نیمه‌ی چپ
 *   رنک ۳ → آخرین سطر نیمه‌ی چپ
 *   رنک ۴ → اولین سطر نیمه‌ی راست
 *   رنک ۲ → آخرین سطر نیمه‌ی راست
 */
// function cornerSlots(size: number): { rank1: number; rank2: number; rank3: number; rank4: number } {
//     return {
//         rank1: 0,
//         rank3: size / 2,
//         rank4: size / 2 - 1,
//         rank2: size - 1,
//     }
// }

/**
 * جایگاه سیدهای اصلی برای جدول‌های ۸تایی و بزرگ‌تر:
 * ۱ → چپ بالا
 * ۲ → راست پایین
 * ۳ → راست بالا
 * ۴ → چپ پایین
 *
 * مجموع سیدهای هر جفت دور اول برابر size + 1 است.
 * بنابراین سیدهای بالاتر، اولویت دریافت استراحت دارند.
 */
function seedOrder(size: number): number[] {
    return standardSeedOrder(size)
}

function shuffle<T>(arr: T[]): T[] {
    const a = [...arr]
    for (let i = a.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1))
        ;[a[i], a[j]] = [a[j], a[i]]
    }
    return a
}

/** رنک معتبر = عدد مثبت. '' و 0 و null و undefined یعنی بدون رنک */
function rankOf(a: Athlete): number | null {
    const v: unknown = a.ranking
    if (v === null || v === undefined || v === '') return null
    const n = Number(v)
    return Number.isFinite(n) && n > 0 ? n : null
}

/** کلید تیم: اگر مربی ثبت شده باشد مربی، وگرنه باشگاه */
function teamKeyOf(a: Athlete | null | undefined): string | null {
    if (!a) return null
    const rec = a as Athlete & { coach?: string | null }
    return rec.coach?.trim() || rec.club?.trim() || null
}

/** دوری که دو اسلات به هم می‌رسند (۱ = دور اول، totalRounds = فینال) */
function meetRound(i: number, j: number, totalRounds: number): number {
    for (let r = 1; r <= totalRounds; r++) {
        const p = 1 << r
        if (Math.floor(i / p) === Math.floor(j / p)) return r
    }
    return totalRounds
}

/** profile[r-1] = تعداد جفت هم‌تیمی که در دور r به هم می‌رسند */
function conflictProfile(
    slots: (string | null)[],
    teamOf: Map<string, string>,
    totalRounds: number
): number[] {
    const byTeam = new Map<string, number[]>()
    slots.forEach((id, i) => {
        if (!id) return
        const key = teamOf.get(id)
        if (!key) return
        const list = byTeam.get(key)
        if (list) list.push(i)
        else byTeam.set(key, [i])
    })

    const profile = new Array<number>(totalRounds).fill(0)
    for (const idxs of byTeam.values()) {
        if (idxs.length < 2) continue
        for (let x = 0; x < idxs.length; x++) {
            for (let y = x + 1; y < idxs.length; y++) {
                profile[meetRound(idxs[x], idxs[y], totalRounds) - 1]++
            }
        }
    }
    return profile
}

/** مقایسه‌ی لغوی: برخورد دور اول مهم‌تر از دور دوم، دور دوم مهم‌تر از سوم و … */
function compareProfiles(a: number[], b: number[]): number {
    for (let r = 0; r < a.length; r++) {
        if (a[r] !== b[r]) return a[r] - b[r]
    }
    return 0
}

/** بهترین حالت ممکن: هیچ برخوردی قبل از فینال نباشد */
function isResolved(profile: number[]): boolean {
    for (let r = 0; r < profile.length - 1; r++) {
        if (profile[r] > 0) return false
    }
    return true
}

/**
 * جابه‌جایی محلی برای دیرترین برخورد ممکن بین هم‌تیمی‌ها.
 * - اسلات‌های قفل‌شده (رنک ۱ تا ۴ در قرعه‌ی رنکینگی) دست نمی‌خورند.
 * - جابه‌جایی فقط درون یک کلاس انجام می‌شود: «حریفش استراحت است» یا «حریف واقعی دارد».
 *   بنابراین مجموعه‌ی کسانی که استراحت می‌گیرند هرگز تغییر نمی‌کند.
 */
function optimizeTeamSeparation(
    slots: (string | null)[],
    athleteMap: Map<string, Athlete>,
    fixed: Set<number>,
    totalRounds: number
): void {
    const teamOf = new Map<string, string>()
    const teamSize = new Map<string, number>()
    for (const [id, athlete] of athleteMap) {
        const key = teamKeyOf(athlete)
        if (!key) continue
        teamOf.set(id, key)
        teamSize.set(key, (teamSize.get(key) ?? 0) + 1)
    }

    let hasConflict = false
    for (const count of teamSize.values()) {
        if (count > 1) { hasConflict = true; break }
    }
    if (!hasConflict) return

    // حریف هر اسلات در دور اول: i ^ 1  (جفت‌ها ۰-۱، ۲-۳، ...)
    const hasBye = (i: number): boolean => slots[i ^ 1] === null

    const movable = slots
        .map((_, i) => i)
        .filter(i => slots[i] !== null && !fixed.has(i))
    const byeSlots = movable.filter(hasBye)
    const playSlots = movable.filter(i => !hasBye(i))
    const groups = [byeSlots, playSlots].filter(g => g.length > 1)
    if (groups.length === 0) return

    const isCrowded = (i: number): boolean => {
        const id = slots[i]
        if (!id) return false
        const key = teamOf.get(id)
        return key ? (teamSize.get(key) ?? 0) > 1 : false
    }

    const hillClimb = (): number[] => {
        let profile = conflictProfile(slots, teamOf, totalRounds)
        for (let pass = 0; pass < 30 && !isResolved(profile); pass++) {
            let improved = false
            for (const group of groups) {
                for (const i of group) {
                    if (!isCrowded(i)) continue
                    for (const j of group) {
                        if (i === j) continue
                            ;[slots[i], slots[j]] = [slots[j], slots[i]]
                        const candidate = conflictProfile(slots, teamOf, totalRounds)
                        if (compareProfiles(candidate, profile) < 0) {
                            profile = candidate
                            improved = true
                        } else {
                            ;[slots[i], slots[j]] = [slots[j], slots[i]]
                        }
                    }
                }
            }
            if (!improved) break
        }
        return profile
    }

    const snapshot = (): (string | null)[] => movable.map(i => slots[i])
    const restore = (values: (string | null)[]): void => {
        movable.forEach((slot, k) => { slots[slot] = values[k] })
    }

    let bestProfile = hillClimb()
    let bestValues = snapshot()

    // چند شروع تصادفی برای فرار از کمینه‌ی محلی؛ بُر زدن جداگانه در هر کلاس
    for (let attempt = 0; attempt < 8 && !isResolved(bestProfile); attempt++) {
        restore(bestValues)
        for (const group of groups) {
            const shuffled = shuffle(group.map(i => slots[i]))
            group.forEach((slot, k) => { slots[slot] = shuffled[k] })
        }

        const profile = hillClimb()
        if (compareProfiles(profile, bestProfile) < 0) {
            bestProfile = profile
            bestValues = snapshot()
        }
    }

    restore(bestValues)
}

export function generateBracket(
    athletes: Athlete[],
    weightCategory: string,
    startCourt: number,
    drawType: 'random' | 'ranked' = 'random'
): Match[] {
    if (athletes.length < 2) return []

    const athleteMap = new Map(athletes.map(a => [a.id, a]))
    const size = nextPowerOf2(athletes.length)
    const order = seedOrder(size)
    const totalRounds = Math.log2(size)

    // entrants[k] = ورزشکار رنک k+1 ؛ رنک‌های بعد از تعداد ورزشکاران خالی = استراحت
    let entrants: Athlete[]
    const fixed = new Set<number>()

    if (drawType === 'ranked') {
        // همه‌ی رنک‌دارها به‌ترتیب رنک روی سیدهای ۱ به بالا؛ تساوی رنک با id شکسته می‌شود
        const ranked = athletes
            .filter(a => rankOf(a) !== null)
            .sort((a, b) => rankOf(a)! - rankOf(b)! || String(a.id).localeCompare(String(b.id)))
        const rankedIds = new Set(ranked.map(a => a.id))
        const rest = shuffle(athletes.filter(a => !rankedIds.has(a.id)))
        entrants = [...ranked, ...rest]

        // چهار گوشه قفل می‌شود: ۱ بالا-چپ، ۳ پایین-چپ، ۴ بالا-راست، ۲ پایین-راست
        const lockCount = Math.min(4, ranked.length)
        for (let k = 1; k <= lockCount; k++) fixed.add(order.indexOf(k))
    } else {
        entrants = shuffle(athletes)
    }

    const slots: (string | null)[] = order.map(seed => {
        const a = entrants[seed - 1]
        return a ? a.id : null
    })

    optimizeTeamSeparation(slots, athleteMap, fixed, totalRounds)

    const matches: Match[] = []
    let matchOrder = 1

    type SlotInfo = { id: string; side: 'left' | 'right' | 'final' }
    const roundMatchIds: SlotInfo[][] = []

    // دور اول — به ترتیب سطرهای بصری از بالا به پایین
    const firstRound: SlotInfo[] = []
    for (let i = 0; i < slots.length; i += 2) {
        const id = crypto.randomUUID()
        const side: 'left' | 'right' | 'final' = totalRounds === 1 ? 'final' : i < slots.length / 2 ? 'left' : 'right'
        const athlete1Id = slots[i]
        const athlete2Id = slots[i + 1]
        const isBye = (athlete1Id === null) !== (athlete2Id === null)

        matches.push({
            id,
            athlete1Id,
            athlete2Id,
            court: isBye ? 0 : startCourt,
            order: isBye ? 0 : matchOrder++,
            bracketIndex: i / 2,
            weightCategory,
            round: 1,
            side,
            isBye,
        })
        firstRound.push({ id, side })
    }
    roundMatchIds.push(firstRound)

    // دورهای بعدی
    let prev = firstRound
    for (let round = 2; round <= totalRounds; round++) {
        const cur: SlotInfo[] = []
        for (let i = 0; i < prev.length; i += 2) {
            const id = crypto.randomUUID()
            const isLast = round === totalRounds
            const side = isLast ? ('final' as const) : prev[i].side
            matches.push({
                id,
                athlete1Id: null,
                athlete2Id: null,
                court: startCourt,
                order: matchOrder++,
                bracketIndex: i / 2,
                weightCategory,
                round,
                side,
                isBye: false,
            })
            cur.push({ id, side })
        }
        roundMatchIds.push(cur)
        prev = cur
    }

    // اتصال nextMatchId و nextSlot (اسلات بالا/پایین در مرحله‌ی بعد)
    for (let r = 0; r < roundMatchIds.length - 1; r++) {
        const cur = roundMatchIds[r]
        const next = roundMatchIds[r + 1]
        cur.forEach((m, i) => {
            const match = matches.find(x => x.id === m.id)!
            match.nextMatchId = next[Math.floor(i / 2)].id
            match.nextSlot = i % 2 === 0 ? 'athlete1' : 'athlete2'
        })
    }

    return autoAdvanceByes(matches)
}

/** حذف زنجیره‌ای یک ورزشکار از مراحل بعد (برای undo یا تغییر برنده) */
function clearDownstream(byId: Map<string, Match>, match: Match, athleteId: string): void {
    if (!match.nextMatchId) return
    const next = byId.get(match.nextMatchId)
    if (!next) return

    if ((match.nextSlot ?? 'athlete1') === 'athlete1') {
        if (next.athlete1Id !== athleteId) return
        next.athlete1Id = null
    } else {
        if (next.athlete2Id !== athleteId) return
        next.athlete2Id = null
    }

    if (next.winnerId === athleteId) {
        next.winnerId = undefined
        clearDownstream(byId, next, athleteId)
    }
}

export function recordResult(matches: Match[], matchId: string, winnerId: string): Match[] {
    const clones = matches.map(m => ({ ...m }))
    const byId = new Map(clones.map(m => [m.id, m]))
    const match = byId.get(matchId)
    if (!match || match.isBye || (winnerId !== match.athlete1Id && winnerId !== match.athlete2Id)) return matches
    const visited = new Set<string>([match.id])
    let nextId = match.nextMatchId
    while (nextId) {
        if (visited.has(nextId)) return matches
        visited.add(nextId)
        const next = byId.get(nextId)
        if (!next || next.winnerId || next.result || next.status === 'ongoing') return matches
        nextId = next.nextMatchId
    }

    const isUndo = match.winnerId === winnerId

    if (isUndo) {
        match.winnerId = undefined
        match.result = undefined
        match.status = 'pending'
        clearDownstream(byId, match, winnerId)
    } else {
        const previous = match.winnerId
        match.winnerId = winnerId
        match.result = undefined
        match.status = 'completed'
        if (previous && previous !== winnerId) clearDownstream(byId, match, previous)

        const next = match.nextMatchId ? byId.get(match.nextMatchId) : undefined
        if (next) {
            if ((match.nextSlot ?? 'athlete1') === 'athlete1') next.athlete1Id = winnerId
            else next.athlete2Id = winnerId
        }
    }

    return autoAdvanceByes(clones)
}

export function autoAdvanceByes(matches: Match[]): Match[] {
    const result = matches.map(m => ({ ...m }))
    const byId = new Map(result.map(m => [m.id, m]))

    let changed = true
    while (changed) {
        changed = false

        for (const m of result) {
            if (m.winnerId) continue

            const lone =
                m.athlete1Id && !m.athlete2Id ? m.athlete1Id :
                    !m.athlete1Id && m.athlete2Id ? m.athlete2Id : null
            if (!lone) continue

            // تا وقتی یکی از خوراک‌دهنده‌ها برنده ندارد، صعود نده
            const pending = result.some(f => f.nextMatchId === m.id && !f.winnerId)
            if (pending) continue

            m.winnerId = lone
            m.isBye = true
            m.status = 'completed'
            m.court = 0
            m.order = 0
            changed = true

            const next = m.nextMatchId ? byId.get(m.nextMatchId) : undefined
            if (!next || next.winnerId) continue

            if ((m.nextSlot ?? 'athlete1') === 'athlete1') next.athlete1Id = lone
            else next.athlete2Id = lone
        }
    }

    return result
}

/** Protect completed/ongoing downstream games when correcting an earlier result. */
export function hasLockedDownstream(matches: Match[], from: Match): boolean {
    const byId = new Map(matches.map(m => [m.id, m]))
    const visited = new Set([from.id])
    let nextId = from.nextMatchId
    while (nextId) {
        if (visited.has(nextId)) return true
        visited.add(nextId)
        const next = byId.get(nextId)
        if (!next || (!next.isBye && (next.winnerId || next.result || next.status === 'ongoing' || next.status === 'completed'))) return true
        nextId = next.nextMatchId
    }
    return false
}
