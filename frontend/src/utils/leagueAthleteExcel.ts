import * as XLSX from 'xlsx'
import { normalizeText } from './athleteImport'

export type ExcelTeam = { id: string; teamId?: string; name: string; groupId?: string | null }
export type ExcelAthlete = { coachName?: string; name: string; weightCategory: string | null; teamId?: string | null; ranking?: number | null; club?: string | null }
export function rosterWorkbook(weights: string[], teams: ExcelTeam[], athletes?: ExcelAthlete[]) {
  const wb = XLSX.utils.book_new()
  weights.forEach((w, i) => {
    const rows = athletes
      ? athletes.filter(a => a.weightCategory === w).sort((a,b) => (a.ranking ?? Infinity)-(b.ranking ?? Infinity))
        .map((a, n) => [n+1, a.name, a.club ?? '', w, a.ranking ?? '', teams.find(t => t.teamId === a.teamId)?.name ?? '', a.coachName ?? ''])
      : Array.from({length:16}, () => ['', '', w, '', '', ''])
    const ws = XLSX.utils.aoa_to_sheet([athletes ? ['#','نام','باشگاه','وزن','رنکینگ','تیم','مربی'] : ['نام','باشگاه','وزن','رنکینگ','تیم','مربی'], ...rows])
    ws['!cols'] = (athletes ? [5,22,18,10,10,22,22] : [22,18,10,10,22,22]).map(wch => ({wch}))
    XLSX.utils.book_append_sheet(wb, ws, `وزن ${i+1} (${w})`)
  })
  return wb
}

export function parseRoster(wb: XLSX.WorkBook, weights: string[], teams: ExcelTeam[], existing: ExcelAthlete[] = []) {
  const rows: { sheet: string; row: number; payload: {coachName:string;name:string; clubId:string|null; teamId:string|null; groupId:string|null; weightCategory:string; ranking:number|null} }[] = []
  const errors: string[] = []
  const key = (name:string, teamId?:string|null) => JSON.stringify([normalizeText(name), teamId ?? null])
  const seen = new Set(existing.map(a => key(a.name,a.teamId)))
  for (const sheet of wb.SheetNames) {
    const data = XLSX.utils.sheet_to_json<unknown[]>(wb.Sheets[sheet], {header:1, defval:''})
    const headers = (data[0] ?? []).map(h => normalizeText(h).toLowerCase())
    const col = (...names:string[]) => headers.findIndex(h => names.includes(h))
    const nameCol = col('نام','name'), teamCol = col('تیم','team'), weightCol = col('وزن','weight'), rankCol = col('رنکینگ','ranking'), coachCol = col('مربی','نام مربی','coach')
    if (nameCol < 0) { errors.push(`${sheet}: ستون نام پیدا نشد`); continue }
    data.slice(1).forEach((cells,index) => {
      const name = String(cells[nameCol] ?? '').trim()
      if (!name) return
      const location = `${sheet}، ردیف ${index+2}`
      const fail = (message:string) => errors.push(`${location}: ${message}`)
      const teamName = normalizeText(cells[teamCol])
      const matches = teams.filter(t => normalizeText(t.name) === teamName)
      if (teamName && matches.length !== 1) { fail('نام تیم نامعتبر یا مبهم است؛ ابتدا تیم را در لیگ ثبت کنید'); return }
      const team = teamName ? matches[0] : undefined
      if (team && !team.teamId) { fail('شناسه تیم پیدا نشد'); return }
      const weight = normalizeText(cells[weightCol]) || weights.find(w => normalizeText(sheet).endsWith(`(${w})`) || normalizeText(sheet) === `وزن ${w}`) || ''
      if (!weights.includes(weight)) { fail('وزن نامعتبر است'); return }
      const rankText = normalizeText(cells[rankCol])
      const ranking = rankText ? Number(rankText) : null
      if (ranking !== null && (!Number.isSafeInteger(ranking) || ranking < 1)) { fail('رنکینگ باید عدد صحیح مثبت باشد'); return }
      const coachName=String(cells[coachCol]??'').trim()
      if(!team && !coachName){fail('نام مربی برای بازیکن آزاد الزامی است');return}
      const id = key(name,team?.teamId)
      if (seen.has(id)) { fail('ورزشکار تکراری است'); return }
      seen.add(id)
      rows.push({sheet,row:index+2,payload:{coachName,name,clubId:team?.id ?? null,teamId:team?.teamId ?? null,groupId:team?.groupId ?? null,weightCategory:weight,ranking}})
    })
  }
  return {rows,errors}
}
