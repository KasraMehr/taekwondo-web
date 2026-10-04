import { isValidJalaaliDate, toGregorian, toJalaali } from 'jalaali-js'
import { ApiError } from '../api'
import type { Scores } from '../types/poomsae'

export const statusLabels = { draft: 'ثبت‌نام', drawn: 'قرعه‌کشی شده', scoring: 'در حال اجرا', finalized: 'نهایی شده' }
export const attendanceLabels = { active: 'حاضر', absent: 'غایب', withdrawn: 'انصراف' }
export const genderLabels = { male: 'آقایان', female: 'بانوان', mixed: 'مختلط' }
export const fa = (value: string | number | null | undefined) => value == null ? '—' : String(value).replace(/[0-9]/g, d => '۰۱۲۳۴۵۶۷۸۹'[Number(d)])
export const digits = (value: string) => value.replace(/[۰-۹]/g, d => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d))).replace(/[٠-٩]/g, d => String('٠١٢٣٤٥٦٧٨٩'.indexOf(d))).replace(/٫/g, '.').trim()
export function hundredths(value: string | null): number | null {
  if (value == null || !digits(value)) return null
  const s = digits(value)
  if (!/^\d{1,4}(\.\d{1,2})?$/.test(s)) throw new Error('نمره را با حداکثر دو رقم اعشار وارد کنید.')
  const [a, b = ''] = s.split('.')
  return Number(a) * 100 + Number(b.padEnd(2, '0'))
}
export const decimal = (value: number) => `${Math.floor(value / 100)}.${String(value % 100).padStart(2, '0')}`
export function normalizeScore(value: string | null, max: string): string | null {
  const n = hundredths(value)
  if (n == null) return null
  if (n > hundredths(max)!) throw new Error(`نمره باید بین صفر و ${fa(max)} باشد.`)
  return decimal(n)
}
export function sum(...values: (string | null)[]): string | null {
  try { const ns = values.map(hundredths); return ns.some(n => n == null) ? null : decimal(ns.reduce<number>((s, n) => s + n!, 0)) } catch { return null }
}
export function scoreKey(s: Scores): string {
  const f = (v: Scores['form1']) => [v.code ?? null, v.accuracy ? digits(v.accuracy) : null, v.presentation ? digits(v.presentation) : null]
  return JSON.stringify([f(s.form1), f(s.form2), s.reason || ''])
}
export const copyScores = (s: Scores): Scores => ({ form1: { ...s.form1 }, form2: { ...s.form2 }, reason: '' })
export function isoFromJalali(value: string): string {
  const m = /^(\d{4})[/-](\d{1,2})[/-](\d{1,2})$/.exec(digits(value))
  if (!m || !isValidJalaaliDate(+m[1], +m[2], +m[3])) throw new Error('تاریخ شمسی معتبر وارد کنید؛ مانند ۱۴۰۵/۰۷/۱۱.')
  const { gy, gm, gd } = toGregorian(+m[1], +m[2], +m[3])
  return `${gy}-${String(gm).padStart(2, '0')}-${String(gd).padStart(2, '0')}`
}
export function jalaliFromIso(value: string | null | undefined): string {
  if (!value) return ''
  const [y, m, d] = value.slice(0, 10).split('-').map(Number)
  if (!y || !m || !d) return ''
  const { jy, jm, jd } = toJalaali(y, m, d)
  return fa(`${jy}/${String(jm).padStart(2, '0')}/${String(jd).padStart(2, '0')}`)
}
export function todayJalali(): string {
  const n = new Date(); return jalaliFromIso(`${n.getFullYear()}-${String(n.getMonth() + 1).padStart(2, '0')}-${String(n.getDate()).padStart(2, '0')}`)
}
export function message(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 409) return 'اطلاعات یا وضعیت مسابقه تغییر کرده است. اطلاعات تازه را دریافت و پیش از ذخیره دوباره بررسی کنید.'
    if (error.status === 403) return 'برای این کار دسترسی ندارید.'
    if (error.status === 401) return 'نشست شما پایان یافته است. دوباره وارد شوید.'
    if (error.status === 404) return 'مسابقه یا رده پیدا نشد.'
    const cases: [string, string][] = [
      ['already registered', 'این ورزشکار قبلاً در این رده ثبت شده است.'],
      ['differs from athlete profile', 'تاریخ تولد یا جنسیت با پروفایل ورزشکار یکسان نیست.'],
      ['outside birth date bounds', 'تاریخ تولد در بازهٔ این رده نیست.'],
      ['correction', 'برای اصلاح اطلاعات، علت تغییر را وارد کنید.'],
      ['reason', 'علت تغییر را وارد کنید.'],
      ['locked', 'این بخش در وضعیت فعلی قفل است.'],
      ['inactive', 'پروفایل ورزشکار غیرفعال است.'],
      ['name and valid gender', 'نام و جنسیت معتبر ورزشکار را وارد کنید.'],
      ['select allowed form', 'دو فرم مجاز را انتخاب کنید.'],
      ['repeated forms', 'انتخاب فرم تکراری مجاز نیست.'],
      ['exceeds', 'نمره از سقف مجاز بیشتر است.'],
    ]
    for (const [key, text] of cases) if (error.message.includes(key)) return text
    return `ثبت اطلاعات انجام نشد: ${error.message}`
  }
  if (error instanceof TypeError) return 'ارتباط با سرور برقرار نشد. ورودی‌های شما حفظ شده‌اند؛ دوباره تلاش کنید.'
  return error instanceof Error ? error.message : 'عملیات انجام نشد.'
}
