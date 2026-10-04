<script setup lang="ts">
import { reactive, ref } from 'vue'
import { webApi } from '../../webApi'
import { ApiError } from '../../api'
import type { EventInput, PoomsaeEvent } from '../../types/poomsae'
import { isoFromJalali, jalaliFromIso, todayJalali, normalizeScore, message } from '../../utils/poomsae'
import PoomsaeModal from './PoomsaeModal.vue'
const props = defineProps<{ event?: PoomsaeEvent; locked?: boolean }>()
const emit = defineEmits<{ close: []; saved: [event: PoomsaeEvent] }>()
const form = reactive({ name: props.event?.name || '', date: props.event ? jalaliFromIso(props.event.date) : todayJalali(), accuracy: props.event?.rules.accuracyMax || '', presentation: props.event?.rules.presentationMax || '', selection: props.event?.rules.formSelection || '', repeat: props.event ? String(props.event.rules.allowRepeatedForm) : '', version: props.event?.rules.version || '' })
const error = ref(''), busy = ref(false), conflict = ref(false)
const revision = ref(props.event?.revision)
async function reload() {
  busy.value = true
  try { const e = await webApi().call<PoomsaeEvent>(`/poomsae-events/${props.event!.id}`); revision.value = e.revision; error.value = `نسخهٔ تازه دریافت شد. نام فعلی سرور: «${e.name}». فرم خود را بررسی و سپس ذخیره کنید.`; conflict.value = false } catch (e) { error.value = message(e) } finally { busy.value = false }
}
async function submit() {
  error.value = ''; busy.value = true
  try {
    const accuracyMax = normalizeScore(form.accuracy, '9999.99'), presentationMax = normalizeScore(form.presentation, '9999.99')
    if (!accuracyMax || accuracyMax === '0.00' || !presentationMax || presentationMax === '0.00') throw new Error('سقف دقت و اجرا باید بزرگ‌تر از صفر باشد.')
    if (!form.selection || !form.repeat || !form.version.trim() || !form.name.trim()) throw new Error('همهٔ تنظیمات مسابقه را کامل کنید.')
    const input: EventInput = { name: form.name.trim(), date: isoFromJalali(form.date), rules: { accuracyMax, presentationMax, precision: 2, tiePolicy: 'shared', formSelection: form.selection as 'division' | 'entry', allowRepeatedForm: form.repeat === 'true', version: form.version.trim() } }
    const e = await webApi().call<PoomsaeEvent>(props.event ? `/poomsae-events/${props.event.id}` : '/poomsae-events', props.event ? 'PUT' : 'POST', input, props.event ? { revision: revision.value } : undefined)
    emit('saved', e)
  } catch (e) { error.value = message(e); conflict.value = e instanceof ApiError && e.status === 409 } finally { busy.value = false }
}
</script>
<template>
  <PoomsaeModal :title="event ? 'ویرایش مسابقه پومسه' : 'مسابقه پومسه جدید'" :busy="busy" @close="emit('close')">
    <form class="space-y-5" @submit.prevent="submit">
      <p class="ps-muted">پومسه انفرادی · دو فرم · جمع نمرات دقت و اجرا</p>
      <div class="ps-grid">
        <label class="ps-field ps-full">نام مسابقه<input v-model="form.name" required maxlength="100" placeholder="مثلاً جام پومسه استان" autofocus></label>
        <label class="ps-field">تاریخ مسابقه (شمسی)<input v-model="form.date" :disabled="locked" required dir="ltr" placeholder="۱۴۰۵/۰۷/۱۱"></label>
        <label class="ps-field">عنوان آیین‌نامه<input v-model="form.version" :disabled="locked" required placeholder="مثلاً آیین‌نامه فصل ۱۴۰۵"></label>
        <label class="ps-field">سقف نمره دقت هر فرم<input v-model="form.accuracy" :disabled="locked" required inputmode="decimal" placeholder="طبق آیین‌نامه"></label>
        <label class="ps-field">سقف نمره اجرای هر فرم<input v-model="form.presentation" :disabled="locked" required inputmode="decimal" placeholder="طبق آیین‌نامه"></label>
        <label class="ps-field">انتخاب دو فرم<select v-model="form.selection" :disabled="locked" required><option disabled value="">انتخاب کنید</option><option value="division">دو فرم مشترک برای هر رده</option><option value="entry">انتخاب جدا برای هر ورزشکار</option></select></label>
        <label class="ps-field">تکرار یک فرم<select v-model="form.repeat" :disabled="locked" required><option disabled value="">انتخاب کنید</option><option value="false">مجاز نیست</option><option value="true">مجاز است</option></select></label>
      </div>
      <p class="ps-note">نمره‌ها با دو رقم اعشار ثبت می‌شوند. امتیاز کل برابر جمع دو فرم است؛ افراد با امتیاز مساوی رتبهٔ مشترک می‌گیرند.</p>
      <p v-if="locked" class="ps-muted">پس از ساخت رده، تاریخ و قواعد امتیازدهی قفل هستند.</p>
      <div v-if="error" class="ps-error" role="alert">{{ error }}<button v-if="conflict" type="button" class="ps-btn secondary small mt-2" @click="reload">دریافت نسخه تازه برای بررسی</button></div>
      <div class="ps-actions justify-end"><button type="button" class="ps-btn secondary" :disabled="busy" @click="emit('close')">انصراف</button><button class="ps-btn" :disabled="busy || conflict">{{ busy ? 'در حال ذخیره…' : event ? 'ذخیره تغییرات' : 'ساخت مسابقه' }}</button></div>
    </form>
  </PoomsaeModal>
</template>
