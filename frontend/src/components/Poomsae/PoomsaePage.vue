<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useUiStore, type PoomsaeTab } from '../../stores/ui'
import { usePoomsaeStore } from '../../stores/poomsae'
import { webApi } from '../../webApi'
import type { PoomsaeEvent } from '../../types/poomsae'
import { fa, genderLabels, jalaliFromIso, message, statusLabels } from '../../utils/poomsae'
import PoomsaeAthletes from './PoomsaeAthletes.vue'
import PoomsaeDivisionForm from './PoomsaeDivisionForm.vue'
import PoomsaeDraw from './PoomsaeDraw.vue'
import PoomsaeEventForm from './PoomsaeEventForm.vue'
import PoomsaeScores from './PoomsaeScores.vue'
import PoomsaeStandings from './PoomsaeStandings.vue'
import PoomsaePrint from './PoomsaePrint.vue'
import './poomsae.css'

const ui = useUiStore(), store = usePoomsaeStore()
const loading = ref(false), error = ref(''), divisionForm = ref(false), editDivision = ref(false), eventForm = ref(false)
const history = ref<{ id: number; action: string; reason: string; createdAt: string }[] | null>(null)
const historyBusy = ref(false)
const tabs: { key: PoomsaeTab; label: string }[] = [{ key: 'athletes', label: 'ورزشکاران' }, { key: 'draw', label: 'قرعه‌کشی' }, { key: 'scores', label: 'ثبت نمرات' }, { key: 'standings', label: 'رتبه‌بندی' }, { key: 'print', label: 'چاپ و PDF' }, { key: 'settings', label: 'تنظیمات' }]
const activeCount = computed(() => store.division?.entries.filter(e => e.status === 'active').length || 0)
let request = 0
async function load() {
  const ticket = ++request; loading.value = true; error.value = ''; history.value = null
  try { await store.open(ui.poomsaeEventId, ui.poomsaeDivisionId); if (ticket === request) ui.poomsaeDivisionId = store.division?.id || '' } catch (e) { if (ticket === request) error.value = message(e) } finally { if (ticket === request) loading.value = false }
}
async function changeDivision(id: string) {
  if (store.busy || id === store.division?.id) return
  loading.value = true; error.value = ''; history.value = null
  try { await store.selectDivision(id); ui.poomsaeDivisionId = id } catch (e) { error.value = message(e) } finally { loading.value = false }
}
async function refresh() { error.value = ''; loading.value = true; try { if (store.division) await store.refreshDivision(); else await load() } catch (e) { error.value = message(e) } finally { loading.value = false } }
function newDivision() { editDivision.value = false; divisionForm.value = true }
function divisionSaved() { divisionForm.value = false; ui.poomsaeDivisionId = store.division!.id }
function eventSaved(e: PoomsaeEvent) { store.event = e; eventForm.value = false }
async function loadHistory() { historyBusy.value = true; error.value = ''; try { history.value = await webApi().call(store.divisionPath() + '/history?limit=20') } catch (e) { error.value = message(e) } finally { historyBusy.value = false } }
function actionLabel(action: string) { if (action.includes('/scores')) return 'ثبت یا اصلاح نمره'; if (action.includes('/reopen')) return 'بازگشایی نتایج'; if (action.includes('/finalize')) return 'نهایی‌کردن نتایج'; if (action.includes('draw')) return 'تغییر قرعه'; if (action.includes('/entries')) return 'تغییر فهرست ورزشکاران'; return 'تنظیمات رده' }
watch(() => ui.poomsaeEventId, load, { immediate: true })
watch(() => ui.poomsaeDivisionId, id => { if (!loading.value && id && store.event?.id === ui.poomsaeEventId && id !== store.division?.id) changeDivision(id) })
</script>
<template>
  <div class="ps space-y-5">
    <section class="ps-hero"><div class="ps-toolbar relative z-1"><div><div class="text-xs text-teal-200 mb-2 font-bold tracking-wide">پومسه / پنل برگزاری</div><h2>{{ store.event?.name || 'مسابقه پومسه' }}</h2><p v-if="store.event" class="text-xs text-slate-300 mt-2">{{ jalaliFromIso(store.event.date) }} · {{ store.event.rules.version }}</p></div><div class="ps-actions"><button class="ps-btn secondary" :disabled="loading || store.busy" @click="refresh">به‌روزرسانی اطلاعات</button><button v-if="store.options?.permissions.manageEvent" class="ps-btn" :disabled="loading || store.busy" @click="newDivision">＋ رده جدید</button></div></div></section>
    <p v-if="error" class="ps-error" role="alert">{{ error }} <button class="ps-link" :disabled="loading" @click="load">تلاش دوباره</button></p>
    <div v-if="loading" class="ps-card ps-muted" role="status">در حال دریافت اطلاعات مسابقه…</div>
    <template v-else-if="store.event">
      <div v-if="!store.divisions.length" class="ps-card ps-empty"><h3 class="ps-title">اولین ردهٔ مسابقه را تعریف کنید</h3><p class="ps-muted my-3">ردهٔ سنی، بازهٔ تولد و فرم‌های مجاز را مشخص کنید؛ سپس ورزشکاران را اضافه کنید.</p><button v-if="store.options?.permissions.manageEvent" class="ps-btn" @click="newDivision">افزودن رده</button></div>
      <div v-else class="ps-card ps-toolbar"><label class="ps-field min-w-64 flex-1 max-w-lg">ردهٔ فعال<select :value="store.division?.id || ''" :disabled="store.busy" @change="changeDivision(($event.target as HTMLSelectElement).value)"><option value="" disabled>انتخاب رده</option><option v-for="d in store.divisions" :key="d.id" :value="d.id">{{ d.name }} · {{ genderLabels[d.gender] }}{{ d.belt ? ` · ${d.belt}` : '' }}</option></select></label><div v-if="store.division" class="text-left"><span class="ps-pill" :class="{ warn: store.division.status !== 'finalized' }">{{ statusLabels[store.division.status] }}</span><p class="ps-muted mt-2">فرم‌های مجاز: {{ store.division.allowedFormCodes.map(fa).join('، ') }}</p></div></div>
      <div v-if="store.dirtyCount" class="ps-note">{{ fa(store.dirtyCount) }} ردیف نمرهٔ ذخیره‌نشده دارید. پیش‌نویس‌ها هنگام جابه‌جایی بین تب‌ها و رده‌ها حفظ می‌شوند.</div>
      <template v-if="store.division">
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4"><div class="ps-stat"><strong>{{ fa(store.division.entries.length) }}</strong><span>ورزشکار ثبت‌شده</span></div><div class="ps-stat"><strong>{{ fa(activeCount) }}</strong><span>ورزشکار حاضر</span></div><div class="ps-stat"><strong class="text-teal-700">{{ fa(store.division.standings?.ranked.length || 0) }}</strong><span>نتیجهٔ کامل</span></div><div class="ps-stat"><strong class="text-amber-600">{{ fa(activeCount - (store.division.standings?.ranked.length || 0)) }}</strong><span>در انتظار نمره</span></div></div>
        <nav class="ps-tabs" aria-label="بخش‌های مسابقه پومسه"><button v-for="tab in tabs" :key="tab.key" :class="{ active: ui.poomsaeTab === tab.key }" :aria-current="ui.poomsaeTab === tab.key ? 'page' : undefined" :disabled="store.busy" @click="ui.poomsaeTab = tab.key">{{ tab.label }}</button></nav>
        <div :key="store.division.id">
          <PoomsaeAthletes v-if="ui.poomsaeTab === 'athletes'" />
          <PoomsaeDraw v-else-if="ui.poomsaeTab === 'draw'" />
          <PoomsaeScores v-else-if="ui.poomsaeTab === 'scores'" />
          <PoomsaeStandings v-else-if="ui.poomsaeTab === 'standings'" />
          <PoomsaePrint v-else-if="ui.poomsaeTab === 'print'" />
          <div v-else class="space-y-5">
            <section class="ps-card space-y-4"><div class="ps-toolbar"><h3 class="ps-title">تنظیمات رده</h3><button v-if="store.options?.permissions.manage" class="ps-btn secondary" :disabled="store.division.status !== 'draft' || store.division.everScored" @click="editDivision = true; divisionForm = true">ویرایش رده</button></div><dl class="ps-grid text-sm"><div><dt class="ps-muted">بازهٔ تولد</dt><dd class="mt-1">{{ jalaliFromIso(store.division.birthDateFrom) || 'بدون حد ابتدایی' }} تا {{ jalaliFromIso(store.division.birthDateTo) || 'بدون حد انتهایی' }}</dd></div><div><dt class="ps-muted">جنسیت / کمربند</dt><dd class="mt-1">{{ genderLabels[store.division.gender] }} · {{ store.division.belt || 'بدون محدودیت کمربند' }}</dd></div><div><dt class="ps-muted">فرم‌های مجاز</dt><dd class="mt-1">{{ store.division.allowedFormCodes.map(fa).join('، ') }}</dd></div><div><dt class="ps-muted">انتخاب فرم</dt><dd class="mt-1">{{ store.event.rules.formSelection === 'division' ? `فرم ${fa(store.division.form1Code)} و ${fa(store.division.form2Code)} برای همه` : 'انتخاب جدا برای هر ورزشکار' }}</dd></div></dl></section>
            <section class="ps-card space-y-4"><div class="ps-toolbar"><h3 class="ps-title">قواعد مسابقه</h3><button v-if="store.options?.permissions.manageEvent" class="ps-btn secondary" @click="eventForm = true">ویرایش مسابقه</button></div><p class="text-sm leading-8">سقف دقت: {{ fa(store.event.rules.accuracyMax) }} · سقف اجرا: {{ fa(store.event.rules.presentationMax) }}<br>نمره کل: جمع دو فرم · تساوی: رتبه مشترک<br>تکرار فرم: {{ store.event.rules.allowRepeatedForm ? 'مجاز' : 'غیرمجاز' }}</p></section>
            <section v-if="store.options?.permissions.audit" class="ps-card space-y-4"><div class="ps-toolbar"><h3 class="ps-title">سابقهٔ تغییرات رده</h3><button class="ps-btn secondary" :disabled="historyBusy" @click="loadHistory">{{ historyBusy ? 'در حال دریافت…' : 'نمایش ۲۰ تغییر اخیر' }}</button></div><ol v-if="history" class="space-y-3"><li v-for="item in history" :key="item.id" class="rounded-xl border border-slate-200 p-4"><div class="ps-toolbar"><strong class="text-sm">{{ actionLabel(item.action) }}</strong><span class="ps-muted">{{ new Date(item.createdAt).toLocaleString('fa-IR') }}</span></div><p class="ps-muted mt-2">{{ item.reason || 'بدون توضیح اضافی' }}</p></li></ol></section>
          </div>
        </div>
      </template>
    </template>
    <PoomsaeDivisionForm v-if="divisionForm" :edit="editDivision" @close="divisionForm = false" @saved="divisionSaved" />
    <PoomsaeEventForm v-if="eventForm && store.event" :event="store.event" :locked="store.divisions.length > 0" @close="eventForm = false" @saved="eventSaved" />
  </div>
</template>
