export const AGE_CATEGORIES = ['خردسالان', 'نونهالان', 'نوجوانان', 'امیدها', 'بزرگسالان'] as const
export type AgeCategory = (typeof AGE_CATEGORIES)[number]

export type Gender = 'male' | 'female'
export const GENDER_LABELS: Record<Gender, string> = { male: 'پسران / مردان', female: 'دختران / زنان' }

// اوزان رسمی WT برای نوجوانان/جوانان/بزرگسالان
// خردسالان و نونهالان طبق بخشنامه‌های رایج فدراسیون — در صورت نیاز همینجا ویرایش کن
export const WEIGHT_CATEGORIES: Record<AgeCategory, Record<Gender, string[]>> = {
    'خردسالان': {
        male:   ['-26', '-28', '-30', '-33', '-36', '-40', '-44', '-48', '-52', '+52'],
        female: ['-26', '-28', '-30', '-33', '-36', '-40', '-44', '-48', '-52', '+52'],
    },
    'نونهالان': {
        male:   ['-33', '-37', '-41', '-45', '-49', '-53', '-57', '-61', '-65', '+65'],
        female: ['-29', '-33', '-37', '-41', '-44', '-47', '-51', '-55', '-59', '+59'],
    },
    'نوجوانان': {
        male:   ['-45', '-48', '-51', '-55', '-59', '-63', '-68', '-73', '-78', '+78'],
        female: ['-42', '-44', '-46', '-49', '-52', '-55', '-59', '-63', '-68', '+68'],
    },
    'امیدها': {
        male:   ['-54', '-58', '-63', '-68', '-74', '-80', '-87', '+87'],
        female: ['-46', '-49', '-53', '-57', '-62', '-67', '-73', '+73'],
    },
    'بزرگسالان': {
        male:   ['-54', '-58', '-63', '-68', '-74', '-80', '-87', '+87'],
        female: ['-46', '-49', '-53', '-57', '-62', '-67', '-73', '+73'],
    },
}

export function weightOptions(age: AgeCategory, gender: Gender): string[] {
    return WEIGHT_CATEGORIES[age]?.[gender] ?? []
}

export function isValidWeightCategory(age: AgeCategory, gender: Gender, value: string): boolean {
    return weightOptions(age, gender).includes(value)
}