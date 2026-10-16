<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useUiStore } from '../../stores/ui'
import { webApi } from '../../webApi'
import type { PoomsaeEvent } from '../../types/poomsae'
import { fa, message } from '../../utils/poomsae'

type Week = { id: string; stage: number; position: number; name: string; eventId: string | null }
type League = { id: string; name: string; seasonName: string; weeks: Week[]; pointsByRank: number[] }
type Ranking = { rank: number; athleteId?: string; name: string; teamName?: string; points: number; gold: number; silver: number; bronze: number; appearances: number }
type Detail = { league: League; standings: { players: Ranking[]; teams: Ranking[] } }

const ui = useUiStore()
const detail = ref<Detail | null>(null)
const events = ref<PoomsaeEvent[]>([])
const busy = ref(false), error = ref(''), notice = ref('')
const backupInput = ref<HTMLInputElement | null>(null)
const stages = computed(() => {
  const weeks = detail.value?.league.weeks || []
  return [...new Set(weeks.map(week => week.stage))].sort((a, b) => a - b)
    .map(number => ({ number, weeks: weeks.filter(week => week.stage === number) }))
})
const linkedCount = computed(() => detail.value?.league.weeks.filter(week => week.eventId).length || 0)
const tabs = [
  { key: 'overview', label: 'نمای کلی', hint: 'وضعیت فصل' },
  { key: 'weeks', label: 'برگزاری هفته‌ها', hint: 'اتصال مسابقات' },
  { key: 'standings', label: 'گزارش و رده‌بندی', hint: 'نتایج فصل' },
  { key: 'backup', label: 'ورودی و خروجی', hint: 'ساختار فصل' },
] as const

