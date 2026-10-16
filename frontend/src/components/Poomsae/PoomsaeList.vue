<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { usePoomsaeStore } from '../../stores/poomsae'
import { useUiStore } from '../../stores/ui'
import type { PoomsaeEvent } from '../../types/poomsae'
import { fa, jalaliFromIso, message } from '../../utils/poomsae'
import PoomsaeEventForm from './PoomsaeEventForm.vue'
import './poomsae.css'
const store = usePoomsaeStore(), ui = useUiStore()
const events = ref<PoomsaeEvent[]>([]), search = ref(''), error = ref(''), loading = ref(true), creating = ref(false)
const filtered = computed(() => events.value.filter(e => e.name.includes(search.value.trim())))
async function load() { loading.value = true; error.value = ''; try { const [items] = await Promise.all([store.all<PoomsaeEvent>('/poomsae-events'), store.settings()]); events.value = items } catch (e) { error.value = message(e) } finally { loading.value = false } }
function open(id: string) { ui.poomsaeEventId = id; ui.poomsaeDivisionId = ''; ui.poomsaeTab = 'athletes'; ui.page = 'poomsae' }
function saved(e: PoomsaeEvent) { creating.value = false; open(e.id) }
onMounted(load)
</script>
<template>
  <section class="ps space-y-5">
    <div class="ps-toolbar"><div><h3 class="ps-title">مسابقات پومسه</h3><p class="ps-muted mt-1">از ثبت‌نام تا آخرین نمره؛ همهٔ مراحل در یک فضای مشترک</p></div><button v-if="store.options?.permissions.createEvent" class="ps-btn" @click="creating = true">＋ مسابقه پومسه جدید</button></div>
    <div v-if="error" class="ps-error" role="alert">{{ error }} <button class="ps-link" @click="load">تلاش دوباره</button></div>
    <div v-if="loading" class="ps-card ps-muted" role="status">در حال دریافت مسابقات پومسه…</div>
    <template v-else-if="!error">
      <label v-if="events.length" class="ps-field max-w-sm">جست‌وجوی مسابقه<input v-model="search" placeholder="نام مسابقه…"></label>
      <div v-if="!filtered.length" class="ps-empty"><div class="text-4xl mb-4 text-teal-700">◎</div><h4 class="ps-title">{{ events.length ? 'مسابقه‌ای پیدا نشد' : 'اولین مسابقه پومسه را بسازید' }}</h4><p class="ps-muted mt-2">{{ events.length ? 'عبارت جست‌وجو را تغییر دهید.' : 'رده‌ها، فرم‌های مجاز و ورزشکاران را پس از ساخت مسابقه اضافه کنید.' }}</p></div>
      <div class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3"><article v-for="event in filtered" :key="event.id" class="ps-card relative overflow-hidden"><div class="absolute inset-x-0 top-0 h-1 bg-teal-600"></div><div class="ps-toolbar"><span class="ps-pill">پومسه انفرادی</span><span class="ps-muted">{{ jalaliFromIso(event.date) }}</span></div><h4 class="text-xl font-black mt-5 mb-2">{{ event.name }}</h4><p class="ps-muted">{{ event.rules.version }}</p><div class="mt-5 mb-5 rounded-xl bg-slate-50 p-3 text-xs leading-7 text-slate-600">دو فرم · رتبه‌بندی با امتیاز کل<br>سقف دقت {{ fa(event.rules.accuracyMax) }} / اجرا {{ fa(event.rules.presentationMax) }}</div><button class="ps-btn w-full" @click="open(event.id)">ورود به مسابقه ←</button></article></div>
    </template>
    <PoomsaeEventForm v-if="creating" @close="creating = false" @saved="saved" />
  </section>
</template>
