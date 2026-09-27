import type { Athlete, WeighIn, WeighInAttempt } from '../types'

export const TOLERANCE_KG = 0.2
export const MAX_ATTEMPTS = 2

export interface WeightCheckResult {
    ok: boolean
    deltaKg: number
    withTolerance: boolean
    message: string
    minKg: number | null
    maxKg: number | null
}

/** '-58' → سقف ۵۸ | '+87' → کف ۸۷ */
export function parseCategory(
    category: string,
): { limitKg: number; isOpen: boolean } | null {
    const normalized = String(category).trim().replace(/[^\d.+\-]/g, '')
    const limitKg = Number.parseFloat(normalized.replace(/[+\-]/g, ''))

    if (!Number.isFinite(limitKg)) return null

    return {
        limitKg,
        isOpen: normalized.startsWith('+'),
    }
}

/**
 * بازه یک دسته وزنی را بر اساس فهرست اوزان محاسبه می‌کند.
 *
 * مثال:
 * categories = ['-42', '-44', '-46', ..., '+68']
 *
 * -42  → بدون حد پایین، حداکثر 42
 * -44  → بیشتر از 42، حداکثر 44
 * -46  → بیشتر از 44، حداکثر 46
 * +68  → بیشتر از 68، بدون حد بالا
 */
export function categoryRange(
    category: string,
    categories: readonly string[],
): { minKg: number | null; maxKg: number | null } | null {
    const categoryIndex = categories.indexOf(category)

    if (categoryIndex === -1) return null

    const currentCategory = parseCategory(category)

    if (!currentCategory) return null

    if (currentCategory.isOpen) {
        return {
            minKg: currentCategory.limitKg,
            maxKg: null,
        }
    }

    const previousCategory =
        categoryIndex > 0
            ? parseCategory(categories[categoryIndex - 1])
            : null

    return {
        minKg: previousCategory?.limitKg ?? null,
        maxKg: currentCategory.limitKg,
    }
}

/**
 * بررسی می‌کند وزن واردشده در محدوده کامل دسته وزنی قرار دارد یا نه.
 *
 * قواعد:
 * - حد پایین دسته، سقف دسته قبلی است و خود آن عدد متعلق به دسته قبلی است.
 * - ۲۰۰ گرم ارفاق فقط روی سقف دسته اعمال می‌شود.
 * - دسته مثبت حد بالا ندارد.
 */
export function checkWeight(
    weightKg: number,
    category: string,
    categories: readonly string[],
): WeightCheckResult {
    const invalidResult: WeightCheckResult = {
        ok: false,
        deltaKg: 0,
        withTolerance: false,
        message: 'وزن یا دستهٔ نامعتبر',
        minKg: null,
        maxKg: null,
    }

    if (!Number.isFinite(weightKg) || weightKg <= 0) {
        return invalidResult
    }

    const range = categoryRange(category, categories)

    if (!range) {
        return invalidResult
    }

    /*
     * محاسبات با گرم انجام می‌شوند تا خطای اعشار ممیز شناور
     * روی مرزهایی مثل 44.2 ایجاد نشود.
     */
    const weightGrams = Math.round(weightKg * 1000)
    const toleranceGrams = Math.round(TOLERANCE_KG * 1000)

    const minGrams =
        range.minKg === null
            ? null
            : Math.round(range.minKg * 1000)

    const maxGrams =
        range.maxKg === null
            ? null
            : Math.round(range.maxKg * 1000)

    // حد پایین متعلق به دسته قبلی است؛ بنابراین وزن باید بیشتر باشد.
    const belowMinimum =
        minGrams !== null && weightGrams <= minGrams

    // ارفاق فقط به سقف دسته اضافه می‌شود.
    const aboveMaximum =
        maxGrams !== null &&
        weightGrams > maxGrams + toleranceGrams

    if (belowMinimum && minGrams !== null) {
        const shortageGrams = minGrams - weightGrams

        return {
            ok: false,
            deltaKg: -(shortageGrams / 1000),
            withTolerance: false,
            message:
                `کم‌وزن — وزن باید بیشتر از ` +
                `${formatNumber(range.minKg!)} کیلوگرم باشد`,
            minKg: range.minKg,
            maxKg: range.maxKg,
        }
    }

    if (aboveMaximum && maxGrams !== null) {
        const excessGrams = weightGrams - maxGrams

        return {
            ok: false,
            deltaKg: excessGrams / 1000,
            withTolerance: false,
            message:
                `اضافه‌وزن — ${formatNumber(excessGrams / 1000)} ` +
                `کیلوگرم بیشتر از سقف ${formatNumber(range.maxKg!)} کیلوگرم`,
            minKg: range.minKg,
            maxKg: range.maxKg,
        }
    }

    const toleranceUsed =
        maxGrams !== null &&
        weightGrams > maxGrams &&
        weightGrams <= maxGrams + toleranceGrams

    const deltaGrams =
        toleranceUsed && maxGrams !== null
            ? weightGrams - maxGrams
            : 0

    return {
        ok: true,
        deltaKg: deltaGrams / 1000,
        withTolerance: toleranceUsed,
        message: toleranceUsed
            ? `سر وزن با ارفاق (${formatNumber(deltaGrams / 1000)} کیلو اختلاف)`
            : `سر وزن — ${formatNumber(weightKg)} کیلوگرم`,
        minKg: range.minKg,
        maxKg: range.maxKg,
    }
}

