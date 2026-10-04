<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePoomsaeStore } from '../../stores/poomsae'
import { attendanceLabels, fa, message } from '../../utils/poomsae'
const store = usePoomsaeStore(), error = ref(''), reason = ref(''), success = ref('')
const standings = computed(() => store.division!.standings)
const incomplete = computed(() => standings.value.unranked.filter(r => r.entry.status === 'active').length)
async function finish(reopen = false) {
  error.value = ''; success.value = ''
  if (reopen && !reason.value.trim()) { error.value = 'علت بازگشایی را بنویسید.'; return }
  if (!confirm(reopen ? 'نتایج برای اصلاح بازگشایی شوند؟' : 'نتایج این رده نهایی و ثبت نمره قفل شود؟')) return
  try { await store.mutate(reopen ? '/reopen' : '/finalize', 'POST', reopen ? { reason: reason.value.trim() } : undefined); success.value = reopen ? 'رده برای اصلاح بازگشایی شد.' : 'نتایج رده نهایی شد.'; reason.value = '' } catch (e) { error.value = message(e) }
}
</script>
<template>
  <div class="space-y-5">
    <div class="ps-card space-y-4"><div class="ps-toolbar"><div><h3 class="ps-title">رتبه‌بندی رده</h3><p class="ps-muted mt-1">مرتب‌شده بر اساس جمع امتیاز دو فرم</p></div><span class="ps-pill" :class="{ warn: !standings.final }">{{ standings.final ? 'نتایج نهایی' : 'نتایج موقت' }}</span></div>
      <div v-if="standings.ranked.length" class="grid gap-3 sm:grid-cols-3"><div v-for="(row, i) in standings.ranked.slice(0, 3)" :key="row.entry.id" class="rounded-2xl border p-5" :class="i === 0 ? 'border-amber-200 bg-amber-50' : 'border-slate-200 bg-slate-50'"><div class="ps-toolbar"><span class="ps-muted">رتبه {{ fa(row.rank) }}</span><strong class="text-2xl font-black text-teal-800">{{ fa(row.total) }}</strong></div><h4 class="mt-4 font-bold">{{ row.entry.firstName }} {{ row.entry.lastName }}</h4><p class="ps-muted mt-1">{{ row.entry.teamName }}</p></div></div>
      <div v-else class="ps-empty"><h4 class="font-bold">هنوز نتیجهٔ کامل ثبت نشده است</h4><p class="ps-muted mt-2">پس از تکمیل چهار نمره و انتخاب فرم‌ها، ورزشکار رتبه می‌گیرد.</p></div>
      <div v-if="standings.ranked.length" class="ps-table-wrap"><table class="ps-table"><thead><tr><th>رتبه</th><th>ورزشکار</th><th>تیم</th><th>فرم اول</th><th>فرم دوم</th><th>امتیاز کل</th></tr></thead><tbody><tr v-for="row in standings.ranked" :key="row.entry.id"><td class="font-black text-lg">{{ fa(row.rank) }}</td><td class="font-bold">{{ row.entry.firstName }} {{ row.entry.lastName }}</td><td>{{ row.entry.teamName }}</td><td>{{ fa(row.form1Total) }}</td><td>{{ fa(row.form2Total) }}</td><td class="font-black text-teal-800">{{ fa(row.total) }}</td></tr></tbody></table></div>
      <p class="ps-muted">امتیاز مساوی، رتبهٔ مشترک دارد؛ ترتیب قرعه در رفع تساوی تأثیری ندارد.</p>
    </div>
    <div v-if="standings.unranked.length" class="ps-card space-y-4"><h3 class="font-bold">بدون رتبه · {{ fa(standings.unranked.length) }} نفر</h3><div class="ps-table-wrap"><table class="ps-table"><thead><tr><th>ورزشکار</th><th>تیم</th><th>وضعیت</th><th>فرم اول</th><th>فرم دوم</th></tr></thead><tbody><tr v-for="row in standings.unranked" :key="row.entry.id"><td>{{ row.entry.firstName }} {{ row.entry.lastName }}</td><td>{{ row.entry.teamName }}</td><td><span class="ps-pill warn">{{ row.entry.status === 'active' ? 'نمره ناقص' : attendanceLabels[row.entry.status] }}</span></td><td>{{ fa(row.form1Total) }}</td><td>{{ fa(row.form2Total) }}</td></tr></tbody></table></div></div>
    <div class="ps-card space-y-4"><h3 class="font-bold">{{ standings.final ? 'اصلاح نتیجه نهایی' : 'نهایی‌کردن نتایج' }}</h3><p class="ps-muted">{{ standings.final ? 'بازگشایی با ثبت علت و دسترسی مسئول مجاز انجام می‌شود.' : incomplete ? `${fa(incomplete)} ورزشکار حاضر هنوز نتیجهٔ کامل ندارد.` : 'پس از نهایی‌کردن، ثبت و ویرایش نمره قفل می‌شود.' }}</p><template v-if="standings.final && store.options?.permissions.override"><label class="ps-field max-w-lg">علت بازگشایی<textarea v-model="reason" rows="2" placeholder="علت بازبینی نتیجه"></textarea></label><button class="ps-btn secondary" :disabled="store.busy" @click="finish(true)">بازگشایی برای اصلاح</button></template><button v-else-if="!standings.final && store.options?.permissions.manage" class="ps-btn" :disabled="store.busy || incomplete > 0 || store.division!.status === 'draft' || store.dirtyCount > 0" @click="finish()">نهایی‌کردن رده</button><p v-if="store.dirtyCount" class="ps-muted">ابتدا پیش‌نویس نمرات را ذخیره یا کنار بگذارید.</p><p v-if="error" class="ps-error" role="alert">{{ error }}</p><p v-if="success" class="ps-note" role="status">{{ success }}</p></div>
  </div>
</template>
