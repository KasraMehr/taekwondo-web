<template>
  <div class="space-y-6">

    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <!-- اکشن‌های اصلی -->
      <button @click="openAdd"
              class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">
        + افزودن ورزشکار
      </button>

      <label class="cursor-pointer rounded-lg border border-blue-200 bg-blue-50 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-100">
        📥 ایمپورت اکسل ورزشکاران
        <input type="file" accept=".xlsx,.xls" class="hidden" @change="importExcel" />
      </label>

      <!-- اکشن‌های کم‌کاربرد -->
      <div class="relative">
        <button @click="showTools = !showTools"
                class="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-600 hover:bg-slate-50">
          ⋯ سایر ابزارها
        </button>

        <div v-if="showTools" class="fixed inset-0 z-40" @click="showTools = false" />
        <div v-if="showTools"
             class="absolute right-0 z-50 mt-1 w-64 overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg">
          <button @click="runTool(exportTemplate)"
                  class="flex w-full items-center gap-2 px-4 py-2.5 text-right text-sm text-slate-700 hover:bg-slate-50">
            📤 دانلود قالب خالی اکسل
          </button>
          <button @click="runTool(exportData)"
                  class="flex w-full items-center gap-2 px-4 py-2.5 text-right text-sm text-slate-700 hover:bg-slate-50">
            ⬇️ خروجی اکسل لیست فعلی
          </button>
          <div class="my-1 border-t border-slate-100" />
          <button @click="runTool(openReset)"
                  class="flex w-full items-center gap-2 px-4 py-2.5 text-right text-sm text-red-600 hover:bg-red-50">
            ♻️ ریست رنکینگ‌ها
          </button>
        </div>
      </div>

      <span class="mr-auto self-center text-sm text-slate-500">{{ athletes.length }} ورزشکار</span>
    </div>

    <!-- فیلتر و سرچ -->
    <div class="flex flex-wrap gap-3 rounded-xl bg-slate-50 p-4">
      <input v-model="searchName" placeholder="جستجو نام ورزشکار..." class="input flex-1 min-w-[160px]" />
      <input v-model="searchClub" placeholder="جستجو باشگاه..." class="input flex-1 min-w-[160px]" />
      <select v-model="filterWeight" class="input min-w-[180px]">
        <option value="">همه وزن‌ها</option>
        <option v-for="w in weights" :key="w" :value="w">
          وزن {{ toFa(weightIndex[w]) }} — {{ w }} kg
        </option>
      </select>
      <button v-if="searchName || searchClub || filterWeight" @click="clearFilters"
              class="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-600 hover:bg-slate-100">
        ✕ پاک کردن فیلتر
      </button>
    </div>

    <!-- گزارش تعداد هر وزن -->
    <div v-if="weightStats.length" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
      <div v-for="stat in weightStats" :key="stat.weight"
           @click="filterWeight = filterWeight === stat.weight ? '' : stat.weight"
           class="relative cursor-pointer rounded-xl border p-3 text-center transition-all"
           :class="filterWeight === stat.weight
             ? 'border-blue-500 bg-blue-50 shadow-sm'
             : 'border-slate-200 bg-white hover:border-blue-300 hover:bg-blue-50/50'">
        <span class="absolute right-2 top-2 rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-500">
          وزن {{ toFa(stat.index) }}
        </span>
        <div class="text-lg font-bold text-slate-800">{{ stat.count }}</div>
        <div class="text-xs text-slate-500 mt-0.5">{{ stat.weight }} kg</div>
        <div class="mt-2 h-1.5 w-full rounded-full bg-slate-100">
          <div class="h-1.5 rounded-full bg-blue-500 transition-all"
               :style="{ width: `${(stat.count / maxCount) * 100}%` }" />
        </div>
      </div>
    </div>

    <!-- Form Modal -->
    <div v-if="showForm"
         class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
         @click.self="showForm = false">
      <div class="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-lg font-bold text-slate-800">
            {{ editing ? 'ویرایش ورزشکار' : 'افزودن ورزشکار' }}
          </h3>
          <span v-if="!editing && lastAddedCount"
                class="rounded-full bg-green-50 px-2.5 py-1 text-xs font-medium text-green-700">
            {{ lastAddedCount }} ورزشکار در این نشست
          </span>
        </div>

        <div class="space-y-4">
          <!-- وزن: اول، چون تعیین‌کننده‌ی بقیه‌ست -->
          <div>
            <label class="mb-1 block text-xs font-medium text-slate-600">وزن *</label>
            <div class="flex flex-wrap gap-1.5">
              <button v-for="w in weights" :key="w"
                      type="button"
                      @click="form.weightCategory = w"
                      class="flex flex-col items-center rounded-lg border px-2.5 py-1.5 text-xs font-medium leading-tight transition-colors"
                      :class="form.weightCategory === w
                        ? 'border-blue-600 bg-blue-600 text-white'
                        : 'border-slate-200 bg-white text-slate-600 hover:border-blue-300'">
                <span class="text-[10px] opacity-70">وزن {{ toFa(weightIndex[w]) }}</span>
                <span>
                  {{ w }}
                  <span class="opacity-60">({{ countByWeight[w] ?? 0 }})</span>
                </span>
              </button>
            </div>
            <p v-if="form.weightCategory" class="mt-1.5 text-xs text-slate-500">
              انتخاب‌شده: <span class="font-medium text-slate-700">{{ weightLabel(form.weightCategory) }}</span>
            </p>
          </div>

          <div>
            <label class="mb-1 block text-xs font-medium text-slate-600">نام و نام خانوادگی *</label>
            <input ref="nameInput" v-model="form.name" @keyup.enter="submitAndNext"
                   placeholder="مثلاً علی رضایی" class="input w-full" />
            <p v-if="duplicateName" class="mt-1 text-xs text-amber-600">
              ⚠️ «{{ form.name.trim() }}» قبلاً در {{ weightLabel(form.weightCategory) }} ثبت شده
            </p>
          </div>

          <div>
            <label class="mb-1 block text-xs font-medium text-slate-600">باشگاه</label>
            <input v-model="form.club" list="club-list" @keyup.enter="submitAndNext"
                   placeholder="نام باشگاه" class="input w-full" />
            <datalist id="club-list">
              <option v-for="c in knownClubs" :key="c" :value="c" />
            </datalist>
            <div v-if="recentClubs.length" class="mt-1.5 flex flex-wrap gap-1.5">
              <button v-for="c in recentClubs" :key="c" type="button" @click="form.club = c"
                      class="rounded-md bg-slate-100 px-2 py-1 text-xs text-slate-600 hover:bg-slate-200">
                {{ c }}
              </button>
            </div>
          </div>

          <div>
            <label class="mb-1 block text-xs font-medium text-slate-600">رنکینگ (اختیاری)</label>
            <div class="flex gap-2">
              <input v-model.number="form.ranking" type="number" min="1" @keyup.enter="submitAndNext"
                     placeholder="خالی = بدون رنک" class="input flex-1" />
              <button v-if="nextFreeRank" type="button" @click="form.ranking = nextFreeRank"
                      class="whitespace-nowrap rounded-lg border border-slate-300 px-3 text-xs text-slate-600 hover:bg-slate-50">
                رنک آزاد: {{ nextFreeRank }}
              </button>
            </div>
            <p v-if="form.weightCategory" class="mt-1 text-xs text-slate-500">
              <template v-if="takenRanks.length">
                رنک‌های گرفته‌شده در {{ weightLabel(form.weightCategory) }}: {{ takenRanks.join('، ') }}
              </template>
              <template v-else>هنوز رنکی در این وزن ثبت نشده</template>
            </p>
          </div>
        </div>

        <p v-if="formError" class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{{ formError }}</p>

        <div class="mt-5 flex flex-wrap items-center justify-end gap-2">
          <button @click="showForm = false"
                  class="rounded-lg px-4 py-2 text-sm text-slate-600 hover:bg-slate-100">انصراف</button>
          <button v-if="!editing" @click="submitAndNext"
                  class="rounded-lg border border-blue-600 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50">
            افزودن و بعدی
          </button>
          <button @click="submit"
                  class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">
            {{ editing ? 'ذخیره' : 'افزودن و بستن' }}
          </button>
        </div>
        <p v-if="!editing" class="mt-2 text-left text-xs text-slate-400">Enter = افزودن و بعدی</p>
      </div>
    </div>

    <!-- Reset Ranking Modal -->
    <div v-if="showReset"
         class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
         @click.self="showReset = false">
      <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl">
        <h3 class="mb-2 text-lg font-bold text-slate-800">ریست رنکینگ‌ها</h3>
        <p class="mb-4 text-sm text-slate-500">رنکینگ ورزشکاران پاک می‌شود. این کار قابل بازگشت نیست.</p>

        <div class="space-y-2">
          <label class="flex cursor-pointer items-center gap-2 rounded-lg border border-slate-200 p-3 text-sm hover:bg-slate-50">
            <input type="radio" value="all" v-model="resetScope" />
            <span>همه وزن‌ها <span class="text-slate-400">({{ rankedCountAll }} رنک)</span></span>
          </label>
          <label v-if="filterWeight"
                 class="flex cursor-pointer items-center gap-2 rounded-lg border border-slate-200 p-3 text-sm hover:bg-slate-50">
            <input type="radio" value="weight" v-model="resetScope" />
            <span>فقط {{ weightLabel(filterWeight) }} <span class="text-slate-400">({{ rankedCountFiltered }} رنک)</span></span>
          </label>
        </div>

        <div class="mt-5 flex justify-end gap-3">
          <button @click="showReset = false"
                  class="rounded-lg px-4 py-2 text-sm text-slate-600 hover:bg-slate-100">انصراف</button>
          <button @click="confirmReset"
                  class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">
            ریست کن
          </button>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!athletes.length"
         class="rounded-xl border-2 border-dashed border-slate-200 py-16 text-center text-slate-400">
      <p class="text-4xl mb-3">🥋</p>
      <p>هنوز ورزشکاری ثبت نشده</p>
      <p class="text-sm mt-1">از دکمه‌های بالا ورزشکار اضافه کنید یا فایل اکسل ایمپورت کنید</p>
    </div>

    <div v-else-if="!filteredAthletes.length"
         class="rounded-xl border-2 border-dashed border-slate-200 py-10 text-center text-slate-400">
      <p class="text-3xl mb-2">🔍</p>
      <p>نتیجه‌ای یافت نشد</p>
    </div>

    <!-- Table -->
    <div v-else class="overflow-x-auto rounded-xl border border-slate-200">
      <table class="w-full text-sm">
        <thead class="bg-slate-50 text-slate-600">
        <tr>
          <th class="px-4 py-3 text-right font-medium">#</th>
          <th class="px-4 py-3 text-right font-medium">نام</th>
          <th class="px-4 py-3 text-right font-medium">باشگاه</th>
          <th class="px-4 py-3 text-right font-medium cursor-pointer select-none" @click="toggleSort">
            وزن <span class="text-xs text-blue-500 mr-1">{{ sortOrder === 'asc' ? '↑' : '↓' }}</span>
          </th>
          <th class="px-4 py-3 text-right font-medium">رنکینگ</th>
          <th class="px-4 py-3 text-right font-medium">عملیات</th>
        </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
        <template v-for="group in groupedAthletes" :key="group.weight">
          <tr class="bg-blue-50/60">
            <td colspan="6" class="px-4 py-2">
              <div class="flex items-center gap-2">
                <span v-if="group.index"
                      class="rounded-md bg-blue-600 px-2 py-0.5 text-xs font-bold text-white">
                  وزن {{ toFa(group.index) }}
                </span>
                <span class="font-semibold text-blue-700 text-sm">{{ group.weight }} kg</span>
                <span class="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-600">
                  {{ group.athletes.length }} نفر
                </span>
                <button @click="openAdd(group.weight)"
                        class="mr-auto text-xs text-blue-600 hover:text-blue-800">+ افزودن به این وزن</button>
              </div>
            </td>
          </tr>
          <tr v-for="(a, i) in group.athletes" :key="a.id" class="hover:bg-slate-50">
            <td class="px-4 py-3 text-slate-400">{{ i + 1 }}</td>
            <td class="px-4 py-3 font-medium text-slate-800">{{ a.name }}</td>
            <td class="px-4 py-3 text-slate-600">{{ a.club || '—' }}</td>
            <td class="px-4 py-3 text-slate-600">{{ a.weightCategory }}</td>
            <td class="px-4 py-3 text-slate-600">{{ a.ranking ?? '—' }}</td>
            <td class="px-4 py-3">
              <div class="flex gap-3">
                <button @click="startEdit(a)" class="text-blue-600 hover:text-blue-800">ویرایش</button>
                <button @click="remove(a.id)" class="text-red-500 hover:text-red-700">حذف</button>
              </div>
            </td>
          </tr>
        </template>
        </tbody>
      </table>
    </div>

    <!-- Toast -->
    <div v-if="toast"
         class="fixed bottom-6 left-1/2 z-50 -translate-x-1/2 rounded-lg bg-slate-800 px-5 py-3 text-sm text-white shadow-lg">
      {{ toast }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { useTournamentStore } from '../stores/tournament'
import * as XLSX from 'xlsx'
import { WEIGHT_CATEGORIES } from '../data/categories'

const store = useTournamentStore()
const athletes = computed(() => store.currentTournament?.athletes ?? [])

// --- فیلتر / سرچ ---
const searchName   = ref('')
const searchClub   = ref('')
const filterWeight = ref('')
const sortOrder    = ref('asc')

function clearFilters() {
  searchName.value = ''
  searchClub.value = ''
  filterWeight.value = ''
}
function toggleSort() {
  sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
}

const weights = computed(() => {
  const t = store.currentTournament
  if (!t) return []
  return WEIGHT_CATEGORIES[t.ageCategory]?.[t.gender] ?? []
})

// --- شماره ترتیبی اوزان (بر اساس ترتیب رسمی در categories.ts) ---
// وزن ۱ = اولین وزن آن رده/جنسیت، مستقل از اینکه ورزشکار داشته باشد یا نه
const weightIndex = computed(() => {
  const map = {}
  weights.value.forEach((w, i) => { map[String(w)] = i + 1 })
  return map
})

function weightNo(w) {
  return weightIndex.value[String(w)] ?? null
}

// «وزن ۳ (‎-51 kg)» — برای متن‌های توضیحی
function weightLabel(w) {
  if (!w) return ''
  const n = weightNo(w)
  return n ? `وزن ${toFa(n)} (${w} kg)` : `${w} kg`
}

// نمایش عدد با ارقام فارسی
function toFa(n) {
  if (n == null) return ''
  return String(n).replace(/\d/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[d])
}

function weightNum(w) {
  return parseFloat(String(w).replace(/[^0-9.]/g, '')) || 0
}

const filteredAthletes = computed(() => {
  const name = searchName.value.trim().toLowerCase()
  const club = searchClub.value.trim().toLowerCase()
  const wcat = filterWeight.value

  return athletes.value.filter((a) => {
    if (name && !a.name.toLowerCase().includes(name)) return false
    if (club && !(a.club ?? '').toLowerCase().includes(club)) return false
    if (wcat && a.weightCategory !== wcat) return false
    return true
  })
})

const groupedAthletes = computed(() => {
  const map = new Map()
  for (const a of filteredAthletes.value) {
    if (!map.has(a.weightCategory)) map.set(a.weightCategory, [])
    map.get(a.weightCategory).push(a)
  }

  return [...map.entries()]
      .sort(([a], [b]) => {
        // ترتیب رسمی اوزان؛ وزن‌های ناشناخته انتهای لیست
        const ia = weightIndex.value[String(a)] ?? 999
        const ib = weightIndex.value[String(b)] ?? 999
        const diff = ia !== ib ? ia - ib : weightNum(a) - weightNum(b)
        return sortOrder.value === 'asc' ? diff : -diff
      })
      .map(([weight, aths]) => ({
        weight,
        index: weightNo(weight),
        athletes: [...aths].sort((a, b) => {
          if (a.ranking == null && b.ranking == null) return 0
          if (a.ranking == null) return 1
          if (b.ranking == null) return -1
          return a.ranking - b.ranking
        }),
      }))
})

const countByWeight = computed(() => {
  const map = {}
  for (const a of athletes.value) map[a.weightCategory] = (map[a.weightCategory] ?? 0) + 1
  return map
})

// ترتیب کارت‌ها = ترتیب رسمی؛ شماره وزن از weightIndex می‌آید (پرش‌دار می‌ماند)
const weightStats = computed(() =>
    weights.value
        .map((w, i) => ({ weight: w, index: i + 1, count: countByWeight.value[w] ?? 0 }))
        .filter((s) => s.count > 0)
)

const maxCount = computed(() => weightStats.value.reduce((m, s) => Math.max(m, s.count), 1))

// --- منوی ابزارها ---
const showTools = ref(false)
function runTool(fn) {
  showTools.value = false
  fn()
}

// --- فرم ---
const showForm  = ref(false)
const editing   = ref(null)
const formError = ref('')
const nameInput = ref(null)
const lastAddedCount = ref(0)
const form = ref({ name: '', club: '', weightCategory: '', ranking: null })

// باشگاه‌های ثبت‌شده برای autocomplete
const knownClubs = computed(() =>
    [...new Set(athletes.value.map((a) => (a.club ?? '').trim()).filter(Boolean))].sort()
)
// آخرین باشگاه‌های استفاده‌شده (میان‌بر)
const recentClubs = computed(() =>
    [...new Set(athletes.value.slice(-12).map((a) => (a.club ?? '').trim()).filter(Boolean))]
        .slice(0, 5)
)

const takenRanks = computed(() =>
    athletes.value
        .filter((a) => a.weightCategory === form.value.weightCategory && a.ranking != null && a.id !== editing.value?.id)
        .map((a) => a.ranking)
        .sort((a, b) => a - b)
)

const nextFreeRank = computed(() => {
  if (!form.value.weightCategory) return null
  const taken = new Set(takenRanks.value)
  let n = 1
  while (taken.has(n)) n++
  return n
})

const duplicateName = computed(() => {
  const n = form.value.name.trim().toLowerCase()
  if (!n || !form.value.weightCategory) return false
  return athletes.value.some(
      (a) => a.id !== editing.value?.id &&
          a.weightCategory === form.value.weightCategory &&
          a.name.trim().toLowerCase() === n
  )
})

function focusName() {
  nextTick(() => nameInput.value?.focus())
}

// وزن پیش‌فرض: فیلتر فعال ← تنها وزن موجود ← خالی
function defaultWeight() {
  if (filterWeight.value) return filterWeight.value
  return weights.value.length === 1 ? weights.value[0] : ''
}

function openAdd(weight) {
  editing.value = null
  lastAddedCount.value = 0
  form.value = { name: '', club: '', weightCategory: weight || defaultWeight(), ranking: null }
  formError.value = ''
  showForm.value = true
  focusName()
}

function persist() {
  const { name, weightCategory } = form.value
  if (!name.trim() || !weightCategory) {
    formError.value = 'نام و وزن اجباری هستند'
    return false
  }

  const tid = store.currentTournamentId
  const payload = { ...form.value, name: name.trim(), ranking: form.value.ranking || undefined }

  const result = editing.value
      ? store.updateAthlete(tid, { ...editing.value, ...payload })
      : store.addAthlete(tid, payload)

  if (result?.error) {
    formError.value = result.error
    return false
  }

  formError.value = ''
  return true
}

function submit() {
  if (!persist()) return
  showForm.value = false
}

// افزودن و آماده‌سازی برای نفر بعدی؛ وزن و باشگاه حفظ می‌شوند
function submitAndNext() {
  if (editing.value) return submit()
  if (!persist()) return
  lastAddedCount.value++
  form.value = {
    name: '',
    club: form.value.club,
    weightCategory: form.value.weightCategory,
    ranking: null,
  }
  focusName()
}

function startEdit(a) {
  editing.value = a
  lastAddedCount.value = 0
  form.value = {
    name: a.name,
    club: a.club ?? '',
    weightCategory: a.weightCategory,
    ranking: a.ranking ?? null,
  }
  formError.value = ''
  showForm.value = true
  focusName()
}

function remove(id) {
  if (confirm('ورزشکار حذف شود؟')) {
    const result = store.removeAthlete(store.currentTournamentId, id)
    if (result.error) alert(result.error)
  }
}

// --- ریست رنکینگ ---
const showReset  = ref(false)
const resetScope = ref('all')

const rankedCountAll = computed(() => athletes.value.filter((a) => a.ranking != null).length)
const rankedCountFiltered = computed(() =>
    athletes.value.filter((a) => a.ranking != null && a.weightCategory === filterWeight.value).length
)

function openReset() {
  if (!store.currentTournament) { notify('ابتدا یک مسابقه انتخاب کنید'); return }
  if (!rankedCountAll.value) { notify('هیچ رنکینگی ثبت نشده'); return }
  resetScope.value = filterWeight.value ? 'weight' : 'all'
  showReset.value = true
}

function confirmReset() {
  const scope = resetScope.value === 'weight' ? filterWeight.value : undefined
  const { count } = store.resetRankings(store.currentTournamentId, scope)
  showReset.value = false
  notify(count ? `${count} رنکینگ پاک شد ✅` : 'رنکینگی برای پاک کردن نبود')
}

// --- Excel Import ---
async function importExcel(e) {
  const file = e.target.files[0]
  if (!file) return
  if (!store.currentTournament) { notify('ابتدا یک مسابقه انتخاب کنید'); e.target.value = ''; return }

  const data = await file.arrayBuffer()
  const wb   = XLSX.read(data)
  const allowed = weights.value.map(String)
  let added = 0, skipped = 0, rankDup = 0

  wb.SheetNames.forEach((name) => {
    const rows = XLSX.utils.sheet_to_json(wb.Sheets[name])
    // پشتیبانی از نام شیت‌های «وزن ۳ (-51)» و «وزن -51»
    const sheetWeight = extractWeightFromSheetName(name, allowed)

    rows.forEach((r) => {
      const athleteName = String(r['نام'] ?? r['name'] ?? '').trim()
      if (!athleteName) return

      let weightCategory = String(r['وزن'] ?? r['weight'] ?? '').trim()
      if (!weightCategory) weightCategory = sheetWeight
      if (!allowed.includes(weightCategory)) { skipped++; return }

      const res = store.addAthlete(store.currentTournamentId, {
        name: athleteName,
        club: String(r['باشگاه'] ?? r['club'] ?? ''),
        weightCategory,
        ranking: Number(r['رنکینگ'] ?? r['ranking']) || undefined,
      })

      if (res?.error) { rankDup++; return }
      added++
    })
  })

  e.target.value = ''
  const parts = [`${added} ورزشکار ثبت شد`]
  if (skipped) parts.push(`${skipped} وزن نامعتبر`)
  if (rankDup) parts.push(`${rankDup} رنک تکراری`)
  notify(parts.join(' | '))
}

// از نام شیت، وزن معتبر را بیرون می‌کشد
function extractWeightFromSheetName(name, allowed) {
  const raw = String(name).trim()
  // اول: هر وزن مجاز که داخل نام شیت آمده باشد
  const found = allowed
      .slice()
      .sort((a, b) => b.length - a.length)
      .find((w) => raw.includes(w))
  if (found) return found
  // fallback: حالت قدیمی «وزن -51»
  return raw.replace(/^وزن\s*/, '').replace(/[()]/g, '').trim()
}

// --- Excel Export ---
function exportTemplate() {
  const t = store.currentTournament
  if (!t) { notify('ابتدا یک مسابقه انتخاب کنید'); return }
  if (!weights.value.length) { notify('برای این مسابقه وزنی تعریف نشده'); return }

  const wb = XLSX.utils.book_new()
  const header = ['نام', 'باشگاه', 'وزن', 'رنکینگ']
  const usedNames = new Set()

  weights.value.forEach((w, i) => {
    const rows = Array.from({ length: 16 }, () => ['', '', String(w), ''])
    const ws = XLSX.utils.aoa_to_sheet([header, ...rows])
    ws['!cols'] = [{ wch: 22 }, { wch: 18 }, { wch: 10 }, { wch: 10 }]
    XLSX.utils.book_append_sheet(wb, ws, sheetName(`وزن ${i + 1} (${w})`, usedNames))
  })

  XLSX.writeFile(wb, buildFilename(t, 'قالب'))
  notify('قالب اکسل با موفقیت دانلود شد ✅')
}

function exportData() {
  const t = store.currentTournament
  if (!t) { notify('ابتدا یک مسابقه انتخاب کنید'); return }
  if (!athletes.value.length) { notify('هنوز ورزشکاری ثبت نشده'); return }

  const wb = XLSX.utils.book_new()
  const usedNames = new Set()

  weights.value.forEach((w, i) => {
    const members = athletes.value
        .filter((a) => String(a.weightCategory) === String(w))
        .sort((a, b) => (a.ranking ?? 9999) - (b.ranking ?? 9999))

    const rows = members.map((a, idx) => [idx + 1, a.name ?? '', a.club ?? '', w, a.ranking ?? ''])
    const ws = XLSX.utils.aoa_to_sheet([['#', 'نام', 'باشگاه', 'وزن', 'رنکینگ'], ...rows])
    ws['!cols'] = [{ wch: 5 }, { wch: 22 }, { wch: 18 }, { wch: 10 }, { wch: 10 }]
    XLSX.utils.book_append_sheet(wb, ws, sheetName(`وزن ${i + 1} (${w})`, usedNames))
  })

  XLSX.writeFile(wb, buildFilename(t, 'ورزشکاران'))
  notify('اکسل ورزشکاران با موفقیت دانلود شد ✅')
}

// --- Helpers ---
function sheetName(name, used) {
  let base = String(name).replace(/[\\/?*[\]:]/g, '').slice(0, 31) || 'Sheet'
  let final = base, n = 2
  while (used.has(final)) {
    const suffix = ` ${n}`
    final = base.slice(0, 31 - suffix.length) + suffix
    n++
  }
  used.add(final)
  return final
}

function buildFilename(t, label) {
  const GENDER_LABELS = { male: 'مردان', female: 'زنان' }
  const parts = [t.name, GENDER_LABELS[t.gender] ?? t.gender, t.ageCategory, toJalali(t.date)].filter(Boolean)
  return `${parts.join(' - ')} - ${label}.xlsx`
}

function toJalali(dateStr) {
  const d = dateStr ? new Date(dateStr) : new Date()
  if (isNaN(d.getTime())) return ''
  return new Intl.DateTimeFormat('fa-IR-u-nu-latn', {
    calendar: 'persian', day: 'numeric', month: 'long', year: 'numeric',
  }).format(d)
}

// --- Toast ---
const toast = ref('')
let toastTimer
function notify(msg) {
  toast.value = msg
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = ''), 3500)
}
</script>