/**
 * یک نوبت وزن‌کشی را روی رکورد ثبت می‌کند و وضعیت را به‌روز می‌کند.
 *
 * categories باید فهرست اوزان مربوط به سن و جنسیت همان ورزشکار باشد.
 */
export function recordAttempt(
    weighIn: WeighIn,
    weightKg: number,
    category: string,
    categories: readonly string[],
): WeighIn {
    if (!Number.isFinite(weightKg) || weightKg <= 0 || !categoryRange(category, categories)) return weighIn
    if (weighIn.status !== 'pending') return weighIn
    if (weighIn.attempts.length >= MAX_ATTEMPTS) return weighIn

    const check = checkWeight(weightKg, category, categories)

    const attempt: WeighInAttempt = {
        no: weighIn.attempts.length + 1,
        weightKg,
        ok: check.ok,
        at: new Date().toISOString(),
    }

    const attempts = [...weighIn.attempts, attempt]

    return {
        ...weighIn,
        attempts,
        weightKg,
        withTolerance: check.withTolerance,
        status: check.ok
            ? 'passed'
            : attempts.length >= MAX_ATTEMPTS
                ? 'failed'
                : 'pending',
    }
}

export const attemptsLeft = (weighIn: WeighIn): number =>
    Math.max(0, MAX_ATTEMPTS - weighIn.attempts.length)

export const canWeigh = (weighIn: WeighIn): boolean =>
    weighIn.status === 'pending' && attemptsLeft(weighIn) > 0

/** امضا فقط بعد از قبولی و فقط یک بار امکان‌پذیر است. */
export const canSign = (weighIn: WeighIn): boolean =>
    weighIn.status === 'passed' && !weighIn.signature

export function sign(
    weighIn: WeighIn,
    imagePath: string,
): WeighIn {
    if (!canSign(weighIn) || !imagePath.trim()) return weighIn

    return {
        ...weighIn,
        signature: {
            imagePath,
            signedAt: new Date().toISOString(),
        },
    }
}

export function emptyWeighIn(): WeighIn {
    return {
        status: 'pending',
        weightKg: null,
        withTolerance: false,
        attempts: [],
        signature: null,
    }
}

/* نمایش */

export const STATUS_LABELS: Record<WeighIn['status'], string> = {
    pending: 'در انتظار',
    passed: 'سر وزن',
    failed: 'مردود',
}

/** قبولی با ارفاق زرد می‌ماند تا داور بداند ارفاق اعمال شده است. */
export function statusTone(
    weighIn: WeighIn,
): 'neutral' | 'success' | 'warning' | 'danger' {
    if (weighIn.status === 'failed') return 'danger'
    if (weighIn.status === 'pending') return 'neutral'

    return weighIn.withTolerance
        ? 'warning'
        : 'success'
}

function formatNumber(value: number): string {
    return Number(value.toFixed(3)).toString()
}

/** Preserve old approvals without fabricating a scale reading or a signature. */
export function migrateAthleteWeighIn(a: Athlete): Athlete {
    const weighIn = a.weighIn ?? {
        ...emptyWeighIn(),
        status: a.weighedIn ? 'passed' as const : 'pending' as const,
    }
    return { ...a, weighIn, weighedIn: weighIn.status === 'passed' }
}
