<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePoomsaeStore } from '../../stores/poomsae'
import type { PoomsaeEntry } from '../../types/poomsae'
import { attendanceLabels, fa, jalaliFromIso, message } from '../../utils/poomsae'
import PoomsaeAthleteForm from './PoomsaeAthleteForm.vue'
const store = usePoomsaeStore(), search = ref(''), team = ref(''), error = ref(''), showing = ref(false), editing = ref<PoomsaeEntry>()
const teams = computed(() => [...new Set(store.division!.entries.map(e => e.teamName))].sort())
const entries = computed(() => store.division!.entries.filter(e => `${e.firstName} ${e.lastName} ${e.teamName}`.includes(search.value.trim()) && (!team.value || e.teamName === team.value)))
function edit(e?: PoomsaeEntry) { editing.value = e; showing.value = true }
async function remove(e: PoomsaeEntry) { if (!confirm(`ثبت‌نام «${e.firstName} ${e.lastName}» از این رده حذف شود؟`)) return; error.value = ''; try { await store.mutate(`/entries/${e.id}`, 'DELETE'); store.clearDraft(e.id) } catch (err) { error.value = message(err) } }
</script>
<template>
  <div class="ps-card space-y-5">
    <div class="ps-toolbar"><div><h3 class="ps-title">ورزشکاران رده</h3><p class="ps-muted mt-1">{{ fa(store.division!.entries.length) }} ورزشکار ثبت‌شده</p></div><button v-if="store.options?.permissions.manage" class="ps-btn" :disabled="store.division!.status !== 'draft' || store.busy" @click="edit()">＋ افزودن ورزشکار</button></div>
    <p v-if="store.division!.status !== 'draft'" class="ps-note">فهرست پس از قرعه قفل است. قبل از ثبت نمره می‌توانید از بخش قرعه‌کشی، قرعه را باطل و فهرست را اصلاح کنید.</p>
    <div class="ps-grid"><label class="ps-field">جست‌وجو<input v-model="search" placeholder="نام، نام خانوادگی یا تیم"></label><label class="ps-field">تیم<select v-model="team"><option value="">همه تیم‌ها</option><option v-for="t in teams" :key="t">{{ t }}</option></select></label></div>
    <div v-if="error" class="ps-error" role="alert">{{ error }}</div>
    <div v-if="!entries.length" class="ps-empty"><h4 class="font-bold">{{ store.division!.entries.length ? 'ورزشکاری با این فیلتر پیدا نشد' : 'فهرست ورزشکاران خالی است' }}</h4><p class="ps-muted mt-2">ورزشکاران هر رده به صورت مستقل قرعه‌کشی و رتبه‌بندی می‌شوند.</p></div>
    <div v-else class="ps-table-wrap"><table class="ps-table"><thead><tr><th>نام و نام خانوادگی</th><th>تیم</th><th>تاریخ تولد</th><th>کمربند</th><th>حضور</th><th v-if="store.options?.permissions.manage">عملیات</th></tr></thead><tbody><tr v-for="e in entries" :key="e.id"><td class="font-bold">{{ e.firstName }} {{ e.lastName }}</td><td>{{ e.teamName }}</td><td class="whitespace-nowrap">{{ jalaliFromIso(e.birthDate) }}</td><td>{{ e.belt || '—' }}</td><td><span class="ps-pill" :class="{ warn: e.status !== 'active' }">{{ attendanceLabels[e.status] }}</span></td><td v-if="store.options?.permissions.manage" class="whitespace-nowrap"><button class="ps-link" :disabled="store.division!.status === 'finalized' || store.busy" @click="edit(e)">ویرایش</button><button v-if="store.division!.status === 'draft'" class="ps-link text-rose-700!" :disabled="store.busy" @click="remove(e)">حذف</button></td></tr></tbody></table></div>
    <PoomsaeAthleteForm v-if="showing" :entry="editing" @close="showing = false" @saved="showing = false" />
  </div>
</template>
