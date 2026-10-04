<script setup lang="ts">
import { computed, ref } from 'vue'
import { ApiError } from '../../api'
import { usePoomsaeStore } from '../../stores/poomsae'
import type { PoomsaeEntry, Scores } from '../../types/poomsae'
import { attendanceLabels, copyScores, fa, message, normalizeScore, scoreKey, sum } from '../../utils/poomsae'
const store = usePoomsaeStore(), notice = ref('')
const fields = ['form1', 'form2'] as const
const rows = computed(() => store.division!.results)
const open = computed(() => ['drawn', 'scoring'].includes(store.division!.status) && store.options?.permissions.score)
function current(e: PoomsaeEntry): Scores { const s = copyScores(e.scores); if (store.event!.rules.formSelection === 'division') { s.form1.code = store.division!.form1Code; s.form2.code = store.division!.form2Code }; return s }
function dirty(id: string) { const d = store.draftFor(id); return scoreKey(d.value) !== scoreKey(d.base) }
function fieldError(value: string | null, max: string) { try { normalizeScore(value, max); return '' } catch (e) { return message(e) } }
function useServer(id: string) { store.clearDraft(id); notice.value = 'نمرهٔ ذخیره‌شدهٔ سرور در ردیف قرار گرفت.' }
function keepDraft(id: string) { const draft = store.draftFor(id); const e = store.division!.entries.find(e => e.id === id)!; draft.base = current(e); draft.conflict = false; draft.error = ''; notice.value = 'پیش‌نویس حفظ شد. برای ثبت تغییر روی نسخهٔ تازه، دوباره ذخیره کنید.' }
async function save(id: string) {
  const divisionId = store.division!.id, draft = store.draftFor(id), e = store.division!.entries.find(e => e.id === id)!
  draft.error = ''; notice.value = ''
  if (scoreKey(current(e)) !== scoreKey(draft.base)) { draft.conflict = true; draft.error = 'نمرهٔ ذخیره‌شده تغییر کرده است. مقادیر سرور را با پیش‌نویس مقایسه کنید.'; return }
  try {
    const input = copyScores(draft.value); input.reason = draft.value.reason?.trim() || ''
    for (const key of fields) { input[key].accuracy = normalizeScore(input[key].accuracy, store.event!.rules.accuracyMax); input[key].presentation = normalizeScore(input[key].presentation, store.event!.rules.presentationMax) }
    if ((input.form1.accuracy !== null || input.form1.presentation !== null || input.form2.accuracy !== null || input.form2.presentation !== null) && (!input.form1.code || !input.form2.code)) throw new Error('فرم اول و دوم را انتخاب کنید.')
    if (input.form1.code && input.form1.code === input.form2.code && !store.event!.rules.allowRepeatedForm) throw new Error('دو فرم باید متفاوت باشند.')
    await store.mutate(`/entries/${id}/scores`, 'PUT', input)
    store.clearDraft(id, divisionId); notice.value = `نمرات ${e.firstName} ${e.lastName} ذخیره شد.`
  } catch (error) {
    draft.error = message(error)
    if (error instanceof ApiError && error.status === 409) { draft.conflict = true; try { await store.refreshDivision() } catch (refreshError) { draft.error += ' ' + message(refreshError) } }
  }
}
</script>
<template>
  <div class="ps-card space-y-5">
    <div class="ps-toolbar"><div><h3 class="ps-title">میز ثبت نمرات</h3><p class="ps-muted mt-1">دقت + اجرا = نمرهٔ فرم · امتیاز کل = جمع دو فرم</p></div><div class="ps-actions"><span class="ps-pill">دقت تا {{ fa(store.event!.rules.accuracyMax) }}</span><span class="ps-pill">اجرا تا {{ fa(store.event!.rules.presentationMax) }}</span></div></div>
    <div v-if="store.division!.status === 'draft'" class="ps-empty"><h4 class="font-bold">ابتدا ترتیب اجرا را قرعه‌کشی کنید</h4><p class="ps-muted mt-2">پس از قرعه، جدول ثبت نمرات فعال می‌شود.</p></div>
    <template v-else>
      <p v-if="store.division!.status === 'finalized'" class="ps-note">نتایج این رده نهایی شده‌اند. برای اصلاح، مسئول مجاز باید از بخش رتبه‌بندی آن را بازگشایی کند.</p>
      <p v-else class="ps-muted">ورودی فارسی و انگلیسی پذیرفته می‌شود. هر ردیف جدا ذخیره می‌شود؛ خانهٔ خالی با صفر متفاوت است.</p>
      <p v-if="notice" role="status" class="ps-note">{{ notice }}</p>
      <div class="ps-table-wrap"><table class="ps-table ps-score-table"><thead><tr><th rowspan="2">ترتیب</th><th rowspan="2">ورزشکار / تیم</th><th colspan="3" class="text-center! border-x border-slate-200">فرم اول <span v-if="store.event!.rules.formSelection === 'division'">· {{ fa(store.division!.form1Code) }}</span></th><th colspan="3" class="text-center! border-x border-slate-200">فرم دوم <span v-if="store.event!.rules.formSelection === 'division'">· {{ fa(store.division!.form2Code) }}</span></th><th rowspan="2">امتیاز کل</th><th rowspan="2">ثبت نتیجه</th></tr><tr><template v-for="key in fields" :key="key"><th>دقت</th><th>اجرا</th><th>نهایی فرم</th></template></tr></thead>
        <tbody><template v-for="row in rows" :key="row.entry.id"><tr :class="{ 'bg-amber-50/40!': dirty(row.entry.id) }"><td class="font-black text-lg">{{ fa(row.position) }}</td><td class="min-w-44"><strong>{{ row.entry.firstName }} {{ row.entry.lastName }}</strong><p class="ps-muted">{{ row.entry.teamName }}</p><span v-if="row.entry.status !== 'active'" class="ps-pill warn">{{ attendanceLabels[row.entry.status] }}</span><div v-if="store.event!.rules.formSelection === 'entry'" class="flex gap-2 mt-2"><label v-for="(key, index) in fields" :key="key" class="text-[10px] text-slate-500">فرم {{ fa(index + 1) }}<select v-model="store.draftFor(row.entry.id).value[key].code" class="ps-code-select" :disabled="!open || row.entry.status !== 'active' || store.busy" :aria-label="`فرم ${index + 1} ${row.entry.firstName} ${row.entry.lastName}`"><option :value="null">—</option><option v-for="code in store.division!.allowedFormCodes" :key="code" :value="code">{{ fa(code) }}</option></select></label></div></td>
          <template v-for="(key, index) in fields" :key="key"><td v-for="component in (['accuracy', 'presentation'] as const)" :key="component"><input v-model="store.draftFor(row.entry.id).value[key][component]" class="ps-score-input" inputmode="decimal" dir="ltr" placeholder="—" :disabled="!open || row.entry.status !== 'active' || store.busy" :aria-label="`${component === 'accuracy' ? 'دقت' : 'اجرا'} فرم ${index + 1} ${row.entry.firstName} ${row.entry.lastName}`" :aria-invalid="!!fieldError(store.draftFor(row.entry.id).value[key][component], component === 'accuracy' ? store.event!.rules.accuracyMax : store.event!.rules.presentationMax)" :title="fieldError(store.draftFor(row.entry.id).value[key][component], component === 'accuracy' ? store.event!.rules.accuracyMax : store.event!.rules.presentationMax)"></td><td class="text-center! font-bold text-slate-600">{{ fa(sum(store.draftFor(row.entry.id).value[key].accuracy, store.draftFor(row.entry.id).value[key].presentation)) }}</td></template>
          <td class="text-center! font-black text-lg text-teal-800 bg-teal-50/40">{{ fa(sum(store.draftFor(row.entry.id).value.form1.accuracy, store.draftFor(row.entry.id).value.form1.presentation, store.draftFor(row.entry.id).value.form2.accuracy, store.draftFor(row.entry.id).value.form2.presentation)) }}</td><td class="min-w-40"><button v-if="open && row.entry.status === 'active'" class="ps-btn small w-full" :disabled="store.busy || !dirty(row.entry.id) || store.draftFor(row.entry.id).conflict" @click="save(row.entry.id)">ذخیره ردیف</button><p class="text-[10px] mt-2" :class="dirty(row.entry.id) ? 'text-amber-700' : 'text-slate-400'">{{ dirty(row.entry.id) ? 'پیش‌نویس ذخیره‌نشده' : row.total !== null ? 'ذخیره‌شده' : 'نتیجه ناقص' }}</p></td></tr>
          <tr v-if="dirty(row.entry.id) || store.draftFor(row.entry.id).error || store.draftFor(row.entry.id).conflict"><td colspan="10" class="bg-slate-50"><div class="space-y-3"><label v-if="dirty(row.entry.id)" class="ps-field max-w-xl">علت اصلاح نمره<input v-model="store.draftFor(row.entry.id).value.reason" placeholder="اگر نمرهٔ قبلی را تغییر می‌دهید، علت را بنویسید" :disabled="store.busy"></label><p v-if="store.draftFor(row.entry.id).error" class="ps-error" role="alert">{{ store.draftFor(row.entry.id).error }}</p><div v-if="store.draftFor(row.entry.id).conflict" class="ps-note"><p>نمرهٔ فعلی سرور — فرم اول: {{ fa(row.entry.scores.form1.accuracy) }} + {{ fa(row.entry.scores.form1.presentation) }}؛ فرم دوم: {{ fa(row.entry.scores.form2.accuracy) }} + {{ fa(row.entry.scores.form2.presentation) }}</p><div class="ps-actions mt-2"><button class="ps-btn secondary small" :disabled="store.busy" @click="useServer(row.entry.id)">استفاده از نمرات سرور</button><button class="ps-btn small" :disabled="store.busy" @click="keepDraft(row.entry.id)">بررسی کردم؛ پیش‌نویس را نگه دار</button></div></div><button v-else class="ps-link" :disabled="store.busy" @click="useServer(row.entry.id)">کنارگذاشتن پیش‌نویس این ردیف</button></div></td></tr>
        </template></tbody></table></div>
    </template>
  </div>
</template>
<style scoped>
.ps-score-table { min-width:1040px; }
.ps-score-table td { padding:12px 7px; }
.ps-score-input { width:76px; padding:11px 5px; border:1px solid #b8cccf; border-radius:9px; text-align:center; font-size:15px; background:white; font-variant-numeric:tabular-nums; }
.ps-score-input[aria-invalid=true] { border:2px solid #e11d48; background:#fff1f2; }
.ps-code-select { display:block; width:65px; border:1px solid #cbd5e1; border-radius:6px; padding:4px; background:white; }
</style>
