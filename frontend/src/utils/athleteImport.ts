import * as XLSX from "xlsx"

export type Gender = "male" | "female"

/** فیلدهای شناخته‌شدهٔ ستون‌ها */
export type AthleteField =
    | "firstName" | "lastName" | "fullName" | "nationalId"
    | "club" | "weight" | "birthDate" | "gender" | "memberCode" | "beltDegree"

/** نام‌های مجاز هر ستون. کلیدها با matchKey مقایسه می‌شوند (بی‌فاصله، بی‌نیم‌فاصله) */
const HEADER_ALIASES: Record<AthleteField, string[]> = {
    firstName:  ["نام", "firstname", "first name", "name"],
    lastName:   ["نامخانوادگی", "فامیل", "فامیلی", "lastname", "last name", "family"],
    fullName:   ["نامونامخانوادگی", "نامکامل", "fullname", "full name"],
    nationalId: ["کدملی", "کدملى", "شمارهملی", "کدملیورزشکار", "nationalid", "national id", "nid"],
    club:       ["باشگاه", "نامباشگاه", "تیم", "club", "team"],
    weight:     ["وزن", "وزنکشی", "weight"],
    birthDate:  ["تاریختولد", "تولد", "birthdate", "birth date", "dob"],
    gender:     ["جنسیت", "جنس", "gender", "sex"],
    memberCode: ["شمارهعضویت", "کدعضویت", "membercode", "code"],
    beltDegree: ["کمربند", "درجهکمربند", "دان", "belt", "beltdegree"],
}

const DIGIT_MAP: Record<string, string> = {}
"۰۱۲۳۴۵۶۷۸۹".split("").forEach((d, i) => (DIGIT_MAP[d] = String(i)))
"٠١٢٣٤٥٦٧٨٩".split("").forEach((d, i) => (DIGIT_MAP[d] = String(i)))

/** ارقام فارسی/عربی → لاتین، یکسان‌سازی ی/ک، حذف اعراب و کشیده، فشرده‌سازی فاصله. نیم‌فاصله حفظ می‌شود. */
export function normalizeText(value: unknown): string {
    if (value === null || value === undefined) return ""
    if (value instanceof Date) return formatDate(value)
    return String(value)
        .replace(/[۰-۹٠-٩]/g, d => DIGIT_MAP[d] ?? d)
        .replace(/[يﻱﻲ]/g, "ی")
        .replace(/[كﻙﻛ]/g, "ک")
        .replace(/[ةۀ]/g, "ه")
        .replace(/[ؤ]/g, "و")
        .replace(/[أإآ]/g, "ا")
        .replace(/[\u064B-\u065F\u0640\u0670]/g, "") // اعراب + کشیده
        .replace(/[\u200e\u200f\u202a-\u202e\u2066-\u2069]/g, "") // کنترل‌های جهت
        .replace(/\s+/g, " ")
        .trim()
}

/** کلید مقایسه: بدون فاصله و نیم‌فاصله. برای تطبیق «علی‌رضا» با «علی رضا» */
export function matchKey(value: unknown): string {
    return normalizeText(value).replace(/[\s\u200c]/g, "").toLowerCase()
}

/** عدد از سلول. ممیز فارسی (٫) و جداکنندهٔ هزار را هم می‌فهمد. */
export function parseNumber(value: unknown): number | null {
    if (typeof value === "number") return Number.isFinite(value) ? value : null
    const s = normalizeText(value).replace(/[٫,]/g, ".").replace(/[^\d.\-]/g, "")
    if (!s) return null
    const n = Number.parseFloat(s)
    return Number.isFinite(n) ? n : null
}

export function parseGender(value: unknown): Gender | null {
    const k = matchKey(value)
    if (!k) return null
    if (["مرد", "آقا", "پسر", "male", "m", "1", "مردان"].includes(k)) return "male"
    if (["زن", "خانم", "دختر", "female", "f", "2", "زنان", "بانو", "بانوان"].includes(k)) return "female"
    return null
}

/**
 * کد ملی ایران: ۱۰ رقم + رقم کنترلی.
 * اکسل صفرهای ابتدایی را می‌خورد، پس تا ۱۰ رقم padStart می‌کنیم.
 */
export function normalizeNationalId(value: unknown): string {
    const digits = normalizeText(value).replace(/\D/g, "")
    return digits.length > 0 && digits.length <= 10 ? digits.padStart(10, "0") : digits
}

export function isValidNationalId(id: string): boolean {
    if (!/^\d{10}$/.test(id)) return false
    if (/^(\d)\1{9}$/.test(id)) return false // ۱۱۱۱۱۱۱۱۱۱ و مشابه
    let sum = 0
    for (let i = 0; i < 9; i++) sum += Number(id[i]) * (10 - i)
    const r = sum % 11
    const check = Number(id[9])
    return r < 2 ? check === r : check === 11 - r
}

function pad2(n: number) { return String(n).padStart(2, "0") }
function formatDate(d: Date) { return `${d.getFullYear()}/${pad2(d.getMonth() + 1)}/${pad2(d.getDate())}` }

/**
 * تاریخ تولد را به شکل نرمال `YYYY/MM/DD` برمی‌گرداند و سال را جدا می‌کند.
 * تشخیص جلالی/میلادی بر اساس بازهٔ سال. تبدیل تقویم انجام نمی‌شود — همان‌طور که وارد شده ذخیره می‌شود.
 */
