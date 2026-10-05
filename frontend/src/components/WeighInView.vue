<template>
  <div class="space-y-6">

    <!-- نوار وضعیت کلی -->
    <div class="flex flex-wrap items-center gap-3">
      <h2 class="text-lg font-bold text-slate-800">⚖️ وزن‌کشی</h2>
      <div class="flex flex-wrap gap-2">
        <button v-for="c in statusChips" :key="c.key"
                @click="filterStatus = filterStatus === c.key ? '' : c.key"
                class="rounded-full border px-3 py-1.5 text-xs font-medium transition-colors"
                :class="filterStatus === c.key ? c.active : c.idle">
          {{ c.label }} {{ toFa(c.count) }}
        </button>
      </div>
      <button
          v-if="athletes.length"
          type="button"
          :disabled="bulkApproving || countByStatus.pending + countByStatus.failed === 0"
          class="mr-auto rounded-lg bg-emerald-600 px-4 py-2 text-sm font-bold text-white shadow-sm hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-40"
          @click="approveAll"
      >
        {{ bulkApproving ? 'در حال تأیید…' : 'تأیید وزن‌کشی همه' }}
      </button>
      <span class="self-center text-sm text-slate-500">
        {{ toFa(doneCount) }} از {{ toFa(athletes.length) }} انجام شده
      </span>
    </div>

    <!-- نوع گروه‌بندی لیست -->
    <div class="flex flex-wrap items-center gap-2">
      <span class="ml-1 text-sm font-medium text-slate-600">نمایش بر اساس:</span>

      <button
          type="button"
          @click="groupBy = 'weight'"
          class="rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
          :class="groupBy === 'weight'
      ? 'border-blue-600 bg-blue-600 text-white'
      : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50'"
      >
        ⚖️ دستهٔ وزنی
      </button>

      <button
          type="button"
          @click="groupBy = 'club'"
          class="rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
          :class="groupBy === 'club'
      ? 'border-blue-600 bg-blue-600 text-white'
      : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50'"
      >
        👥 باشگاه / تیم
      </button>

      <span class="text-xs text-slate-400">
    <template v-if="groupBy === 'weight'">
      ورزشکاران بر اساس دستهٔ وزنی نمایش داده می‌شوند
    </template>
    <template v-else>
      اعضای هر باشگاه، مستقل از دستهٔ وزنی، کنار هم نمایش داده می‌شوند
    </template>
  </span>
    </div>

    <!-- جست‌وجو و مرتب‌سازی -->
    <div class="space-y-2 rounded-xl bg-slate-50 p-4">
      <div class="flex flex-wrap gap-3">
        <input
            ref="searchInput"
            v-model="searchName"
            @keyup.enter="enterSearch"
            aria-label="جست‌وجوی ورزشکار یا باشگاه"
            placeholder="جست‌وجوی نام ورزشکار یا باشگاه..."
            class="input min-w-[240px] flex-1 text-base"
        />

        <select
            v-model="filterClub"
            aria-label="فیلتر تیم یا باشگاه"
            class="input min-w-[210px] flex-1"
        >
          <option value="">همهٔ تیم‌ها / باشگاه‌ها</option>
          <option value="__no_club__">بدون باشگاه</option>

          <option
              v-for="club in clubs"
              :key="club.value"
              :value="club.value"
          >
            {{ club.label }}
          </option>
        </select>


        <select
            v-model="filterWeight"
            aria-label="فیلتر وزن"
            class="input min-w-[180px]"
        >
          <option value="">همه وزن‌ها</option>
          <option v-for="w in weights" :key="w" :value="w">
            وزن {{ toFa(weightIndex[w]) }} — {{ w }} kg
          </option>
        </select>

        <button
            v-if="searchName || filterClub || filterWeight || filterStatus"
            @click="clearFilters"
            class="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-600 hover:bg-slate-100"
        >
          ✕ پاک کردن
        </button>
      </div>

      <p class="text-xs text-slate-400">
        مرتب‌سازی داخل هر دستهٔ وزنی انجام می‌شود.
        Enter = اگر فقط یک نتیجه بود، پنجرهٔ وزن‌کشی همان نفر باز می‌شود.
      </p>
    </div>


    <!-- پیشرفت هر وزن -->
    <div
      v-if="groupBy === 'weight' && weightStats.length"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5"
    >
      <div
          v-for="stat in weightStats"
          :key="stat.weight"
          class="relative rounded-xl border p-3 transition-all"
          :class="filterWeight === stat.weight
      ? 'border-blue-500 bg-blue-50 shadow-sm'
      : 'border-slate-200 bg-white hover:border-blue-300 hover:bg-blue-50/50'"
      >
        <button
            type="button"
            class="w-full text-center"
            @click="filterWeight = filterWeight === stat.weight ? '' : stat.weight"
        >
    <span class="absolute right-2 top-2 rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-500">
      وزن {{ toFa(stat.index) }}
    </span>

          <span v-if="stat.complete" class="absolute left-2 top-2 text-xs">✅</span>

          <div class="text-lg font-bold text-slate-800">
            {{ toFa(stat.done) }}
            <span class="text-sm font-normal text-slate-400">
        /{{ toFa(stat.count) }}
      </span>
          </div>

          <div class="mt-0.5 text-xs text-slate-500">
            {{ stat.weight }} kg
          </div>

          <div class="mt-2 h-1.5 w-full rounded-full bg-slate-100">
            <div
                class="h-1.5 rounded-full transition-all"
                :class="stat.complete ? 'bg-green-500' : 'bg-blue-500'"
                :style="{ width: `${(stat.done / stat.count) * 100}%` }"
            />
          </div>
        </button>

        <button
            v-if="hasBracketForCategory(stat.weight)"
            type="button"
            class="mt-3 w-full rounded-lg border border-red-200 bg-red-50 px-2 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100"
            @click.stop="requestBracketReset(stat.weight)"
        >
          حذف براکت و ریست وزن
        </button>
      </div>
    </div>

    <!-- Empty states -->
    <div v-if="!athletes.length"
         class="rounded-xl border-2 border-dashed border-slate-200 py-16 text-center text-slate-400">
      <p class="mb-3 text-4xl">⚖️</p>
      <p>ورزشکاری برای وزن‌کشی ثبت نشده</p>
      <p class="mt-1 text-sm">ابتدا از صفحهٔ ورزشکاران، افراد را اضافه کنید</p>
    </div>

    <div v-else-if="!filteredAthletes.length"
         class="rounded-xl border-2 border-dashed border-slate-200 py-10 text-center text-slate-400">
      <p class="mb-2 text-3xl">🔍</p>
      <p>نتیجه‌ای یافت نشد</p>
    </div>

    <!-- جدول -->
    <div v-else class="overflow-x-auto rounded-xl border border-slate-200">
      <table class="w-full text-sm">
        <thead class="bg-slate-50 text-slate-600">
        <tr>
          <th class="px-4 py-3 text-right font-medium">#</th>
          <th class="px-4 py-3 text-right font-medium">نام</th>
          <th class="px-4 py-3 text-right font-medium">باشگاه</th>
          <th class="px-4 py-3 text-right font-medium">حد مجاز</th>
          <th class="px-4 py-3 text-right font-medium">وزن ثبت‌شده</th>
          <th class="px-4 py-3 text-right font-medium">نوبت‌ها</th>
          <th class="px-4 py-3 text-right font-medium">وضعیت</th>
          <th class="px-4 py-3 text-right font-medium">امضا</th>
          <th class="px-4 py-3 text-right font-medium">عملیات</th>
        </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
        <template v-for="group in groupedAthletes" :key="group.key">
          <!-- هدر گروه -->
          <tr
              class="bg-blue-50/60"
              :class="groupBy === 'club' ? 'bg-violet-50/70' : 'bg-blue-50/60'"
          >
            <td colspan="9" class="px-4 py-2">
              <div class="flex flex-wrap items-center gap-2">
                <!-- محتوای فعلی هدر گروه تو -->
              </div>
            </td>
          </tr>

          <!-- این قسمت در کد تو وجود نداشت -->
          <tr
              v-for="(a, i) in group.athletes"
              :key="a.id"
              class="hover:bg-slate-50"
              :class="rowTint(a)"
          >
            <td class="px-4 py-3 text-slate-400">
              {{ toFa(i + 1) }}
            </td>

            <td class="px-4 py-3 font-medium text-slate-800">
              {{ a.name }}
            </td>

            <td class="px-4 py-3 text-slate-600">
              {{ a.club || '—' }}
            </td>

            <td class="px-4 py-3 text-slate-600">
              <div v-if="groupBy === 'club'" class="mb-1 text-xs font-medium text-blue-600">
                {{ weightLabel(a.weightCategory) }}
              </div>

              {{ limitText(a.weightCategory) }}
            </td>

            <td
                class="px-4 py-3 font-medium tabular-nums"
                :class="statusMeta(a).text"
            >
              {{ lastWeight(a) != null ? formatKg(lastWeight(a)) : '—' }}
            </td>

            <td class="whitespace-nowrap px-4 py-3">
        <span class="inline-flex gap-1 align-middle">
          <span
              v-for="n in MAX_ATTEMPTS"
              :key="n"
              class="h-2 w-2 rounded-full"
              :class="n <= attemptCount(a)
              ? 'bg-slate-500'
              : 'bg-slate-200'"
          />
        </span>

              <span class="mr-2 text-xs text-slate-400">
          {{ toFa(attemptCount(a)) }}/{{ toFa(MAX_ATTEMPTS) }}
        </span>
            </td>

            <td class="px-4 py-3">
        <span
            class="rounded-full px-2 py-0.5 text-xs font-medium"
            :class="statusMeta(a).badge"
        >
          {{ statusMeta(a).label }}
        </span>
            </td>

            <td class="px-4 py-3">
              <span v-if="a.weighIn?.signature" class="text-green-600">✔</span>
              <span v-else class="text-slate-300">—</span>
            </td>

            <td class="whitespace-nowrap px-4 py-3">
              <div class="flex gap-3">
                <button
                    type="button"
                    @click="openWeigh(a)"
                    class="font-medium text-blue-600 hover:text-blue-800"
                >
                  {{ isFinal(a) ? 'مشاهده' : 'وزن‌کشی' }}
                </button>

                <button
                    v-if="isFinal(a)"
                    type="button"
                    @click="doReset(a)"
                    class="text-amber-600 hover:text-amber-700"
                >
                  ریست
                </button>
              </div>
            </td>
          </tr>
        </template>
        </tbody>
      </table>
    </div>

    <!-- ===== مودال وزن‌کشی ===== -->
    <div v-if="active" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
         @click.self="closeWeigh">
      <div class="w-full max-w-xl overflow-hidden rounded-2xl bg-white shadow-xl">

        <!-- هویت ورزشکار -->
        <div class="flex items-start justify-between border-b border-slate-100 bg-slate-50 px-6 py-4">
          <div>
            <h3 class="text-lg font-bold text-slate-800">{{ active.name }}</h3>
            <p class="mt-0.5 text-sm text-slate-500">
              {{ active.club || 'بدون باشگاه' }} · {{ weightLabel(active.weightCategory) }}
            </p>
          </div>
          <button @click="closeWeigh"
                  class="rounded-lg p-1 text-slate-400 hover:bg-slate-200 hover:text-slate-600">✕</button>
        </div>

        <div class="max-h-[75vh] space-y-5 overflow-y-auto p-6">

          <!-- حد مجاز دسته وزنی -->
          <div class="grid grid-cols-3 gap-2.5">
            <!-- کف وزن (حد پایین) -->
            <div class="rounded-xl border border-slate-200 bg-white p-3 text-center">
              <p class="text-[11px] font-medium text-slate-500">حد پایین (کف)</p>
              <p class="mt-1 text-xl font-bold tabular-nums text-slate-800">
                <template v-if="activeLimit?.isOpen">
                  {{ formatKg(activeLimit.minKg) }}
                </template>
                <template v-else-if="activeLimit?.minKg != null">
                  {{ formatKg(activeLimit.minKg) }}
                </template>
                <template v-else>
                  —
                </template>
                <span class="text-xs font-normal text-slate-400"> kg</span>
              </p>
              <p class="text-[10px] text-slate-400">
                {{ activeLimit?.isOpen ? 'حداقل مجاز (+)' : (activeLimit?.minKg ? 'بیشتر از این وزن' : 'بدون کف') }}
              </p>
            </div>

            <!-- سقف قانونی -->
            <div class="rounded-xl border border-slate-200 bg-white p-3 text-center">
              <p class="text-[11px] font-medium text-slate-500">سقف قانونی دسته</p>
              <p class="mt-1 text-xl font-bold tabular-nums text-slate-800">
                <template v-if="activeLimit?.isOpen">
                  آزاد
                </template>
                <template v-else>
                  {{ formatKg(activeLimit?.maxKg) }}
                  <span class="text-xs font-normal text-slate-400">kg</span>
                </template>
              </p>
              <p class="text-[10px] text-slate-400">
                {{ activeLimit?.isOpen ? 'سقف ندارد' : 'وزن اسمی دسته' }}
              </p>
            </div>

            <!-- با ارفاق رسمی فدراسیون -->
            <div class="rounded-xl border border-amber-200 bg-amber-50/70 p-3 text-center">
              <p class="text-[11px] font-medium text-amber-800">حداکثر با ارفاق</p>
              <p class="mt-1 text-xl font-bold tabular-nums text-amber-900">
                <template v-if="activeLimit?.isOpen">
                  —
                </template>
                <template v-else>
                  {{ formatKg(activeLimit?.effectiveMax) }}
                  <span class="text-xs font-normal text-amber-600">kg</span>
                </template>
              </p>
              <p class="text-[10px] text-amber-700">
                {{ activeLimit?.isOpen ? 'ارفاق اعمال نمی‌شود' : `+${toFa(200)} گرم ارفاق` }}
              </p>
            </div>
          </div>

          <!-- نوبت‌ها -->
          <div>
            <p class="mb-2 text-xs font-medium text-slate-600">
              نوبت‌ها ({{ toFa(attemptCount(active)) }} از {{ toFa(MAX_ATTEMPTS) }})
            </p>
            <div class="grid grid-cols-2 gap-2">
              <div
                  v-for="n in MAX_ATTEMPTS"
                  :key="n"
                  class="rounded-lg border p-2.5 text-center"
                  :class="attemptAt(active, n)
    ? (attemptAt(active, n).ok
      ? 'border-green-200 bg-green-50'
      : 'border-red-200 bg-red-50')
    : 'border-dashed border-slate-200 bg-slate-50'"
              >
                <p class="text-[10px] text-slate-500">نوبت {{ toFa(n) }}</p>

                <p
                    v-if="attemptAt(active, n)"
                    class="text-base font-bold tabular-nums"
                    :class="attemptAt(active, n).ok ? 'text-green-700' : 'text-red-600'"
                >
                  {{ formatKg(attemptAt(active, n).weightKg) }}
                </p>

                <p v-else class="text-base text-slate-300">—</p>
              </div>
            </div>
          </div>

          <!-- ورود وزن -->
          <div v-if="canWeighActive">
            <label class="mb-1 block text-xs font-medium text-slate-600">
              وزن روی باسکول (kg) — نوبت {{ toFa(attemptCount(active) + 1) }}
            </label>
            <div class="flex gap-2">
              <input ref="weightInput" v-model="weightInputValue" @keyup.enter="submitWeight"
                     type="number" step="0.01" min="0" inputmode="decimal" placeholder="مثلاً 57.85"
                     class="input flex-1 text-center text-2xl font-bold tabular-nums" />
              <div class="flex flex-col gap-1">
                <button type="button" @click="bump(0.05)"
                        class="rounded-md border border-slate-200 px-3 text-xs text-slate-600 hover:bg-slate-50">
                  +۰٫۰۵
                </button>
                <button type="button" @click="bump(-0.05)"
                        class="rounded-md border border-slate-200 px-3 text-xs text-slate-600 hover:bg-slate-50">
                  −۰٫۰۵
                </button>
              </div>
            </div>

            <!-- پیش‌نمایش زنده -->
            <div v-if="preview" class="mt-3 rounded-lg px-4 py-3 text-sm font-medium"
                 :class="preview.passed ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-600'">
              <template v-if="preview.passed">
                ✅ قبول — اختلاف {{ formatKg(Math.abs(preview.diff)) }} kg
                <span v-if="preview.withTolerance" class="text-amber-600">
                  (با استفاده از ارفاق)
                </span>
              </template>
              <template v-else>
                ❌ مردود — {{ formatKg(Math.abs(preview.diff)) }} kg اختلاف
                <span v-if="attemptsLeftActive > 1"> · {{ toFa(attemptsLeftActive - 1) }} نوبت دیگر باقی می‌ماند</span>
                <span v-else class="font-bold"> · این آخرین نوبت است</span>
              </template>
            </div>

            <p v-if="modalError" class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
              {{ modalError }}
            </p>

            <div class="mt-4 flex items-center justify-end gap-2">
              <button @click="closeWeigh" class="rounded-lg px-4 py-2 text-sm text-slate-600 hover:bg-slate-100">
                انصراف
              </button>
              <button @click="submitWeight" :disabled="!isValidWeight"
                      class="rounded-lg bg-blue-600 px-5 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-40">
                ثبت نوبت {{ toFa(attemptCount(active) + 1) }}
              </button>
            </div>
            <p class="mt-2 text-left text-xs text-slate-400">Enter = ثبت</p>
          </div>

          <!-- نتیجهٔ نهایی + امضا -->
          <div v-else class="space-y-4">
            <div class="rounded-xl px-4 py-4 text-center" :class="statusMeta(active).box">
              <p class="text-lg font-bold">{{ statusMeta(active).label }}</p>
              <p class="mt-1 text-sm opacity-80">
                وزن نهایی: {{ lastWeight(active) != null ? formatKg(lastWeight(active)) + ' kg' : '—' }}
              </p>
              <p v-if="statusOf(active) === 'failed'" class="mt-1 text-xs">
                این ورزشکار از قرعه‌کشی حذف می‌شود
              </p>
            </div>

            <div
                v-if="statusOf(active) === 'passed'"
                class="rounded-xl border border-slate-200 p-4"
            >
              <p class="mb-2 text-xs font-medium text-slate-600">
                امضای بازیکن
              </p>

              <div
                  v-if="active.weighIn?.signature"
                  class="flex items-center justify-between gap-3"
              >
                <span class="truncate text-sm text-green-700">✔ امضا ثبت شده</span>
                <button
                    @click="doClearSignature"
                    class="rounded-lg border border-red-200 px-3 py-1.5 text-xs text-red-600 hover:bg-red-50"
                >
                  حذف امضا
                </button>
              </div>

              <div v-else class="flex items-center justify-between gap-3">
                <span class="text-sm text-slate-500">هنوز امضا نشده</span>
                <button
                    @click="openSignaturePad"
                    :disabled="signing"
                    class="rounded-lg bg-slate-800 px-4 py-2 text-sm font-medium text-white hover:bg-slate-900 disabled:opacity-40"
                >
                  ✍️ گرفتن امضا
                </button>
              </div>
            </div>

            <p v-if="modalError" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
              {{ modalError }}
            </p>

            <div class="flex items-center justify-end gap-2">
              <button @click="doReset(active)"
                      class="rounded-lg border border-amber-300 px-4 py-2 text-sm text-amber-700 hover:bg-amber-50">
                ریست وزن‌کشی
              </button>
              <button @click="closeWeigh"
                      class="rounded-lg bg-blue-600 px-5 py-2 text-sm font-medium text-white hover:bg-blue-700">
                بستن و نفر بعد
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- تأیید ریست -->
    <div v-if="resetTarget" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4"
         @click.self="resetTarget = null">
      <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl">
        <h3 class="mb-2 text-lg font-bold text-slate-800">ریست وزن‌کشی</h3>
        <p class="mb-4 text-sm text-slate-500">
          تمام نوبت‌ها و امضای «{{ resetTarget.name }}» پاک می‌شود و وضعیت به «در انتظار» برمی‌گردد.
        </p>
        <div class="flex justify-end gap-3">
          <button @click="resetTarget = null"
                  class="rounded-lg px-4 py-2 text-sm text-slate-600 hover:bg-slate-100">انصراف</button>
          <button @click="confirmReset"
                  class="rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700">
            ریست کن
          </button>
        </div>
      </div>
    </div>

    <!-- تأیید حذف براکت و ریست وزن -->
    <div
        v-if="bracketResetTarget"
        class="fixed inset-0 z-[65] flex items-center justify-center bg-black/40 p-4"
        @click.self="bracketResetTarget = null"
    >
      <div class="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
        <h3 class="mb-2 text-lg font-bold text-slate-800">
          حذف براکت و ریست وزن
        </h3>

        <p class="mb-3 text-sm leading-6 text-slate-600">
          براکت وزن
          <strong>{{ bracketResetTarget.category }}</strong>
          حذف می‌شود و وزن‌کشی تمام ورزشکاران این دسته به حالت «در انتظار» برمی‌گردد.
        </p>

        <p class="mb-5 rounded-lg bg-red-50 px-3 py-2 text-xs leading-5 text-red-700">
          نتایج و مسابقات ذخیره‌شدهٔ این وزن نیز حذف خواهند شد. این عملیات قابل بازگشت نیست.
        </p>

        <div class="flex justify-end gap-3">
          <button
              type="button"
              @click="bracketResetTarget = null"
              class="rounded-lg px-4 py-2 text-sm text-slate-600 hover:bg-slate-100"
          >
            انصراف
          </button>

          <button
              type="button"
              @click="confirmBracketReset"
              class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
          >
            حذف براکت و ریست
          </button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="toast"
         class="fixed bottom-6 left-1/2 z-[70] -translate-x-1/2 rounded-lg bg-slate-800 px-5 py-3 text-sm text-white shadow-lg">
      {{ toast }}
    </div>
  </div>

  <!-- مودال پد رسم امضا -->
  <div v-if="showSignaturePad" class="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 p-4"
       @click.self="closeSignaturePad">
    <div class="w-full max-w-md overflow-hidden rounded-2xl bg-white shadow-2xl">
      <div class="flex items-center justify-between border-b border-slate-100 bg-slate-50 px-5 py-3.5">
        <div>
          <h3 class="font-bold text-slate-800">امضای بازیکن</h3>
          <p class="text-xs text-slate-500">امضای تأیید وزن‌کشی «{{ active?.name }}»</p>
        </div>
        <button @click="closeSignaturePad" class="rounded-lg p-1 text-slate-400 hover:bg-slate-200">✕</button>
      </div>

      <div class="p-5 space-y-3">
        <div class="rounded-xl border-2 border-dashed border-slate-300 bg-slate-50 p-1">
          <canvas ref="sigCanvas"
                  width="400"
                  height="180"
                  class="w-full h-[180px] bg-white rounded-lg cursor-crosshair touch-none"
                  @mousedown="startDrawing"
                  @mousemove="draw"
                  @mouseup="stopDrawing"
                  @mouseleave="stopDrawing"
                  @touchstart.prevent="handleTouchStart"
                  @touchmove.prevent="handleTouchMove"
                  @touchend.prevent="stopDrawing" />
        </div>
        <p class="text-center text-xs text-slate-400">با ماوس یا قلم/لمس در کادر بالا امضا کنید</p>
      </div>

      <div class="flex items-center justify-between border-t border-slate-100 bg-slate-50 px-5 py-3">
        <button @click="clearSignaturePad" type="button"
                class="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-600 hover:bg-slate-100">
          پاک کردن بوم
        </button>
        <div class="flex gap-2">
          <button @click="closeSignaturePad" type="button"
                  class="rounded-lg px-3 py-1.5 text-xs text-slate-600 hover:bg-slate-200">
            انصراف
          </button>
          <button @click="saveAndSubmitSignature" type="button" :disabled="signing"
                  class="rounded-lg bg-blue-600 px-4 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-40">
            {{ signing ? 'در حال ثبت...' : 'تأیید و ذخیره' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted } from 'vue'
import { useTournamentStore } from '../stores/tournament'
import { WEIGHT_CATEGORIES } from '../data/categories'
import { parseCategory, checkWeight, MAX_ATTEMPTS, TOLERANCE_KG } from '../utils/weighIn'

const store = useTournamentStore()
const athletes = computed(() => store.currentTournament?.athletes ?? [])
const tid = computed(() => store.currentTournamentId)

/* ---------- فیلترها ---------- */
const searchName = ref('')
const filterClub = ref('')
const filterWeight = ref('')
const filterStatus = ref('')
const groupBy = ref('weight')
const searchInput = ref(null)
const bulkApproving = ref(false)

async function approveAll() {
  const remaining = countByStatus.value.pending + countByStatus.value.failed
  if (!remaining || bulkApproving.value) return
  if (!confirm(`وزن‌کشی ${toFa(remaining)} ورزشکار بدون ثبت عدد وزن، یک‌جا تأیید شود؟`)) return
  bulkApproving.value = true
  const result = await store.approveAllWeighIns(tid.value)
  bulkApproving.value = false
  if (result.error) {
    notify(result.error)
    return
  }
  notify(`وزن‌کشی ${toFa(result.count)} ورزشکار تأیید شد`)
}

function focusSearch() {
  nextTick(() => searchInput.value?.focus())
}

onMounted(focusSearch)

function clearFilters() {
  searchName.value = ''
  filterClub.value = ''
  filterWeight.value = ''
  filterStatus.value = ''
  focusSearch()
}

// یکسان‌سازی حروف فارسی/عربی، فاصله‌ها و نیم‌فاصله
function normalizeText(value) {
  return String(value ?? '')
      .normalize('NFKC')
      .replace(/ي|ى/g, 'ی')
      .replace(/ك/g, 'ک')
      .replace(/[\u064B-\u065F\u0670\u0640]/g, '')
      .replace(/\u200c/g, ' ')
      .replace(/\s+/g, ' ')
      .trim()
      .toLowerCase()
}

const faCollator = new Intl.Collator('fa', {
  numeric: true,
  sensitivity: 'base',
})

const NO_CLUB_FILTER = '__no_club__'

const clubs = computed(() => {
  const map = new Map()

  for (const athlete of athletes.value) {
    const label = String(athlete.club ?? '').trim()
    const value = normalizeText(label)

    if (!value || map.has(value)) {
      continue
    }

    map.set(value, label)
  }

  return [...map.entries()]
      .map(([value, label]) => ({
        value,
        label,
      }))
      .sort((a, b) => {
        return faCollator.compare(
            normalizeText(a.label),
            normalizeText(b.label),
        )
      })
})

function compareNames(a, b) {
  return faCollator.compare(
      normalizeText(a.name),
      normalizeText(b.name),
  )
}

const statusOrder = {
  pending: 0,
  passed: 1,
  failed: 2,
}

function compareByStatusAndName(a, b) {
  return (
      (statusOrder[statusOf(a)] ?? 3) - (statusOrder[statusOf(b)] ?? 3)
      || compareNames(a, b)
  )
}

function compareByWeightThenStatusThenName(a, b) {
  const weightA = weightIndex.value[String(a.weightCategory)] ?? 999
  const weightB = weightIndex.value[String(b.weightCategory)] ?? 999

  return (
      weightA - weightB
      || weightNum(a.weightCategory) - weightNum(b.weightCategory)
      || compareByStatusAndName(a, b)
  )
}

const weights = computed(() => {
  const t = store.currentTournament
  if (!t) return []
  return WEIGHT_CATEGORIES[t.ageCategory]?.[t.gender] ?? []
})

const weightIndex = computed(() => {
  const map = {}
  weights.value.forEach((w, i) => { map[String(w)] = i + 1 })
  return map
})

function weightNo(w) { return weightIndex.value[String(w)] ?? null }
function weightLabel(w) {
  if (!w) return ''
  const n = weightNo(w)
  return n ? `وزن ${toFa(n)} (${w} kg)` : `${w} kg`
}
function toFa(n) {
  if (n == null) return ''
  return String(n).replace(/\d/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[d])
}
function weightNum(w) { return parseFloat(String(w).replace(/[^0-9.]/g, '')) || 0 }
function formatKg(n) { return Number(n).toFixed(2) }

/* ---------- وضعیت ---------- */
function statusOf(a)     { return a?.weighIn?.status ?? 'pending' }
function isFinal(a)      { return statusOf(a) !== 'pending' }
function attempts(a)     { return a?.weighIn?.attempts ?? [] }
function attemptCount(a) { return attempts(a).length }
function attemptAt(a, n) { return attempts(a)[n - 1] ?? null }
function lastWeight(a) {
  const list = attempts(a)
  return list.length ? list[list.length - 1].weightKg : null
}

function canWeighNow(a) { return !isFinal(a) && attemptCount(a) < MAX_ATTEMPTS }

const STATUS = {
  pending: { label: 'در انتظار', text: 'text-slate-400', badge: 'bg-slate-100 text-slate-600', box: 'bg-slate-100 text-slate-700' },
  passed:  { label: 'قبول',      text: 'text-green-700', badge: 'bg-green-100 text-green-700', box: 'bg-green-50 text-green-700' },
  failed:  { label: 'مردود',     text: 'text-red-600',   badge: 'bg-red-100 text-red-700',     box: 'bg-red-50 text-red-700' },
}
function statusMeta(a) { return STATUS[statusOf(a)] ?? STATUS.pending }
function rowTint(a) {
  const s = statusOf(a)
  return s === 'failed' ? 'bg-red-50/40' : s === 'passed' ? 'bg-green-50/30' : ''
}

/* ---------- حد مجاز ---------- */
function limitInfo(category) {
  const parsed = parseCategory(category)
  if (!parsed) return null

  return {
    limit: parsed.limitKg,
    effective: parsed.isOpen
        ? parsed.limitKg - TOLERANCE_KG
        : parsed.limitKg + TOLERANCE_KG,
    type: parsed.isOpen ? 'min' : 'max',
  }
}

/* ---------- حد مجاز و بازه وزنی ---------- */
function getCategoryLimits(cat) {
  if (!cat) return null
  const list = weights.value
  const idx = list.indexOf(cat)
  const isPlus = String(cat).startsWith('+')
  const limitNum = weightNum(cat)

  // اگر دسته با + شروع شود: سقف ندارد، فقط کف دارد
  if (isPlus) {
    return {
      isOpen: true,
      minKg: limitNum,
      maxKg: null,
      effectiveMax: null,
    }
  }

  // اگر اولین دسته وزنی باشد: کف ندارد (یا صفر است)
  // در غیر این صورت کف وزن برابر است با سقف دسته قبلی
  const prevCategory = idx > 0 ? list[idx - 1] : null
  const minKg = prevCategory ? weightNum(prevCategory) : null
  const maxKg = limitNum
  const effectiveMax = maxKg + TOLERANCE_KG

  return {
    isOpen: false,
    minKg,
    maxKg,
    effectiveMax,
  }
}

function limitText(cat) {
  const info = getCategoryLimits(cat)
  if (!info) return String(cat)

  if (info.isOpen) {
    return `بیش از ${formatKg(info.minKg)} kg`
  }

  if (info.minKg != null) {
    return `${formatKg(info.minKg)} تا ${formatKg(info.effectiveMax)}`
  }

  return `تا ${formatKg(info.maxKg)} (ارفاق تا ${formatKg(info.effectiveMax)})`
}

/* ---------- لیست ---------- */
const filteredAthletes = computed(() => {
  const query = normalizeText(searchName.value)
  const selectedClub = filterClub.value

  return athletes.value.filter((a) => {
    const name = normalizeText(a.name)
    const club = normalizeText(a.club)

    // جست‌وجوی عمومی بر اساس نام ورزشکار یا نام باشگاه
    if (
        query &&
        !name.includes(query) &&
        !club.includes(query)
    ) {
      return false
    }

    // فیلتر ورزشکاران بدون باشگاه
    if (selectedClub === NO_CLUB_FILTER) {
      if (club) {
        return false
      }
    }
    // فیلتر باشگاه انتخاب‌شده
    else if (selectedClub && club !== selectedClub) {
      return false
    }

    // فیلتر دستهٔ وزنی
    if (
        filterWeight.value !== '' &&
        String(a.weightCategory) !== String(filterWeight.value)
    ) {
      return false
    }

    // فیلتر وضعیت وزن‌کشی
    if (
        filterStatus.value &&
        statusOf(a) !== filterStatus.value
    ) {
      return false
    }

    return true
  })
})


const groupedAthletes = computed(() => {
  const map = new Map()

  for (const athlete of filteredAthletes.value) {
    const isClubMode = groupBy.value === 'club'

    // ورزشکار بدون باشگاه هم در یک گروه مستقل نمایش داده می‌شود
    const groupKey = isClubMode
        ? normalizeText(athlete.club) || '__no_club__'
        : String(athlete.weightCategory ?? '__no_weight__')

    const groupLabel = isClubMode
        ? (String(athlete.club ?? '').trim() || 'بدون باشگاه')
        : String(athlete.weightCategory ?? 'بدون دستهٔ وزنی')

    if (!map.has(groupKey)) {
      map.set(groupKey, {
        key: groupKey,
        label: groupLabel,
        weight: isClubMode ? null : athlete.weightCategory,
        index: isClubMode ? null : weightNo(athlete.weightCategory),
        athletes: [],
      })
    }

    map.get(groupKey).athletes.push(athlete)
  }

  const groups = [...map.values()]

  // ترتیب گروه‌ها
  groups.sort((a, b) => {
    if (groupBy.value === 'club') {
      // گروه «بدون باشگاه» همیشه انتهای لیست باشد
      if (a.key === '__no_club__') return 1
      if (b.key === '__no_club__') return -1

      return faCollator.compare(
          normalizeText(a.label),
          normalizeText(b.label),
      )
    }

    const weightA = weightIndex.value[String(a.weight)] ?? 999
    const weightB = weightIndex.value[String(b.weight)] ?? 999

    return weightA - weightB || weightNum(a.weight) - weightNum(b.weight)
  })

  return groups.map((group) => {
    const athletesInGroup = [...group.athletes].sort(
        groupBy.value === 'club'
            ? compareByWeightThenStatusThenName
            : compareByStatusAndName,
    )

    const done = athletesInGroup.filter(isFinal).length

    return {
      ...group,
      athletes: athletesInGroup,
      done,
      complete: athletesInGroup.length > 0 && athletesInGroup.every(isFinal),
    }
  })
})

const countByStatus = computed(() => {
  const c = { pending: 0, passed: 0, failed: 0 }
  for (const a of athletes.value) c[statusOf(a)]++
  return c
})
const doneCount = computed(() => countByStatus.value.passed + countByStatus.value.failed)

const statusChips = computed(() => [
  { key: 'pending', label: 'در انتظار', count: countByStatus.value.pending,
    idle: 'border-slate-200 bg-white text-slate-600 hover:border-slate-400',
    active: 'border-slate-600 bg-slate-600 text-white' },
  { key: 'passed', label: 'قبول', count: countByStatus.value.passed,
    idle: 'border-green-200 bg-white text-green-700 hover:border-green-400',
    active: 'border-green-600 bg-green-600 text-white' },
  { key: 'failed', label: 'مردود', count: countByStatus.value.failed,
    idle: 'border-red-200 bg-white text-red-600 hover:border-red-400',
    active: 'border-red-600 bg-red-600 text-white' },
])

function isComplete(w) { return store.isCategoryWeighInComplete(tid.value, w) }

const weightStats = computed(() =>
    weights.value
        .map((w, i) => {
          const list = athletes.value.filter((a) => String(a.weightCategory) === String(w))
          return {
            weight: w,
            index: i + 1,
            count: list.length,
            done: list.filter(isFinal).length,
            complete: list.length > 0 && list.every(isFinal),
          }
        })
        .filter((s) => s.count > 0)
)

/* ---------- مودال ---------- */
const active = ref(null)
const weightInput = ref(null)
const weightInputValue = ref('')
const modalError = ref('')
const signing = ref(false)

const activeLimit = computed(() => (active.value ? getCategoryLimits(active.value.weightCategory) : null))
const canWeighActive = computed(() => !!active.value && canWeighNow(active.value))
const attemptsLeftActive = computed(() =>
    active.value ? Math.max(0, MAX_ATTEMPTS - attemptCount(active.value)) : 0
)

const isValidWeight = computed(() => {
  const n = Number(weightInputValue.value)
  return weightInputValue.value !== '' && Number.isFinite(n) && n > 0 && n < 300
})

const preview = computed(() => {
  if (!active.value || !isValidWeight.value) return null

  // ارسال weights.value به عنوان آرگومان سوم
  const result = checkWeight(
      Number(weightInputValue.value),
      active.value.weightCategory,
      weights.value
  )

  return {
    passed: result.ok,
    diff: result.deltaKg,
    withTolerance: result.withTolerance,
    message: result.message,
  }
})

// رفرنس مودال را از استور تازه می‌کند (اگر متدها آبجکت جدید بسازند)
function refreshActive(id) {
  active.value = athletes.value.find((a) => a.id === id) ?? null
}

function openWeigh(a) {
  active.value = a
  weightInputValue.value = ''
  modalError.value = ''
  nextTick(() => weightInput.value?.focus())
}

function closeWeigh() {
  active.value = null
  weightInputValue.value = ''
  modalError.value = ''
  searchName.value = ''
}

function bump(d) {
  const n = Number(weightInputValue.value) || 0
  weightInputValue.value = Math.max(0, n + d).toFixed(2)
}

function enterSearch() {
  if (filteredAthletes.value.length === 1) openWeigh(filteredAthletes.value[0])
}

function submitWeight() {
  if (!active.value || !isValidWeight.value) return
  const id = active.value.id
  const res = store.recordWeighIn(tid.value, id, Number(weightInputValue.value))
  if (res?.error) { modalError.value = res.error; return }

  modalError.value = ''
  weightInputValue.value = ''
  refreshActive(id)
  if (!active.value) return

  const s = statusOf(active.value)
  if (s === 'passed')      notify(`${active.value.name}: قبول ✅`)
  else if (s === 'failed') notify(`${active.value.name}: مردود ❌`)
  else {
    notify(`نوبت ثبت شد — ${toFa(attemptsLeftActive.value)} نوبت باقی`)
    nextTick(() => weightInput.value?.focus())
  }
}

/* ---------- امضا ---------- */
const showSignaturePad = ref(false)
const sigCanvas = ref(null)
let isDrawing = false
let hasDrawn = false

function openSignaturePad() {
  showSignaturePad.value = true
  hasDrawn = false
  nextTick(() => {
    initCanvas()
  })
}

function closeSignaturePad() {
  showSignaturePad.value = false
  hasDrawn = false
}

function initCanvas() {
  const canvas = sigCanvas.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  ctx.strokeStyle = '#0f172a' // رنگ تیره امضا
  ctx.lineWidth = 2.5
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.clearRect(0, 0, canvas.width, canvas.height)
}

function clearSignaturePad() {
  initCanvas()
  hasDrawn = false
}

// رویدادهای ماوس
function startDrawing(e) {
  isDrawing = true
  hasDrawn = true
  const canvas = sigCanvas.value
  const ctx = canvas.getContext('2d')
  const rect = canvas.getBoundingClientRect()
  const scaleX = canvas.width / rect.width
  const scaleY = canvas.height / rect.height
  ctx.beginPath()
  ctx.moveTo((e.clientX - rect.left) * scaleX, (e.clientY - rect.top) * scaleY)
}

function draw(e) {
  if (!isDrawing) return
  const canvas = sigCanvas.value
  const ctx = canvas.getContext('2d')
  const rect = canvas.getBoundingClientRect()
  const scaleX = canvas.width / rect.width
  const scaleY = canvas.height / rect.height
  ctx.lineTo((e.clientX - rect.left) * scaleX, (e.clientY - rect.top) * scaleY)
  ctx.stroke()
}

function stopDrawing() {
  isDrawing = false
}

// رویدادهای لمسی (موبایل و تبلت)
function handleTouchStart(e) {
  if (e.touches.length === 0) return
  const touch = e.touches[0]
  isDrawing = true
  hasDrawn = true
  const canvas = sigCanvas.value
  const ctx = canvas.getContext('2d')
  const rect = canvas.getBoundingClientRect()
  const scaleX = canvas.width / rect.width
  const scaleY = canvas.height / rect.height
  ctx.beginPath()
  ctx.moveTo((touch.clientX - rect.left) * scaleX, (touch.clientY - rect.top) * scaleY)
}

function handleTouchMove(e) {
  if (!isDrawing || e.touches.length === 0) return
  const touch = e.touches[0]
  const canvas = sigCanvas.value
  const ctx = canvas.getContext('2d')
  const rect = canvas.getBoundingClientRect()
  const scaleX = canvas.width / rect.width
  const scaleY = canvas.height / rect.height
  ctx.lineTo((touch.clientX - rect.left) * scaleX, (touch.clientY - rect.top) * scaleY)
  ctx.stroke()
}

// فراخوانی کامند Rust
async function captureSignature(athlete, signatureBase64) {
  const { invoke } = await import('@tauri-apps/api/core')
  return invoke('capture_signature', {
    athleteId: String(athlete.id),
    tournamentId: String(tid.value),
    signatureData: signatureBase64,
  })
}

// تأیید و ذخیره نهایی
async function saveAndSubmitSignature() {
  if (!hasDrawn) {
    notify('لطفاً ابتدا امضا را رسم کنید')
    return
  }
  if (!active.value) return

  const canvas = sigCanvas.value
  const dataUrl = canvas.toDataURL('image/png')
  const id = active.value.id
  signing.value = true

  try {
    const path = await captureSignature(active.value, dataUrl)
    if (!path) return
    const res = store.signWeighIn(tid.value, id, path)
    if (res?.error) {
      modalError.value = res.error
      return
    }
    refreshActive(id)
    closeSignaturePad()
    notify('امضا با موفقیت ثبت شد ✅')
  } catch (e) {
    modalError.value = String(e?.message ?? e)
  } finally {
    signing.value = false
  }
}

function doClearSignature() {
  if (!active.value) return
  const id = active.value.id
  const res = store.clearWeighInSignature(tid.value, id)
  if (res?.error) { modalError.value = res.error; return }
  refreshActive(id)
  notify('امضا حذف شد')
}

/* ---------- ریست ---------- */
const resetTarget = ref(null)
const bracketResetTarget = ref(null)

function hasBracketForCategory(category) {
  return (store.currentTournament?.matches ?? [])
      .some((match) => String(match.weightCategory) === String(category))
}

function requestBracketReset(category) {
  bracketResetTarget.value = {
    category,
    athletes: athletes.value.filter(
        (a) => String(a.weightCategory) === String(category)
    ),
  }
}

function doReset(a) { resetTarget.value = a }

function confirmReset() {
  const a = resetTarget.value
  if (!a) return
  const res = store.resetWeighIn(tid.value, a.id)
  resetTarget.value = null
  if (res?.error) {
    modalError.value = res.error
    notify(res.error)
    return
  }
  if (active.value?.id === a.id) {
    refreshActive(a.id)
    weightInputValue.value = ''
    modalError.value = ''
    nextTick(() => weightInput.value?.focus())
  }
  notify('وزن‌کشی ریست شد')
}

function confirmBracketReset() {
  const target = bracketResetTarget.value
  if (!target) return

  const category = target.category

  // این نام باید با متد واقعی استور هماهنگ باشد.
  const result = store.resetCategoryBracketAndWeighIns(
      tid.value,
      category,
  )

  if (result?.error) {
    bracketResetTarget.value = null
    notify(result.error)
    return
  }

  bracketResetTarget.value = null
  notify(`براکت وزن ${category} حذف و وزن‌کشی ریست شد`)
}

/* ---------- Toast ---------- */
const toast = ref('')
let toastTimer
function notify(msg) {
  toast.value = msg
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = ''), 3500)
}

</script>