function exportStructure() {
  if (!detail.value) return
  const league = detail.value.league
  const data = { kind: 'tkd-poomsae-league-structure', version: 1, name: league.name, seasonName: league.seasonName,
    weeks: league.weeks.map(({ stage, position, eventId }) => ({ stage, position, eventId })) }
  const url = URL.createObjectURL(new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' }))
  const link = document.createElement('a')
  link.href = url; link.download = `poomsae-league-${league.name.replace(/[\\/:*?"<>|]/g, '-')}.json`; link.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
async function importStructure(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const backup = JSON.parse(await file.text())
    const weeks = backup.weeks
    const stageCount = Array.isArray(weeks) ? Math.max(0, ...weeks.map((week: any) => Number(week.stage))) : 0
    const weeksPerStage = Array.isArray(weeks) ? Math.max(0, ...weeks.map((week: any) => Number(week.position))) : 0
    if (backup.kind !== 'tkd-poomsae-league-structure' || backup.version !== 1 || !String(backup.name || '').trim() || !String(backup.seasonName || '').trim() || stageCount < 1 || stageCount > 10 || weeksPerStage < 1 || weeksPerStage > 52 || weeks.length !== stageCount * weeksPerStage || weeks.some((week: any) => !Number.isInteger(week.stage) || !Number.isInteger(week.position) || week.stage < 1 || week.position < 1) || new Set(weeks.map((week: any) => `${week.stage}:${week.position}`)).size !== weeks.length) throw new Error('ساختار فایل لیگ پومسه معتبر نیست.')
    const existingEventIds = weeks.map((week: any) => week.eventId).filter((id: unknown) => typeof id === 'string' && events.value.some(item => item.id === id))
    if (new Set(existingEventIds).size !== existingEventIds.length) throw new Error('یک مسابقه در چند هفتهٔ فایل تکرار شده است.')
    const league = await webApi().call<League>('/poomsae-leagues', 'POST', { name: backup.name, seasonName: backup.seasonName, stageCount, weeksPerStage })
    let linked = 0
    for (const week of weeks) {
      if (!week.eventId || !events.value.some(item => item.id === week.eventId)) continue
      const target = league.weeks.find(item => item.stage === week.stage && item.position === week.position)
      if (!target) continue
      await webApi().call(`/poomsae-leagues/${league.id}/weeks/${target.id}`, 'PUT', { eventId: week.eventId })
      linked++
    }
    ui.poomsaeLeagueId = league.id
    ui.poomsaeLeagueTab = 'overview'
    await load()
    notice.value = `لیگ جدید ساخته شد؛ ${fa(linked)} مسابقهٔ موجود به هفته‌ها متصل شد.`
  } catch (e) { error.value = message(e) }
  finally { busy.value = false; input.value = '' }
}

async function load() {
  if (!ui.poomsaeLeagueId) return
  busy.value = true; error.value = ''
  try {
    detail.value = await webApi().call<Detail>(`/poomsae-leagues/${ui.poomsaeLeagueId}`)
  } catch (e) { error.value = message(e); detail.value = null }
  finally { busy.value = false }
}
async function loadEvents() {
  try { events.value = await webApi().call<PoomsaeEvent[]>('/poomsae-events?limit=200') }
  catch (e) { error.value = message(e) }
}
async function link(week: Week, value: string) {
  if (!detail.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await webApi().call(`/poomsae-leagues/${detail.value.league.id}/weeks/${week.id}`, 'PUT', { eventId: value || null })
    await load()
    notice.value = 'مسابقهٔ هفته ذخیره شد.'
  } catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
function openEvent(id: string) {
  ui.poomsaeEventId = id
  ui.poomsaeDivisionId = ''
  ui.poomsaeTab = 'athletes'
  ui.page = 'poomsae'
}
onMounted(() => { load(); loadEvents() })
watch(() => ui.poomsaeLeagueId, load)
</script>

<template>
  <div class="space-y-5" dir="rtl">
    <header class="overflow-hidden rounded-2xl bg-gradient-to-l from-slate-950 to-teal-900 text-white shadow-lg">
      <div class="flex flex-col gap-4 p-6 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2"><span class="rounded-full bg-teal-300/20 px-3 py-1 text-xs font-bold text-teal-100">لیگ پومسه</span><span v-if="detail" class="text-xs text-slate-300">فصل {{ detail.league.seasonName }}</span></div>
          <h2 class="text-2xl font-black">{{ detail?.league.name || 'لیگ پومسه' }}</h2>
          <p v-if="detail" class="mt-2 text-sm text-teal-100">{{ fa(stages.length) }} مرحله · {{ fa(detail.league.weeks.length) }} هفته · {{ fa(linkedCount) }} مسابقه متصل</p>
        </div>
        <div class="flex gap-2"><button :disabled="busy" class="rounded-xl bg-white/10 px-4 py-2 text-sm hover:bg-white/20 disabled:opacity-50" @click="load">↻ به‌روزرسانی</button><button class="rounded-xl bg-white px-4 py-2 text-sm font-medium text-slate-900" @click="ui.homeSection='leagues'; ui.goHome()">بازگشت به لیگ‌ها</button></div>
      </div>
      <div v-if="detail" class="grid grid-cols-3 border-t border-white/10 bg-white/5 text-center"><div class="p-3"><b class="text-xl">{{ fa(stages.length) }}</b><div class="text-xs text-teal-100">مرحله</div></div><div class="border-x border-white/10 p-3"><b class="text-xl">{{ fa(detail.league.weeks.length) }}</b><div class="text-xs text-teal-100">هفته</div></div><div class="p-3"><b class="text-xl">{{ fa(detail.standings.players.length) }}</b><div class="text-xs text-teal-100">ورزشکار در جدول</div></div></div>
    </header>
    <p v-if="error" class="rounded-xl bg-rose-50 px-4 py-3 text-sm text-rose-700" role="alert">{{ error }}</p>
    <p v-if="notice" class="rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-700" role="status">{{ notice }}</p>
    <nav class="grid gap-2 rounded-2xl border border-slate-200 bg-white p-2 shadow-sm sm:grid-cols-2 lg:grid-cols-4" aria-label="بخش‌های لیگ پومسه"><button v-for="(tab, index) in tabs" :key="tab.key" class="flex items-center gap-3 rounded-xl px-3 py-3 text-right transition" :class="ui.poomsaeLeagueTab===tab.key?'bg-teal-700 text-white shadow-sm':'text-slate-600 hover:bg-slate-50'" @click="ui.poomsaeLeagueTab=tab.key"><span class="flex h-8 w-8 items-center justify-center rounded-lg text-sm font-bold" :class="ui.poomsaeLeagueTab===tab.key?'bg-white/20':'bg-slate-100'">{{ fa(index+1) }}</span><span><b class="block text-sm">{{ tab.label }}</b><small :class="ui.poomsaeLeagueTab===tab.key?'text-teal-100':'text-slate-400'">{{ tab.hint }}</small></span></button></nav>
    <div v-if="busy && !detail" class="rounded-2xl bg-white p-10 text-center text-slate-500">در حال دریافت لیگ…</div>
    <template v-if="detail">
      <section v-if="ui.poomsaeLeagueTab==='overview'" class="grid gap-4 lg:grid-cols-2">
        <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"><h3 class="text-lg font-black">برنامهٔ فصل</h3><p class="mt-1 text-sm text-slate-500">مسابقهٔ هر هفته را از تب برگزاری هفته‌ها انتخاب کنید.</p><div class="mt-5 space-y-3"><div v-for="stage in stages" :key="stage.number" class="rounded-xl bg-slate-50 p-4"><div class="flex justify-between font-bold"><span>مرحلهٔ {{ fa(stage.number) }}</span><span class="text-teal-700">{{ fa(stage.weeks.filter(week=>week.eventId).length) }} از {{ fa(stage.weeks.length) }} هفته</span></div><div class="mt-2 text-xs text-slate-500">{{ stage.weeks.map(week=>week.name).join(' · ') }}</div></div></div></article>
        <article class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"><h3 class="text-lg font-black">رتبه‌بندی فصل</h3><p class="mt-1 text-sm text-slate-500">امتیاز رده‌های نهایی‌شدهٔ مسابقات به‌طور خودکار جمع می‌شود.</p><div class="mt-5 space-y-2"><div v-for="row in detail.standings.players.slice(0,5)" :key="row.athleteId || row.name" class="flex items-center justify-between rounded-xl bg-teal-50 px-4 py-3 text-sm"><span><b class="ml-3 text-teal-700">{{ fa(row.rank) }}</b>{{ row.name }}</span><b>{{ fa(row.points) }} امتیاز</b></div><p v-if="!detail.standings.players.length" class="rounded-xl border border-dashed p-5 text-center text-sm text-slate-500">هنوز نتیجهٔ نهایی ثبت نشده است.</p></div><button class="mt-4 text-sm font-bold text-teal-700" @click="ui.poomsaeLeagueTab='standings'">نمایش جدول کامل ←</button></article>
      </section>
      <section v-else-if="ui.poomsaeLeagueTab==='weeks'" class="space-y-4"><div class="rounded-xl bg-teal-50 p-4 text-sm text-teal-900">هر هفته را به یک مسابقهٔ پومسه وصل کنید. پس از نهایی‌شدن رده‌ها، امتیازها در جدول فصل ظاهر می‌شوند.</div><article v-for="stage in stages" :key="stage.number" class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"><h3 class="mb-4 text-lg font-black">مرحلهٔ {{ fa(stage.number) }}</h3><div class="grid gap-3 md:grid-cols-2"><div v-for="week in stage.weeks" :key="week.id" class="rounded-xl border border-slate-200 p-4"><label class="block text-sm font-bold text-slate-700">{{ week.name }}<select :value="week.eventId || ''" :disabled="busy" class="mt-3 w-full rounded-lg border border-slate-300 bg-white p-2 text-sm" @change="link(week, ($event.target as HTMLSelectElement).value)"><option value="">بدون مسابقه</option><option v-for="event in events" :key="event.id" :value="event.id">{{ event.name }}</option></select></label><button v-if="week.eventId" class="mt-3 text-xs font-bold text-teal-700" @click="openEvent(week.eventId)">ورود به مسابقه ←</button></div></div></article></section>
      <section v-else-if="ui.poomsaeLeagueTab==='standings'" class="space-y-4"><div class="rounded-xl bg-white p-4 text-sm text-slate-600">امتیاز رتبه‌های اول تا چهارم: {{ detail.league.pointsByRank.map(fa).join('، ') }}</div><div class="grid gap-4 lg:grid-cols-2"><article v-for="kind in ['players','teams'] as const" :key="kind" class="overflow-x-auto rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"><h3 class="mb-4 text-lg font-black">{{ kind==='players'?'جدول ورزشکاران':'جدول باشگاه‌ها' }}</h3><table class="w-full text-right text-sm"><thead class="bg-teal-900 text-white"><tr><th class="p-2">رتبه</th><th class="p-2">{{ kind==='players'?'ورزشکار':'باشگاه' }}</th><th class="p-2">امتیاز</th><th class="p-2">طلا</th><th class="p-2">نقره</th><th class="p-2">برنز</th></tr></thead><tbody><tr v-for="row in detail.standings[kind]" :key="row.athleteId || row.name" class="border-b"><td class="p-2">{{ fa(row.rank) }}</td><td class="p-2 font-bold">{{ row.name }}<span v-if="kind==='players'" class="block text-xs font-normal text-slate-500">{{ row.teamName }}</span></td><td class="p-2 font-black text-teal-700">{{ fa(row.points) }}</td><td class="p-2">{{ fa(row.gold) }}</td><td class="p-2">{{ fa(row.silver) }}</td><td class="p-2">{{ fa(row.bronze) }}</td></tr><tr v-if="!detail.standings[kind].length"><td colspan="6" class="p-6 text-center text-slate-500">هنوز نتیجه‌ای ثبت نشده است.</td></tr></tbody></table></article></div></section>
      <section v-else class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"><div class="text-xs font-bold text-teal-700">مدیریت داده‌های فصل</div><h3 class="mt-1 text-xl font-black">ورودی و خروجی لیگ پومسه</h3><p class="mt-2 text-sm leading-7 text-slate-500">فایل JSON ساختار مرحله‌ها و هفته‌ها را ذخیره می‌کند. نتیجه‌ها و ورزشکاران در مسابقات پومسه ثبت می‌شوند؛ هنگام بازیابی، مسابقه‌های موجود در همین سازمان دوباره به هفته‌ها وصل می‌شوند.</p><div class="mt-5 grid gap-4 md:grid-cols-2"><div class="rounded-xl bg-slate-50 p-4"><h4 class="font-bold">خروجی ساختار فصل</h4><p class="my-3 text-xs text-slate-500">نام فصل، هفته‌ها و پیوند مسابقه‌ها</p><button class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-bold text-white" @click="exportStructure">دریافت فایل JSON</button></div><div class="rounded-xl bg-slate-50 p-4"><h4 class="font-bold">ایمپورت ساختار فصل</h4><p class="my-3 text-xs text-slate-500">ساخت یک لیگ پومسهٔ جدید از فایل</p><button :disabled="busy" class="rounded-lg border border-teal-600 px-4 py-2 text-sm font-bold text-teal-800 disabled:opacity-50" @click="backupInput?.click()">انتخاب فایل JSON</button><input ref="backupInput" class="hidden" type="file" accept=".json" @change="importStructure"></div></div></section>
    </template>
  </div>
</template>
