<script setup lang="ts">
import { ref } from 'vue'
import * as XLSX from 'xlsx'
import { webApi, loadWebLeagues } from '../../webApi'
import { useLeagueStore } from '../../stores/league'

const store = useLeagueStore()
const selectedId = ref('')
const mode = ref<'add' | 'replace'>('add')
const fileInput = ref<HTMLInputElement | null>(null)
const busy = ref(false), error = ref(''), notice = ref('')
const safeName = (value: string) => value.replace(/[\\/:*?"<>|]/g, '-')

async function exportLeague() {
  if (!selectedId.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const backup = await webApi().call<any>(`/leagues/${selectedId.value}/export`)
    const workbook = XLSX.utils.book_new()
    const raw = JSON.stringify(backup)
    const sheet = (rows: any[]) => XLSX.utils.json_to_sheet(rows.length ? rows : [{}])
    const chunks = Array.from({ length: Math.ceil(raw.length / 30000) }, (_, seq) => ({ seq, part: raw.slice(seq * 30000, (seq + 1) * 30000) }))
    XLSX.utils.book_append_sheet(workbook, sheet(chunks), 'Data')
    XLSX.utils.book_append_sheet(workbook, sheet([{
      name: backup.league.name, season: backup.league.settings.seasonName,
      stages: backup.league.stages.length, athletes: backup.league.athletes.length,
      tournaments: backup.tournaments.length,
    }]), 'Info')
    XLSX.utils.book_append_sheet(workbook, sheet(backup.league.stages.flatMap((stage: any) => stage.weeks.map((week: any) => ({ stage: stage.name, week: week.name, tournament: backup.tournaments.find((t: any) => t.weekId === week.id)?.name || '' })))), 'Weeks')
    XLSX.utils.book_append_sheet(workbook, sheet(backup.league.athletes.map((a: any) => ({ name: a.name, weightCategory: a.weightCategory, club: a.clubName }))), 'Athletes')
    XLSX.writeFile(workbook, `league-${safeName(backup.league.name)}.xlsx`)
    notice.value = 'فایل لیگ از داده‌های ذخیره‌شدهٔ سرور ساخته شد.'
  } catch (e: any) { error.value = e?.message || 'ساخت خروجی لیگ ناموفق بود' }
  finally { busy.value = false }
}

function legacyToServerBundle(bundle: any) {
  const l = bundle.league
  if (l.format === 'team') throw new Error('فایل قدیمی لیگ تیمی اطلاعات کامل مواجهه‌ها را ندارد. برای انتقال آن، از نسخهٔ جدید خروجی بگیرید.')
  const tournamentIds = new Set((bundle.tournaments || []).map((t: any) => t.id))
  if ((l.season?.stages || []).some((stage: any) => (stage.weeks || []).some((week: any) => week.tournamentId && !tournamentIds.has(week.tournamentId)))) {
    throw new Error('فایل قدیمی همهٔ مسابقات هفته‌ها را ندارد؛ از نسخهٔ جدید خروجی کامل لیگ بگیرید.')
  }
  const teams = (l.clubs || []).map((club: any) => ({ id: club.teamId || club.id, clubId: club.id, clubName: club.name, groupId: club.groupId || null, active: club.active !== false }))
  const stages = (l.season?.stages || []).map((stage: any) => ({
    id: stage.id, name: stage.name, position: stage.order, status: stage.completed ? 'completed' : stage.order === 1 ? 'active' : 'pending',
    promotedTeamIds: (stage.promotedClubs || []).map((value: string) => teams.find((team: any) => team.id === value || team.clubName === value)?.id).filter(Boolean),
    weeks: (stage.weeks || []).map((week: any) => ({ id: week.id, name: week.name, position: week.order, dateRange: week.dateRange || null, locked: Boolean(week.locked) })),
  }))
  const league = {
    id: l.id, name: l.name, gender: l.gender, ageCategory: l.ageCategory, status: l.season?.completed ? 'completed' : 'draft',
    settings: { seasonName: l.season?.name || 'فصل جاری', scoring: { gold: l.season?.scoringConfig?.gold ?? 10, silver: l.season?.scoringConfig?.silver ?? 6, bronze: l.season?.scoringConfig?.bronze ?? 3, weighInPoint: l.season?.scoringConfig?.weighInPoint ?? 1, winPoint: l.season?.scoringConfig?.winPoint ?? 2, countWeighIn: l.season?.scoringConfig?.countWeighIn ?? true, countWin: l.season?.scoringConfig?.countWin ?? true }, teamsToPromote: l.promotionRules?.teamsToPromote ?? 2, autoPromote: l.promotionRules?.autoPromote ?? true },
    stages, groups: (l.season?.groups || []).map((group: any) => ({ id: group.id, name: group.name, position: group.order })), teams,
    athletes: (l.athletes || []).map((athlete: any) => ({ id: athlete.id, profileId: athlete.profileId || '', name: athlete.name, weightCategory: athlete.weightCategory, teamId: athlete.teamId || null, groupId: athlete.groupId || null, clubName: athlete.club || '', coachName: athlete.coachName || '', ranking: athlete.ranking || null, active: athlete.isActive !== false, createdAt: athlete.createdAt, updatedAt: athlete.updatedAt })),
  }
  const tournaments = (bundle.tournaments || []).map((t: any) => ({
    id: t.id, name: t.name, date: t.date?.length === 10 ? `${t.date}T00:00:00Z` : t.date, courts: t.courts,
    gender: t.gender, ageCategory: t.ageCategory, format: t.format || 'grandPrix',
    athletes: (t.athletes || []).map((a: any) => ({ id: a.id, profileId: a.profileId || null, coach: a.coach || '', number: a.number || 0, name: a.name, club: a.club, weightCategory: a.weightCategory, ranking: a.ranking || null, weighedIn: Boolean(a.weighedIn), weighIn: a.weighIn || null })),
    matches: t.matches || [], courtAssignment: t.courtAssignment || {}, eliminationMatches: t.eliminationMatches || [],
    leagueId: l.id, stageOrder: t.stageOrder || null, weekId: t.weekId || null, groupId: t.groupId || null,
    settings: t.settings || null, createdAt: t.createdAt, updatedAt: t.updatedAt,
  }))
  return { kind: 'tkd-league-server-bundle', version: 1, league, tournaments }
}

async function readBackup(file: File) {
  if (/\.json$/i.test(file.name)) return JSON.parse(await file.text())
  const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array' })
  if (!workbook.Sheets.Data) throw new Error('شیت Data در فایل لیگ پیدا نشد.')
  const rows = XLSX.utils.sheet_to_json<{ seq: number; part: string }>(workbook.Sheets.Data)
  if (!rows.length) throw new Error('شیت Data خالی است.')
  return JSON.parse(rows.sort((a, b) => Number(a.seq) - Number(b.seq)).map(row => row.part || '').join(''))
}

async function importLeague(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    let backup = await readBackup(file)
    if (backup.kind === 'tkd-league-bundle') backup = legacyToServerBundle(backup)
    if (backup.kind !== 'tkd-league-server-bundle' || backup.version !== 1 || !backup.league?.id) throw new Error('ساختار فایل لیگ معتبر نیست.')
    const created = await webApi().call<any>('/leagues/import', 'POST', { mode: mode.value, backup })
    store.replaceFromBackend(await loadWebLeagues())
    store.selectLeague(created.id)
    selectedId.value = created.id
    notice.value = 'لیگ و مسابقاتش در سرور ذخیره شدند.'
  } catch (e: any) { error.value = e?.message || 'ایمپورت لیگ ناموفق بود' }
  finally { busy.value = false; input.value = '' }
}
</script>

<template>
  <div class="mb-5 rounded-2xl border border-amber-200 bg-white p-4 shadow-sm">
    <div class="mb-3"><h3 class="font-black text-slate-900">پشتیبان لیگ</h3><p class="mt-1 text-xs text-slate-500">خروجی از سرور خوانده می‌شود و ایمپورت پس از ذخیرهٔ کامل در سرور تأیید می‌شود.</p></div>
    <div class="flex flex-wrap items-end gap-2">
      <label class="min-w-48 flex-1 text-xs font-bold text-slate-600">انتخاب لیگ برای خروجی<select v-model="selectedId" class="mt-1 block w-full rounded-lg border p-2 text-sm"><option value="">انتخاب کنید</option><option v-for="league in store.leagues" :key="league.id" :value="league.id">{{ league.name }}</option></select></label>
      <button :disabled="busy || !selectedId" class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-bold text-white disabled:opacity-40" @click="exportLeague">خروجی اکسل لیگ</button>
      <label class="text-xs font-bold text-slate-600">روش ایمپورت<select v-model="mode" class="mt-1 block rounded-lg border p-2 text-sm"><option value="add">افزودن به‌عنوان لیگ جدید</option><option value="replace">جایگزینی لیگ با شناسهٔ فایل</option></select></label>
      <button :disabled="busy" class="rounded-lg border border-amber-400 px-4 py-2 text-sm font-bold text-amber-900 disabled:opacity-40" @click="fileInput?.click()">ایمپورت اکسل / JSON</button>
      <input ref="fileInput" type="file" accept=".xlsx,.xls,.json" class="hidden" @change="importLeague">
    </div>
    <p v-if="error" class="mt-3 rounded-lg bg-rose-50 p-2 text-sm text-rose-700" role="alert">{{ error }}</p>
    <p v-if="notice" class="mt-3 rounded-lg bg-emerald-50 p-2 text-sm text-emerald-700" role="status">{{ notice }}</p>
  </div>
</template>
