<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ApiError } from '../../api'
import { webApi } from '../../webApi'
import { usePoomsaeStore } from '../../stores/poomsae'
import type { EntryInput, PoomsaeEntry, Profile } from '../../types/poomsae'
import { isoFromJalali, jalaliFromIso, message } from '../../utils/poomsae'
import PoomsaeModal from './PoomsaeModal.vue'
const props = defineProps<{ entry?: PoomsaeEntry }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const store = usePoomsaeStore()
const profiles = ref<Profile[]>([]), loading = ref(false), query = ref(''), source = ref<'existing' | 'new'>('existing')
const busy = ref(false), error = ref(''), conflict = ref(false), createdProfile = ref('')
const form = reactive({ athleteId: props.entry?.athleteId || '', firstName: props.entry?.firstName || '', lastName: props.entry?.lastName || '', teamName: props.entry?.teamName || '', birthDate: jalaliFromIso(props.entry?.birthDate), gender: props.entry?.gender || (store.division?.gender === 'female' ? 'female' : 'male'), belt: props.entry?.belt || store.division?.belt || '', status: props.entry?.status || 'active', reason: '' })
const matches = computed(() => profiles.value.filter(p => p.isActive && p.name.includes(query.value.trim())))
const eligibilityLocked = computed(() => !!props.entry && store.division?.status !== 'draft')
function select() { const profile = profiles.value.find(p => p.id === form.athleteId); if (!profile) return; form.birthDate = jalaliFromIso(profile.birthDate); if (profile.gender) form.gender = profile.gender; form.firstName = ''; form.lastName = '' }
async function reload() { busy.value = true; try { await store.refreshDivision(); conflict.value = false; error.value = 'اطلاعات تازه دریافت شد. فهرست ورزشکاران و ورودی خود را بررسی و سپس ذخیره کنید.' } catch (e) { error.value = message(e) } finally { busy.value = false } }
async function submit() {
  busy.value = true; error.value = ''
  try {
    const birthDate = isoFromJalali(form.birthDate), d = store.division!
    if ((d.birthDateFrom && birthDate < d.birthDateFrom) || (d.birthDateTo && birthDate > d.birthDateTo)) throw new Error('تاریخ تولد در بازهٔ این رده نیست.')
    if (d.gender !== 'mixed' && form.gender !== d.gender) throw new Error('جنسیت ورزشکار با رده مطابقت ندارد.')
    if (d.belt && form.belt.trim() !== d.belt) throw new Error('کمربند ورزشکار با رده مطابقت ندارد.')
    if (!form.firstName.trim() || !form.lastName.trim() || !form.teamName.trim()) throw new Error('نام، نام خانوادگی و تیم را وارد کنید.')
    if (!props.entry && source.value === 'new' && !createdProfile.value) {
      const profile = await webApi().call<Profile>('/athletes', 'POST', { name: `${form.firstName.trim()} ${form.lastName.trim()}`, birthDate, gender: form.gender })
      createdProfile.value = profile.id; form.athleteId = profile.id
    }
    if (!form.athleteId) throw new Error('یک پروفایل ورزشکار انتخاب کنید.')
    const input: EntryInput = { athleteId: form.athleteId, firstName: form.firstName.trim(), lastName: form.lastName.trim(), teamName: form.teamName.trim(), birthDate, gender: form.gender as EntryInput['gender'], belt: form.belt.trim(), status: form.status as EntryInput['status'], reason: form.reason.trim() }
    await store.mutate(props.entry ? `/entries/${props.entry.id}` : '/entries', props.entry ? 'PUT' : 'POST', input)
    emit('saved')
  } catch (e) { error.value = message(e); conflict.value = e instanceof ApiError && e.status === 409 } finally { busy.value = false }
}
let request = 0, timer: ReturnType<typeof setTimeout> | undefined
async function loadProfiles() { if (props.entry || !store.options?.permissions.athletesRead) return; const ticket = ++request; loading.value = true; try { const list = await webApi().call<Profile[]>(`/athletes?limit=100&search=${encodeURIComponent(query.value.trim())}`); if (ticket === request) profiles.value = list } catch (e) { if (ticket === request) error.value = message(e) } finally { if (ticket === request) loading.value = false } }
watch(query, () => { clearTimeout(timer); request++; form.athleteId = ''; timer = setTimeout(loadProfiles, 250) })
onMounted(loadProfiles)
onUnmounted(() => { request++; clearTimeout(timer) })
</script>
<template>
  <PoomsaeModal :title="entry ? 'ویرایش ورزشکار' : 'افزودن ورزشکار به رده'" :busy="busy" @close="emit('close')">
    <form class="space-y-5" @submit.prevent="submit">
      <template v-if="!entry">
        <div class="ps-actions"><button type="button" class="ps-btn small" :class="{ secondary: source !== 'existing' }" :disabled="!!createdProfile" @click="source = 'existing'; form.athleteId = ''">پروفایل موجود</button><button v-if="store.options?.permissions.athletesManage" type="button" class="ps-btn small" :class="{ secondary: source !== 'new' }" :disabled="!!createdProfile" @click="source = 'new'; form.athleteId = ''">ورزشکار جدید</button></div>
        <div v-if="source === 'existing'" class="space-y-3"><label class="ps-field">جست‌وجوی نام در پروفایل‌ها<input v-model="query" placeholder="نام ورزشکار…"></label><label class="ps-field">پروفایل ورزشکار<select v-model="form.athleteId" required :disabled="loading" @change="select"><option value="">{{ loading ? 'در حال دریافت…' : 'انتخاب ورزشکار' }}</option><option v-for="p in matches" :key="p.id" :value="p.id">{{ p.name }} · {{ jalaliFromIso(p.birthDate) || 'تولد ثبت نشده' }}</option></select><small>پس از انتخاب، نام و نام خانوادگی را در دو فیلد جدا تکمیل کنید.</small></label></div>
        <p v-if="createdProfile" class="ps-note">پروفایل ساخته شده است؛ برای تکمیل ثبت‌نام دوباره ذخیره کنید. پروفایل تکراری ساخته نمی‌شود.</p>
      </template>
      <div class="ps-grid">
        <label class="ps-field">نام<input v-model="form.firstName" required maxlength="50"></label>
        <label class="ps-field">نام خانوادگی<input v-model="form.lastName" required maxlength="50"></label>
        <label class="ps-field ps-full">نام تیم یا باشگاه<input v-model="form.teamName" required maxlength="100"></label>
        <label class="ps-field">تاریخ تولد (شمسی)<input v-model="form.birthDate" required dir="ltr" :disabled="eligibilityLocked || !!createdProfile" placeholder="۱۳۹۵/۰۱/۱۵"></label>
        <label class="ps-field">جنسیت<select v-model="form.gender" :disabled="eligibilityLocked || !!createdProfile"><option value="male">آقایان</option><option value="female">بانوان</option></select></label>
        <label class="ps-field">کمربند<input v-model="form.belt" :disabled="eligibilityLocked" placeholder="در صورت نیاز"></label>
        <label class="ps-field">وضعیت حضور<select v-model="form.status"><option value="active">حاضر / فعال</option><option value="absent">غایب</option><option value="withdrawn">انصراف</option></select></label>
        <label v-if="entry" class="ps-field ps-full">علت اصلاح<textarea v-model="form.reason" rows="2" :required="store.division?.everScored || (eligibilityLocked && form.status !== entry.status)" placeholder="برای تغییر حضور بعد از قرعه یا اصلاح پس از امتیازدهی لازم است"></textarea></label>
      </div>
      <div v-if="error" class="ps-error" role="alert">{{ error }}<button v-if="conflict" type="button" class="ps-link" @click="reload">دریافت نسخه تازه</button></div>
      <div class="ps-actions justify-end"><button type="button" class="ps-btn secondary" :disabled="busy" @click="emit('close')">انصراف</button><button class="ps-btn" :disabled="busy || conflict">{{ busy ? 'در حال ذخیره…' : 'ذخیره ورزشکار' }}</button></div>
    </form>
  </PoomsaeModal>
</template>