export function parseBirthDate(value: unknown): { raw: string; year: number | null; calendar: "jalali" | "gregorian" | "unknown" } {
    const raw = normalizeText(value)
    if (!raw) return { raw: "", year: null, calendar: "unknown" }
    const m = raw.match(/(\d{4})\D+(\d{1,2})\D+(\d{1,2})/) ?? raw.match(/(\d{4})/)
    const year = m ? Number(m[1]) : null
    const normalized = m && m.length === 4 ? `${m[1]}/${pad2(Number(m[2]))}/${pad2(Number(m[3]))}` : raw
    const calendar = year === null ? "unknown" : year >= 1200 && year <= 1500 ? "jalali" : year >= 1900 && year <= 2200 ? "gregorian" : "unknown"
    return { raw: normalized, year, calendar }
}

// ─── پارس فایل ───

export interface RawSheetRow {
    /** شمارهٔ واقعی ردیف در اکسل، برای پیام خطا */
    excelRow: number
    cells: Partial<Record<AthleteField, unknown>>
}

export interface WorkbookParseResult {
    sheetName: string
    sheetNames: string[]
    /** ستون‌های شناسایی‌شده */
    mappedFields: AthleteField[]
    /** هدرهایی که به هیچ فیلدی نخوردند — برای هشدار در UI */
    unknownHeaders: string[]
    rows: RawSheetRow[]
}

const ALIAS_LOOKUP = new Map<string, AthleteField>()
for (const [field, aliases] of Object.entries(HEADER_ALIASES) as [AthleteField, string[]][]) {
    for (const a of aliases) ALIAS_LOOKUP.set(matchKey(a), field)
}

/**
 * فایل اکسل/CSV را می‌خواند و ردیف‌های خام برمی‌گرداند.
 * ردیف هدر خودکار پیدا می‌شود (اولین ردیفی که ≥۲ ستون شناخته‌شده دارد) تا فایل‌هایی
 * که چند سطر عنوان/لوگو بالای جدول دارند هم کار کنند.
 */
export function parseAthleteWorkbook(
    data: ArrayBuffer | Uint8Array,
    options: { sheet?: string } = {},
): WorkbookParseResult {
    const wb = XLSX.read(data instanceof Uint8Array ? data : new Uint8Array(data), {
        type: "array",
        cellDates: true,
        cellText: false,
        // فرمول‌ها را ارزیابی نمی‌کنیم؛ فقط مقدار ذخیره‌شده
    })
    const sheetName = options.sheet ?? wb.SheetNames[0]
    const ws = sheetName ? wb.Sheets[sheetName] : undefined
    if (!ws) throw new Error(`شیت «${sheetName ?? "?"}» در فایل پیدا نشد.`)

    const matrix = XLSX.utils.sheet_to_json<unknown[]>(ws, {
        header: 1,
        blankrows: false,
        defval: "",
        raw: true,
    })

    let headerIndex = -1
    let mapping: (AthleteField | null)[] = []
    for (let i = 0; i < Math.min(matrix.length, 20); i++) {
        const candidate = (matrix[i] ?? []).map(c => ALIAS_LOOKUP.get(matchKey(c)) ?? null)
        if (candidate.filter(Boolean).length >= 2) {
            headerIndex = i
            mapping = candidate
            break
        }
    }
    if (headerIndex === -1) {
        throw new Error("ردیف عنوان پیدا نشد. فایل باید ستون‌هایی مثل «نام»، «نام خانوادگی» و «باشگاه» داشته باشد.")
    }

    const headerRow = matrix[headerIndex] ?? []
    const unknownHeaders = headerRow
        .map((c, i) => (mapping[i] ? null : normalizeText(c)))
        .filter((v): v is string => !!v)

    const rows: RawSheetRow[] = []
    for (let i = headerIndex + 1; i < matrix.length; i++) {
        const row = matrix[i] ?? []
        const cells: Partial<Record<AthleteField, unknown>> = {}
        let hasValue = false
        mapping.forEach((field, col) => {
            if (!field) return
            const v = row[col]
            if (v !== "" && v !== null && v !== undefined) hasValue = true
            cells[field] = v
        })
        if (!hasValue) continue // ردیف خالی
        rows.push({ excelRow: i + 1, cells })
    }

    return {
        sheetName,
        sheetNames: wb.SheetNames,
        mappedFields: mapping.filter((f): f is AthleteField => !!f),
        unknownHeaders,
        rows,
    }
}

/** فایل نمونه برای دانلود از UI */
export function buildAthleteTemplate(): XLSX.WorkBook {
    const headers = ["نام", "نام خانوادگی", "کد ملی", "باشگاه", "وزن", "جنسیت", "تاریخ تولد", "کمربند"]
    const sample = ["کسری", "مهرعلی‌زاد", "0012345678", "باشگاه نمونه", "68", "مرد", "1385/04/12", "دان ۱"]
    const ws = XLSX.utils.aoa_to_sheet([headers, sample])
    ws["!cols"] = headers.map(h => ({ wch: Math.max(12, h.length + 4) }))
    const wb = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb, ws, "ورزشکاران")
    return wb
}
