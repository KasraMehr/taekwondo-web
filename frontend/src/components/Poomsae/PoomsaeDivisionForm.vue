<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ApiError } from '../../api'
import { webApi } from '../../webApi'
import { usePoomsaeStore } from '../../stores/poomsae'
import type { DivisionInput, PoomsaeEvent } from '../../types/poomsae'
import { fa, isoFromJalali, jalaliFromIso, message } from '../../utils/poomsae'
import PoomsaeModal from './PoomsaeModal.vue'
const props = defineProps<{ edit?: boolean }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const store = usePoomsaeStore(), d = props.edit ? store.division : null
const form = reactive({ name: d?.name || '', gender: d?.gender || 'male', belt: d?.belt || '', from: jalaliFromIso(d?.birthDateFrom), to: jalaliFromIso(d?.birthDateTo), codes: [...d?.allowedFormCodes || []], first: d?.form1Code ?? null, second: d?.form2Code ?? null })
const preset = ref(''), error = ref(''), busy = ref(false), conflict = ref(false)
function template() { const p = store.options?.categories.find(p => p.key === preset.value); if (p) { form.name = p.name; form.codes = [...p.allowedFormCodes]; form.first = null; form.second = null } }
function toggle(n: number) { form.codes = form.codes.includes(n) ? form.codes.filter(v => v !== n) : [...form.codes, n].sort((a, b) => a - b); if (form.first && !form.codes.includes(form.first)) form.first = null; if (form.second && !form.codes.includes(form.second)) form.second = null }
async function reload() {
  busy.value = true
  try { store.event = await webApi().call<PoomsaeEvent>(`/poomsae-events/${store.event!.id}`); if (props.edit) await store.refreshDivision(); conflict.value = false; error.value = 'نسخهٔ تازه دریافت شد؛ تنظیمات فعلی را پیش از ذخیره دوباره بررسی کنید.' } catch (e) { error.value = message(e) } finally { busy.value = false }
}
async function submit() {
  busy.value = true; error.value = ''
  try {
    if (!form.name.trim() || !form.codes.length) throw new Error('نام رده و دست‌کم یک فرم مجاز را مشخص کنید.')
    const from = form.from.trim() ? isoFromJalali(form.from) : null, to = form.to.trim() ? isoFromJalali(form.to) : null
    if (!from && !to) throw new Error('حداقل یکی از مرزهای تاریخ تولد را مشخص کنید.')
    if (from && to && from > to) throw new Error('ابتدای بازهٔ تولد باید قبل از انتهای آن باشد.')
    if ((from && from > store.event!.date) || (to && to > store.event!.date)) throw new Error('تاریخ تولد نمی‌تواند بعد از تاریخ مسابقه باشد.')
    const common = store.event!.rules.formSelection === 'division'
    if (common && (!form.first || !form.second)) throw new Error('فرم اول و دوم رده را انتخاب کنید.')
    if (common && !store.event!.rules.allowRepeatedForm && form.first === form.second) throw new Error('فرم اول و دوم باید متفاوت باشند.')
    const input: DivisionInput = { name: form.name.trim(), competitionType: 'individual', gender: form.gender as DivisionInput['gender'], belt: form.belt.trim(), birthDateFrom: from, birthDateTo: to, allowedFormCodes: form.codes, form1Code: common ? form.first : null, form2Code: common ? form.second : null }
    if (props.edit) await store.mutate('', 'PUT', input); else await store.createDivision(input)
    emit('saved')
  } catch (e) { error.value = message(e); conflict.value = e instanceof ApiError && e.status === 409 } finally { busy.value = false }
}
</script>
<template>
  <PoomsaeModal :title="edit ? 'ویرایش رده' : 'افزودن رده مسابقه'" :busy="busy" @close="emit('close')">
    <form class="space-y-5" @submit.prevent="submit">
      <label v-if="!edit" class="ps-field">الگوی رده سنی<select v-model="preset" @change="template"><option value="">انتخاب الگو یا تعریف دستی</option><option v-for="p in store.options?.categories" :key="p.key" :value="p.key">{{ p.name }}</option></select><small>فهرست فرم‌ها از جدول ارسالی آماده شده؛ بازهٔ تولد را مطابق سال مسابقه وارد کنید.</small></label>
      <div class="ps-grid">
        <label class="ps-field ps-full">نام رده<input v-model="form.name" required placeholder="مثلاً زیر ۱۲ سال"></label>
        <label class="ps-field">جنسیت<select v-model="form.gender"><option value="male">آقایان</option><option value="female">بانوان</option><option value="mixed">مختلط</option></select></label>
        <label class="ps-field">کمربند<input v-model="form.belt" placeholder="خالی = بدون محدودیت"></label>
        <label class="ps-field">متولدین از (شمسی)<input v-model="form.from" dir="ltr" placeholder="۱۳۹۳/۰۱/۰۱"><small>این تاریخ هم در بازه محاسبه می‌شود.</small></label>
        <label class="ps-field">متولدین تا (شمسی)<input v-model="form.to" dir="ltr" placeholder="۱۳۹۵/۱۲/۲۹"><small>برای ردهٔ یک‌طرفه یکی از مرزها را خالی بگذارید.</small></label>
      </div>
      <fieldset><legend class="text-sm font-bold mb-3">فرم‌های مجاز این رده</legend><div class="flex flex-wrap gap-2"><button v-for="n in 16" :key="n" type="button" :aria-pressed="form.codes.includes(n)" class="ps-btn small" :class="{ secondary: !form.codes.includes(n) }" @click="toggle(n)">{{ fa(n) }}</button></div></fieldset>
      <div v-if="store.event?.rules.formSelection === 'division'" class="ps-grid"><label class="ps-field">فرم اول<select v-model="form.first" required><option :value="null" disabled>انتخاب فرم</option><option v-for="code in form.codes" :key="code" :value="code">فرم {{ fa(code) }}</option></select></label><label class="ps-field">فرم دوم<select v-model="form.second" required><option :value="null" disabled>انتخاب فرم</option><option v-for="code in form.codes" :key="code" :value="code">فرم {{ fa(code) }}</option></select></label></div>
      <p v-else class="ps-note">فرم اول و دوم هر ورزشکار هنگام ثبت نمره انتخاب می‌شوند.</p>
      <div v-if="error" class="ps-error" role="alert">{{ error }}<button v-if="conflict" type="button" class="ps-link" @click="reload">دریافت نسخه تازه</button></div>
      <div class="ps-actions justify-end"><button type="button" class="ps-btn secondary" :disabled="busy" @click="emit('close')">انصراف</button><button class="ps-btn" :disabled="busy || conflict">{{ busy ? 'در حال ذخیره…' : edit ? 'ذخیره رده' : 'افزودن رده' }}</button></div>
    </form>
  </PoomsaeModal>
</template>
