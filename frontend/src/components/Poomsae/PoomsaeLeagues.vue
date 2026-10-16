<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { webApi } from '../../webApi'
import type { PoomsaeEvent } from '../../types/poomsae'
import { fa, message } from '../../utils/poomsae'

type Week = { id: string; stage: number; position: number; name: string; eventId: string | null }
type League = { id: string; name: string; seasonName: string; weeks: Week[]; pointsByRank: number[] }
type Ranking = { rank: number; athleteId?: string; name: string; teamName?: string; points: number; gold: number; silver: number; bronze: number; appearances: number }
type Detail = { league: League; standings: { players: Ranking[]; teams: Ranking[] } }

const props = withDefaults(defineProps<{ events?: PoomsaeEvent[] }>(), { events: () => [] })
const loadedEvents = ref<PoomsaeEvent[]>([])
const availableEvents = computed(() => props.events.length ? props.events : loadedEvents.value)
const leagues = ref<League[]>([])
const detail = ref<Detail | null>(null)
const name = ref(''), seasonName = ref(''), stageCount = ref(1), weeksPerStage = ref(3)
const busy = ref(false), error = ref(''), notice = ref('')

async function load() {
  try { leagues.value = await webApi().call<League[]>('/poomsae-leagues') }
  catch (e) { error.value = message(e) }
}
async function open(id: string) {
  busy.value = true; error.value = ''
  try { detail.value = await webApi().call<Detail>(`/poomsae-leagues/${id}`) }
  catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
async function create() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const league = await webApi().call<League>('/poomsae-leagues', 'POST', {
      name: name.value.trim(), seasonName: seasonName.value.trim(), stageCount: stageCount.value, weeksPerStage: weeksPerStage.value,
    })
    name.value = ''; seasonName.value = ''
    await load(); await open(league.id)
    notice.value = 'لیگ پومسه ساخته شد.'
  } catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
async function link(week: Week, value: string) {
  if (!detail.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await webApi().call(`/poomsae-leagues/${detail.value.league.id}/weeks/${week.id}`, 'PUT', { eventId: value || null })
    await open(detail.value.league.id); await load()
    notice.value = 'مسابقهٔ هفته ذخیره شد.'
  } catch (e) { error.value = message(e) }
  finally { busy.value = false }
}
onMounted(async () => {
  await load()
  try { loadedEvents.value = await webApi().call<PoomsaeEvent[]>('/poomsae-events?limit=200') } catch { /* league list shows its own permission error */ }
})
</script>

<template>
  <section class="space-y-4 rounded-2xl border border-teal-200 bg-teal-50/40 p-5">
    <div><h3 class="text-xl font-black text-teal-950">لیگ‌های پومسه</h3><p class="mt-1 text-sm text-teal-800">هر هفته را به یک مسابقهٔ پومسه وصل کنید. جدول فصل از رده‌های نهایی‌شده محاسبه می‌شود.</p></div>
    <div class="grid gap-2 rounded-xl bg-white p-4 sm:grid-cols-4">
      <label class="text-xs font-bold text-slate-600">نام لیگ<input v-model="name" class="mt-1 w-full rounded-lg border p-2 text-sm" placeholder="لیگ پومسه استان"></label>
      <label class="text-xs font-bold text-slate-600">فصل<input v-model="seasonName" class="mt-1 w-full rounded-lg border p-2 text-sm" placeholder="۱۴۰۵"></label>
      <label class="text-xs font-bold text-slate-600">تعداد مراحل<input v-model.number="stageCount" type="number" min="1" max="10" class="mt-1 w-full rounded-lg border p-2 text-sm"></label>
      <label class="text-xs font-bold text-slate-600">هفته در هر مرحله<input v-model.number="weeksPerStage" type="number" min="1" max="52" class="mt-1 w-full rounded-lg border p-2 text-sm"></label>
      <button :disabled="busy || !name.trim() || !seasonName.trim()" class="rounded-lg bg-teal-700 px-4 py-2 text-sm font-bold text-white disabled:opacity-40 sm:col-span-4" @click="create">ساخت لیگ پومسه</button>
    </div>
    <p v-if="error" class="rounded-lg bg-rose-50 p-3 text-sm text-rose-700" role="alert">{{ error }}</p>
    <p v-if="notice" class="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700" role="status">{{ notice }}</p>
    <div class="flex flex-wrap gap-2"><button v-for="league in leagues" :key="league.id" class="rounded-lg border px-3 py-2 text-sm font-bold" :class="detail?.league.id === league.id ? 'border-teal-700 bg-teal-700 text-white' : 'border-teal-200 bg-white text-teal-900'" @click="open(league.id)">{{ league.name }} · {{ league.seasonName }}</button></div>
    <div v-if="detail" class="space-y-5 rounded-xl bg-white p-4">
      <div class="flex flex-wrap items-center justify-between gap-2"><div><h4 class="text-lg font-black">{{ detail.league.name }}</h4><p class="text-xs text-slate-500">امتیاز رتبه‌های اول تا چهارم: {{ detail.league.pointsByRank.map(fa).join('، ') }}</p></div><button :disabled="busy" class="rounded-lg border px-3 py-2 text-sm" @click="open(detail.league.id)">به‌روزرسانی جدول</button></div>
      <div class="grid gap-2 md:grid-cols-2"><label v-for="week in detail.league.weeks" :key="week.id" class="rounded-lg border border-slate-200 p-3 text-xs font-bold text-slate-600">مرحله {{ fa(week.stage) }} · {{ week.name }}<select :value="week.eventId || ''" :disabled="busy" class="mt-2 w-full rounded-lg border bg-white p-2 text-sm" @change="link(week, ($event.target as HTMLSelectElement).value)"><option value="">بدون مسابقه</option><option v-for="event in availableEvents" :key="event.id" :value="event.id">{{ event.name }}</option></select></label></div>
      <div class="grid gap-4 lg:grid-cols-2"><div v-for="kind in ['players','teams'] as const" :key="kind" class="overflow-x-auto"><h5 class="mb-2 font-bold">{{ kind === 'players' ? 'جدول ورزشکاران فصل' : 'جدول باشگاه‌های فصل' }}</h5><table class="w-full text-right text-sm"><thead class="bg-teal-900 text-white"><tr><th class="p-2">رتبه</th><th class="p-2">{{ kind === 'players' ? 'ورزشکار' : 'باشگاه' }}</th><th class="p-2">امتیاز</th><th class="p-2">طلا</th><th class="p-2">نقره</th><th class="p-2">برنز</th></tr></thead><tbody><tr v-for="row in detail.standings[kind]" :key="row.athleteId || row.name" class="border-b"><td class="p-2">{{ fa(row.rank) }}</td><td class="p-2 font-bold">{{ row.name }}<span v-if="kind === 'players'" class="block text-xs font-normal text-slate-500">{{ row.teamName }}</span></td><td class="p-2 font-bold">{{ fa(row.points) }}</td><td class="p-2">{{ fa(row.gold) }}</td><td class="p-2">{{ fa(row.silver) }}</td><td class="p-2">{{ fa(row.bronze) }}</td></tr><tr v-if="!detail.standings[kind].length"><td colspan="6" class="p-4 text-center text-slate-500">هنوز نتیجهٔ نهایی برای این لیگ ثبت نشده است.</td></tr></tbody></table></div></div>
    </div>
  </section>
</template>
