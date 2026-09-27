// src/utils/teamSlots.ts
import type { AgeCategory } from '../data/categories'

/** تعداد اوزان (اسلات) هر رده سنی در مسابقات تیمی */
const SLOT_COUNT_BY_AGE: Record<string, 8 | 10> = {
    'خردسالان': 10,
    'نونهالان': 10,
    'نوجوانان': 10,
    'امیدها': 8,
    'بزرگسالان': 8,
}

export const DEFAULT_SLOT_COUNT: 8 | 10 = 8

export function slotCountFor(ageCategory: AgeCategory): 8 | 10 {
    return SLOT_COUNT_BY_AGE[ageCategory as string] ?? DEFAULT_SLOT_COUNT
}
