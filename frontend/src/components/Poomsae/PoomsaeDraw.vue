<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePoomsaeStore } from '../../stores/poomsae'
import { fa, message, attendanceLabels } from '../../utils/poomsae'
const store = usePoomsaeStore(), reason = ref(''), error = ref(''), done = ref('')
const draw = computed(() => store.division!.draws.find(d => d.active))
const rows = computed(() => draw.value?.entryIds.map(id => store.division!.entries.find(e => e.id === id)).filter(e => !!e) || [])
const locked = computed(() => store.division!.everScored || store.division!.status === 'finalized')
async function action(reset = false) {
  error.value = ''; done.value = ''
  if ((reset || store.division!.draws.length) && !reason.value.trim()) { error.value = 'علت تغییر قرعه را بنویسید.'; return }
  if ((reset || draw.value) && !confirm(reset ? 'قرعه باطل شود تا امکان تغییر فهرست فراهم شود؟' : 'ترتیب اجرای جدید جایگزین قرعهٔ فعلی شود؟')) return
  try { await store.mutate(reset ? '/invalidate-draw' : '/draw', 'POST', { reason: reason.value.trim() }); reason.value = ''; done.value = reset ? 'قرعه باطل شد. می‌توانید فهرست را ویرایش کنید.' : 'قرعه ذخیره شد. ترتیب زیر، ترتیب اجرای ورزشکاران است.' } catch (e) { error.value = message(e) }
}
</script>
<template>
  <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
    <section class="ps-card space-y-5"><div class="ps-toolbar"><div><h3 class="ps-title">ترتیب اجرای ورزشکاران</h3><p class="ps-muted mt-1">{{ draw ? `قرعه شماره ${fa(draw.version)}` : 'قرعه‌کشی برای این رده انجام نشده است' }}</p></div><span class="ps-pill">{{ fa(rows.length) }} اجرا</span></div>
      <div v-if="!draw" class="ps-empty"><div class="text-5xl text-teal-600 mb-4">↝</div><h4 class="font-bold">نوبت اجرای هر ورزشکار را مشخص کنید</h4><p class="ps-muted mt-2">قرعه فقط میان افراد حاضر همین رده انجام می‌شود.</p></div>
      <ol v-else class="space-y-3"><li v-for="(entry, i) in rows" :key="entry!.id" class="flex items-center gap-4 rounded-xl border border-slate-200 p-4"><span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-teal-50 text-xl font-black text-teal-800">{{ fa(i + 1) }}</span><div class="flex-1"><h4 class="font-bold text-sm">{{ entry!.firstName }} {{ entry!.lastName }}</h4><p class="ps-muted">{{ entry!.teamName }}</p></div><span v-if="entry!.status !== 'active'" class="ps-pill warn">{{ attendanceLabels[entry!.status] }}</span></li></ol>
    </section>
    <aside class="space-y-4"><div class="ps-card space-y-4"><h3 class="font-bold">مدیریت قرعه</h3><p class="ps-muted">{{ fa(store.division!.entries.filter(e => e.status === 'active').length) }} ورزشکار حاضر در فهرست</p><p v-if="locked" class="ps-note">پس از شروع نمره‌گذاری، ترتیب اجرا قفل می‌شود.</p><template v-else-if="store.options?.permissions.draw"><label v-if="store.division!.draws.length" class="ps-field">علت تغییر قرعه<textarea v-model="reason" rows="3" placeholder="علت قرعه مجدد یا ابطال"></textarea></label><button class="ps-btn w-full" :disabled="store.busy || !store.division!.entries.some(e => e.status === 'active')" @click="action()">{{ store.busy ? 'در حال ذخیره…' : draw ? 'قرعه‌کشی مجدد' : 'انجام قرعه‌کشی' }}</button><button v-if="draw" class="ps-btn secondary w-full" :disabled="store.busy" @click="action(true)">ابطال قرعه برای اصلاح فهرست</button></template><p v-else class="ps-muted">دسترسی این حساب فقط برای مشاهدهٔ قرعه است.</p></div><div v-if="error" class="ps-error" role="alert">{{ error }}</div><p v-if="done" class="ps-note" role="status">{{ done }}</p><div class="ps-card"><h4 class="font-bold text-sm mb-2">سابقهٔ قرعه</h4><p v-if="!store.division!.draws.length" class="ps-muted">هنوز قرعه‌ای ثبت نشده است.</p><div v-for="item in [...store.division!.draws].reverse()" :key="item.id" class="border-b border-slate-100 py-3 last:border-0"><div class="ps-toolbar"><span class="text-xs">نسخه {{ fa(item.version) }}</span><span class="ps-pill" :class="{ warn: !item.active }">{{ item.active ? 'فعال' : 'غیرفعال' }}</span></div><p class="ps-muted mt-2">{{ new Date(item.createdAt).toLocaleString('fa-IR') }}</p><p v-if="item.reason" class="ps-muted">{{ item.reason }}</p></div></div></aside>
  </div>
</template>
