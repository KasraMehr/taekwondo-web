<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { weightOptions, type AgeCategory, type Gender } from '../../data/categories'
import type { AthleteDraft, LeagueAthlete } from '../../types'

const props = defineProps<{
  open: boolean
  /** جنسیت لیگ؛ روی ورزشکار همین ثبت می‌شود */
  gender: Gender
  /** رده سنی لیگ؛ لیست وزن‌ها از همین ساخته می‌شود */
  ageCategory: AgeCategory
  athlete?: LeagueAthlete | null
  clubs?: Array<{ id: string; name: string }>
  error?: string
}>()

const emit = defineEmits<{
  submit: [draft: AthleteDraft]
  close: []
}>()

const form = reactive({
  name: '',
  weightCategory: null as string | null,
  clubId: null as string | null,
})

const localError = ref('')
const nameInput = ref<HTMLInputElement | null>(null)

/** رده‌های وزنی رسمی همین لیگ (جنسیت + رده سنی) */
const weights = computed<Array<{ value: string; label: string }>>(() => {
  const raw = weightOptions(props.ageCategory, props.gender) as unknown[]
  return (raw ?? []).map((item) => {
    if (typeof item === 'string' || typeof item === 'number') {
      return { value: String(item), label: String(item) }
    }
    const o = item as Record<string, unknown>
    const value = String(o.value ?? o.key ?? o.id ?? o.title ?? o.label ?? '')
    return { value, label: String(o.label ?? o.title ?? value) }
  }).filter((o) => o.value !== '')
})

const genderLabel = computed(() => (props.gender === 'female' ? 'زن' : 'مرد'))

watch(
    () => [props.open, props.athlete, props.gender, props.ageCategory] as const,
    ([open, athlete]) => {
      if (!open) return
      form.name = athlete?.name ?? ''
      form.clubId = athlete?.clubId ?? null
      // وزن قبلی فقط اگر در لیست مجاز این لیگ باشد نگه داشته می‌شود
      const prev = athlete?.weightCategory ?? null
      form.clubId = athlete?.clubId ?? null
      form.weightCategory =
          prev && weights.value.some((w) => w.value === prev) ? prev : null
      localError.value = ''
      void nextTick(() => nameInput.value?.focus())
    },
    { immediate: true }
)

function submit() {
  const name = form.name.replace(/\s+/g, ' ').trim()
  if (!name) {
    localError.value = 'نام ورزشکار الزامی است.'
    return
  }
  if (!form.weightCategory) {
    localError.value = 'انتخاب رده وزنی الزامی است.'
    return
  }
  localError.value = ''
  emit('submit', {
    name,
    gender: props.gender,
    weightCategory: form.weightCategory,
    clubId: form.clubId || null,
  })
}
</script>

<template>
  <Teleport to="body">
    <div
        v-if="open"
        dir="rtl"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
        @click.self="emit('close')"
        @keydown.esc="emit('close')"
    >
      <div
          role="dialog"
          aria-modal="true"
          :aria-label="athlete ? 'ویرایش ورزشکار' : 'افزودن ورزشکار'"
          class="w-full max-w-md rounded-xl bg-white p-5 shadow-xl"
      >
        <div class="mb-4 flex items-center justify-between gap-2">
          <h3 class="font-semibold text-slate-800">
            {{ athlete ? 'ویرایش ورزشکار' : 'افزودن ورزشکار' }}
          </h3>
          <span class="rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-600">
            {{ genderLabel }} — {{ ageCategory }}
          </span>
        </div>

        <form class="space-y-3" @submit.prevent="submit">
          <label class="block text-sm">
            <span class="mb-1 block text-slate-600">نام و نام خانوادگی</span>
            <input
                ref="nameInput"
                v-model="form.name"
                required
                autocomplete="off"
                placeholder="مثال: علی رضایی"
                class="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
            />
          </label>

          <label class="block text-sm">
            <span class="mb-1 block text-slate-600">رده وزنی</span>
            <select
                v-model="form.weightCategory"
                required
                :disabled="weights.length === 0"
                class="w-full rounded-lg border border-slate-300 px-2 py-2 text-sm disabled:bg-slate-100"
            >
              <option :value="null" disabled>انتخاب کنید…</option>
              <option v-for="w in weights" :key="w.value" :value="w.value">
                {{ w.label }}
              </option>
            </select>
            <span v-if="weights.length === 0" class="mt-1 block text-xs text-amber-600">
              برای این جنسیت و رده سنی، رده وزنی تعریف‌شده‌ای یافت نشد.
            </span>
          </label>

          <label class="block text-sm">
            <span class="mb-1 block text-slate-600">باشگاه</span>
            <select
                v-model="form.clubId"
                class="w-full rounded-lg border border-slate-300 px-2 py-2 text-sm"
            >
              <option :value="null">آزاد (بدون باشگاه)</option>
              <option v-for="c in clubs" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </label>

          <p v-if="localError || error" role="alert"
             class="rounded-lg bg-rose-50 px-3 py-2 text-xs text-rose-700">
            {{ localError || error }}
          </p>

          <div class="flex gap-2 pt-1">
            <button type="submit"
                    class="flex-1 rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-700">
              {{ athlete ? 'ذخیره' : 'افزودن' }}
            </button>
            <button type="button"
                    class="rounded-lg bg-slate-100 px-3 py-2 text-sm hover:bg-slate-200"
                    @click="emit('close')">
              انصراف
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>
