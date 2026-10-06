<template>
  <div class="bg-gradient-to-br from-slate-100 via-white to-blue-50/30 p-4 md:p-6 relative min-h-screen">

    <!-- Toast -->
    <transition-group name="toast" tag="div" class="fixed top-5 left-5 z-50 flex flex-col gap-2">
      <div v-for="toast in toasts" :key="toast.id"
           class="flex items-center gap-2 px-4 py-3 rounded-xl shadow-lg border text-sm font-medium"
           :class="toast.type === 'success'
             ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
             : 'bg-amber-50 border-amber-200 text-amber-800'">
        <span>{{ toast.type === 'success' ? '✅' : '⚠️' }}</span>
        {{ toast.message }}
      </div>
    </transition-group>

    <!-- ===== نوار بالا ===== -->
    <div class="max-w-6xl mx-auto mb-6 space-y-4">

      <!-- پیشرفت -->
      <div v-if="weightSections.length" class="rounded-2xl bg-white border border-slate-200 shadow-sm px-5 py-4">
        <div class="flex items-center justify-between mb-2">
          <h1 class="text-base font-bold text-slate-800">قرعه‌کشی مسابقات</h1>
          <span class="text-sm font-semibold text-slate-500">
            {{ drawnCount }} از {{ weightSections.length }} وزن قرعه‌کشی شد
          </span>
        </div>
        <div class="h-2 w-full rounded-full bg-slate-100 overflow-hidden">
          <div class="h-full rounded-full bg-gradient-to-r from-blue-500 to-indigo-500 transition-all duration-500"
               :style="{ width: `${progressPercent}%` }" />
        </div>
      </div>

      <!-- کنترل‌ها -->
      <div class="flex items-center justify-center gap-3 flex-wrap">
        <!-- نوع قرعه‌کشی -->
        <div class="inline-flex rounded-2xl border border-slate-200 overflow-hidden shadow-sm bg-white text-sm">
          <button @click="drawType = 'random'"
                  class="px-5 py-2.5 font-semibold transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                  :class="drawType === 'random'
                    ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white'
                    : 'text-slate-500 hover:bg-slate-50 hover:text-slate-800'">🎲 رندوم</button>
          <div class="w-px bg-slate-200 my-1.5" />
          <button @click="drawType = 'ranked'"
                  class="px-5 py-2.5 font-semibold transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                  :class="drawType === 'ranked'
                    ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white'
                    : 'text-slate-500 hover:bg-slate-50 hover:text-slate-800'">🏅 رنکینگی</button>
        </div>

        <!-- حالت مراسم -->
        <button @click="toggleCeremony"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-2xl font-semibold text-sm shadow-sm border transition-all focus:outline-none focus-visible:ring-2"
                :class="ceremonyMode
                  ? 'bg-gradient-to-r from-purple-600 to-fuchsia-600 text-white border-transparent focus-visible:ring-purple-500'
                  : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50 focus-visible:ring-slate-400'">
          🎤 {{ ceremonyMode ? 'خروج از حالت مراسم' : 'حالت مراسم (وزن به وزن)' }}
        </button>

        <!-- قرعه‌کشی همه -->
        <button v-if="!ceremonyMode"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-2xl font-semibold text-sm shadow-sm text-white transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:opacity-40 disabled:cursor-not-allowed"
                :class="hasMatches
                  ? 'bg-gradient-to-r from-orange-500 to-amber-500 hover:from-orange-600 hover:to-amber-600'
                  : 'bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700'"
                @click="confirmDrawAll"
                :disabled="isDrawing || !drawableSections.length || sectionsWithoutRanking.length > 0"
                :title="sectionsWithoutRanking.length ? 'برای قرعه‌کشی رنکینگی ابتدا رنک ورزشکاران را ثبت کنید' : ''">
          {{ hasMatches ? '🔄 قرعه‌کشی مجدد همه' : '🎲 قرعه‌کشی همه وزن‌ها' }}
        </button>

        <!-- جابه‌جایی -->
        <button v-if="hasMatches"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-2xl font-semibold text-sm shadow-sm border transition-all focus:outline-none focus-visible:ring-2"
                :class="swapMode
                  ? 'bg-gradient-to-r from-emerald-500 to-teal-500 text-white border-transparent focus-visible:ring-emerald-500'
                  : 'bg-white text-slate-600 border-slate-200 hover:bg-slate-50 focus-visible:ring-slate-400'"
                @click="toggleSwapMode">
          {{ swapMode ? '✔ پایان جابه‌جایی' : '⇄ جابه‌جایی بازیکن‌ها' }}
        </button>

        <!-- زوم -->
        <div v-if="hasMatches" class="inline-flex items-center gap-1 rounded-2xl border border-slate-200 bg-white shadow-sm px-1.5 py-1 text-sm">
          <button @click="setZoom(zoom - 0.1)" class="w-8 h-8 rounded-lg hover:bg-slate-100 text-slate-600 font-bold">−</button>
          <button @click="fitZoom" class="px-2 h-8 rounded-lg hover:bg-slate-100 text-slate-500 text-xs whitespace-nowrap" title="متناسب با صفحه">
            {{ Math.round(zoom * 100) }}٪
          </button>
          <button @click="setZoom(zoom + 0.1)" class="w-8 h-8 rounded-lg hover:bg-slate-100 text-slate-600 font-bold">+</button>
        </div>
      </div>
    </div>

    <!-- راهنمای جابه‌جایی -->
    <transition name="fade">
      <div v-if="swapMode"
           class="text-center text-sm text-amber-800 bg-amber-50 border border-amber-200 rounded-2xl py-3 px-5 mb-6 max-w-lg mx-auto shadow-sm">
        روی دو بازیکن در <b>دور اول</b> (یک وزن) کلیک کنید تا جابه‌جا شوند؛ بازیکنان دارای استراحت هم قابل انتخاب‌اند.
      </div>
    </transition>

    <transition name="fade">
      <div v-if="sectionsWithoutRanking.length"
           class="max-w-3xl mx-auto mb-6 rounded-2xl border border-rose-200 bg-rose-50 px-5 py-3.5 text-sm text-rose-800 shadow-sm">
        <div class="font-bold mb-2">⚠️ برای این وزن‌ها هیچ رنکینگی ثبت نشده است</div>
        <div class="flex flex-wrap gap-1.5 mb-2">
          <button v-for="s in sectionsWithoutRanking" :key="s.key"
                  @click="selectedCategory = s.key"
                  class="rounded-full border border-rose-200 bg-white px-2.5 py-0.5 text-xs font-semibold hover:bg-rose-100">
            وزن {{ s.ordinal }} · {{ s.label }}
          </button>
        </div>
        <div class="text-xs text-rose-600">
          رنک ورزشکاران را در بخش ورزشکاران وارد کنید، یا حالت «🎲 رندوم» را انتخاب کنید.
        </div>
      </div>
    </transition>


    <!-- فیلتر وزن (فقط حالت عادی) -->
    <div v-if="!ceremonyMode && weightSections.length"
         class="flex items-center justify-center gap-2 mb-6 flex-wrap">
      <button @click="selectedCategory = null"
              class="px-4 py-1.5 rounded-full text-sm border font-semibold transition-all"
              :class="selectedCategory === null
                ? 'bg-slate-800 text-white border-slate-800 shadow-sm'
                : 'bg-white text-slate-500 border-slate-200 hover:bg-slate-50'">همه وزن‌ها</button>
      <button v-for="s in weightSections" :key="s.key"
              @click="selectedCategory = s.key"
              class="px-4 py-1.5 rounded-full text-sm border font-semibold transition-all flex items-center gap-1.5"
              :class="selectedCategory === s.key
                ? 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white border-transparent shadow-sm'
                : 'bg-white text-slate-500 border-slate-200 hover:bg-blue-50 hover:text-blue-700'">
        <span>وزن {{ s.ordinal }}</span>
        <span class="opacity-70">·</span>
        <span>{{ s.label }}</span>
        <span class="w-1.5 h-1.5 rounded-full" :class="s.drawn ? 'bg-emerald-400' : 'bg-amber-400'" />
      </button>
    </div>

    <!-- ناوبری حالت مراسم -->
    <div v-if="ceremonyMode && currentSection"
         class="flex items-center justify-between max-w-3xl mx-auto mb-5 gap-3">
      <button @click="ceremonyPrev" :disabled="ceremonyIndex === 0"
              class="px-4 py-2 rounded-xl bg-white border border-slate-200 text-sm font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed">
        → وزن قبلی
      </button>
      <div class="text-sm font-bold text-slate-700">
        وزن {{ currentSection.ordinal }} ({{ currentSection.label }}) — {{ ceremonyIndex + 1 }} از {{ weightSections.length }}
      </div>
      <button @click="ceremonyNext" :disabled="ceremonyIndex >= weightSections.length - 1"
              class="px-4 py-2 rounded-xl bg-white border border-slate-200 text-sm font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed">
        وزن بعدی ←
      </button>
    </div>

    <!-- حالت خالی -->
    <div v-if="!weightSections.length" class="text-center text-slate-400 py-24">
      <div class="text-5xl mb-4 opacity-60">🥋</div>
      <p class="text-base font-medium">ابتدا در بخش ورزشکاران، شرکت‌کننده‌ها را ثبت کنید.</p>
    </div>

    <!-- ===== کارت هر وزن ===== -->
    <transition-group name="bracket-reveal" tag="div" class="max-w-full">
      <div v-for="(section, index) in sectionsToRender" :key="section.key"
           class="mb-6 rounded-3xl bg-white border border-slate-100 shadow-sm overflow-hidden max-w-6xl mx-auto"
           :style="{ animationDelay: `${index * 60}ms` }">

        <!-- هدر وزن -->
        <div class="flex items-center justify-between px-5 py-3 bg-gradient-to-r from-slate-50 to-blue-50/50 border-b border-slate-100 gap-3 flex-wrap">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="bg-gradient-to-r from-blue-600 to-indigo-600 text-white px-4 py-1 rounded-full text-xs font-bold shadow-sm">
              وزن {{ section.ordinal }}
            </span>
            <span class="bg-white border border-slate-200 text-slate-700 px-3 py-1 rounded-full text-xs font-bold">
              {{ section.label }}
            </span>
            <span class="text-xs font-semibold text-slate-500">{{ section.roster.length }} ورزشکار</span>
            <span v-if="section.byes > 0" class="text-xs text-slate-400">({{ section.byes }} استراحت)</span>
            <span class="text-[11px] font-bold px-2 py-0.5 rounded-full"
                  :class="section.drawn ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'">
              {{ section.drawn ? 'قرعه‌کشی شد' : 'در انتظار قرعه‌کشی' }}
            </span>
          </div>

          <button v-if="section.drawn"
                  @click="confirmDrawCategory(section)"
                  class="text-xs bg-white hover:bg-blue-50 text-slate-500 hover:text-blue-600 px-3 py-1.5 rounded-full border border-slate-200 hover:border-blue-200 font-medium">
            🔄 قرعه‌کشی مجدد
          </button>
        </div>

        <!-- لیست ورزشکاران (قبل از قرعه‌کشی) -->
        <div v-if="!section.drawn" class="p-5">
          <p class="text-sm text-slate-500 mb-4 text-center">
            لیست ورزشکاران این وزن را بررسی و تأیید کنید، سپس قرعه‌کشی را انجام دهید.
          </p>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 mb-5">
            <div v-for="(a, i) in section.roster" :key="a.id"
                 class="flex items-center gap-2 px-3 py-2 rounded-xl border text-sm"
                 :class="drawType === 'ranked' && section.seedIds.has(a.id)
                   ? 'border-amber-300 bg-amber-50'
                   : 'border-slate-100 bg-slate-50'">
              <span class="w-6 h-6 shrink-0 rounded-full bg-white border border-slate-200 text-[11px] font-bold text-slate-500 flex items-center justify-center">{{ i + 1 }}</span>
              <div class="min-w-0 flex-1">
                <div class="font-medium text-slate-800 truncate">{{ a.name }}</div>
                <div v-if="a.club" class="text-xs text-slate-500 truncate">{{ a.club }}</div>
              </div>
              <span v-if="a.ranking != null"
                    class="shrink-0 text-[10px] font-bold rounded px-1.5 py-0.5" :class="rankBadgeClass(a.ranking)">
                رنک {{ a.ranking }}
              </span>
              <span v-if="drawType === 'ranked' && section.seedIds.has(a.id)"
                    class="shrink-0 text-amber-500" title="سید">★</span>
            </div>
          </div>

          <div class="flex justify-center">
            <button @click="handleDrawCategory(section)"
                    :disabled="!canDraw(section)"
                    class="inline-flex items-center gap-2 px-6 py-2.5 rounded-2xl font-bold text-sm text-white shadow-sm bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all">
              ✅ تأیید لیست و قرعه‌کشی این وزن
            </button>
          </div>
          <p v-if="section.roster.length < 2" class="text-center text-xs text-amber-600 mt-2">
            برای قرعه‌کشی حداقل ۲ ورزشکار لازم است.
          </p>
          <p v-else-if="needsRanking(section)" class="text-center text-xs font-medium text-rose-600 mt-2">
            قرعه‌کشی رنکینگی نیاز به حداقل یک ورزشکار رنک‌دار دارد. رنک‌ها را ثبت کنید یا حالت رندوم را انتخاب کنید.
          </p>
        </div>

        <!-- براکت (بعد از قرعه‌کشی) -->
        <div v-else class="overflow-x-auto pb-3 pt-4 px-3">
          <div class="w-max mx-auto" :style="{ zoom }">
            <div class="bracket-canvas flex items-stretch flex-nowrap">
              <template v-for="round in leftRounds(section.key)" :key="'L' + round">
                <div class="flex flex-col min-w-[132px]">
                  <div class="text-center text-[10px] text-slate-400 font-bold mb-2 uppercase tracking-wider">
                    {{ roundLabel(section.key, round) }}
                  </div>
                  <div class="bracket-col bracket-col--left" :class="{ 'is-first': round === 1 }"
                       :style="columnHeightStyle(section.key)">
                    <div class="bracket-slot" v-for="m in matchesBySideRound(section.key, 'left', round)" :key="m.id">
                      <div class="match-wrap">
                        <MatchCard :match="m" :swap-mode="swapMode" :selected-slot="selectedSlot" @select-slot="onSelectSlot" />
                      </div>
                    </div>
                  </div>
                </div>
              </template>

              <!-- فینال -->
              <div class="flex flex-col items-center min-w-[140px] px-1">
                <div class="text-center text-[10px] text-amber-600 font-bold mb-2">🏆 فینال</div>
                <div class="bracket-final flex w-full flex-col items-center justify-center"
                     :style="finalColumnStyle(section.key)">
                  <div class="w-full shrink-0 rounded-2xl border border-amber-100 bg-gradient-to-b from-amber-50 to-orange-50/50 p-2">
                    <div class="flex flex-col gap-2">
                      <MatchCard v-for="m in finalMatches(section.key)" :key="m.id" :match="m"
                                 :swap-mode="swapMode && finalRound(section.key) === 1"
                                 :selected-slot="selectedSlot" @select-slot="onSelectSlot" />
                    </div>
                  </div>
                </div>
              </div>

              <template v-for="round in rightRounds(section.key)" :key="'R' + round">
                <div class="flex flex-col min-w-[132px]">
                  <div class="text-center text-[10px] text-slate-400 font-bold mb-2 uppercase tracking-wider">
                    {{ roundLabel(section.key, round) }}
                  </div>
                  <div class="bracket-col bracket-col--right" :class="{ 'is-first': round === 1 }"
                       :style="columnHeightStyle(section.key)">
                    <div class="bracket-slot" v-for="m in matchesBySideRound(section.key, 'right', round)" :key="m.id">
                      <div class="match-wrap">
                        <MatchCard :match="m" :swap-mode="swapMode" :selected-slot="selectedSlot" @select-slot="onSelectSlot" />
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </transition-group>

    <!-- Overlay قرعه‌کشی -->
    <transition name="draw-overlay">
      <div v-if="isDrawing" class="fixed inset-0 z-50 flex flex-col items-center justify-center"
           style="background: radial-gradient(ellipse at center, rgba(15,23,42,0.97) 0%, rgba(2,6,23,0.99) 100%)">
        <div class="absolute inset-0 overflow-hidden pointer-events-none">
          <div v-for="i in 20" :key="i" class="particle" :style="particleStyle(i)" />
        </div>
        <div class="relative flex flex-col items-center gap-8 px-8 text-center">
          <div class="relative">
            <div class="w-28 h-28 rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center shadow-2xl draw-spin">
              <span class="text-5xl">🎲</span>
            </div>
            <div class="absolute inset-0 rounded-full border-4 border-blue-400/30 draw-ping" />
            <div class="absolute inset-[-8px] rounded-full border-2 border-indigo-500/20 draw-ping" style="animation-delay: 0.3s" />
          </div>
          <div class="flex flex-col items-center gap-3">
            <h2 class="text-3xl font-black text-white tracking-wide draw-text-glow">{{ drawingLabel }}</h2>
            <p class="text-blue-300 text-sm font-medium animate-pulse">در حال انجام قرعه‌کشی...</p>
          </div>
          <div class="w-64 h-1.5 bg-white/10 rounded-full overflow-hidden">
            <div class="h-full bg-gradient-to-r from-blue-400 to-indigo-400 rounded-full draw-progress" />
          </div>
        </div>
      </div>
    </transition>
    <!-- ===== نمایش متمرکز یک وزن (Focus) ===== -->
    <Teleport to="body">
      <transition name="fade">
        <div v-if="focusedSection" dir="rtl" class="fixed inset-0 z-[70] bg-white flex flex-col">

          <!-- هدر -->
          <div class="flex items-center justify-between px-6 py-3 border-b border-slate-100 bg-gradient-to-r from-slate-50 to-blue-50/50 gap-3 flex-wrap">
            <div class="flex items-center gap-2 flex-wrap">
          <span class="bg-gradient-to-r from-blue-600 to-indigo-600 text-white px-4 py-1 rounded-full text-xs font-bold shadow-sm">
            وزن {{ focusedSection.ordinal }}
          </span>
              <span class="bg-white border border-slate-200 text-slate-700 px-3 py-1 rounded-full text-xs font-bold">
            {{ focusedSection.label }}
          </span>
              <span class="text-xs font-semibold text-slate-500">{{ focusedSection.roster.length }} ورزشکار</span>
            </div>

            <div class="flex items-center gap-2">
              <div class="inline-flex items-center gap-1 rounded-2xl border border-slate-200 bg-white shadow-sm px-1.5 py-1 text-sm">
                <button @click="setZoom(zoom - 0.1)" class="w-8 h-8 rounded-lg hover:bg-slate-100 text-slate-600 font-bold">−</button>
                <button @click="fitZoom" class="px-2 h-8 rounded-lg hover:bg-slate-100 text-slate-500 text-xs whitespace-nowrap">
                  {{ Math.round(zoom * 100) }}٪
                </button>
                <button @click="setZoom(zoom + 0.1)" class="w-8 h-8 rounded-lg hover:bg-slate-100 text-slate-600 font-bold">+</button>
              </div>
              <button @click="closeFocus"
                      class="px-4 py-2 rounded-2xl font-semibold text-sm shadow-sm bg-white border border-slate-200 text-slate-700 hover:bg-slate-50">
                ✕ خروج (Esc)
              </button>
            </div>
          </div>

          <!-- بدنه‌ی براکت -->
          <div class="flex-1 overflow-auto p-6">
            <div class="w-max mx-auto" :style="{ zoom }">
              <div class="bracket-canvas flex items-stretch flex-nowrap">
                <template v-for="round in leftRounds(focusedSection.key)" :key="'FL' + round">
                  <div class="flex flex-col min-w-[132px]">
                    <div class="text-center text-[10px] text-slate-400 font-bold mb-2 uppercase tracking-wider">
                      {{ roundLabel(focusedSection.key, round) }}
                    </div>
                    <div class="bracket-col bracket-col--left" :class="{ 'is-first': round === 1 }"
                         :style="columnHeightStyle(focusedSection.key)">
                      <div class="bracket-slot" v-for="m in matchesBySideRound(focusedSection.key, 'left', round)" :key="m.id">
                        <div class="match-wrap">
                          <MatchCard :match="m" :swap-mode="swapMode" :selected-slot="selectedSlot" @select-slot="onSelectSlot" />
                        </div>
                      </div>
                    </div>
                  </div>
                </template>

                <div class="flex flex-col items-center min-w-[140px] px-1">
                  <div class="text-center text-[10px] text-amber-600 font-bold mb-2">🏆 فینال</div>
                  <div class="bracket-final flex w-full flex-col items-center justify-center"
                       :style="finalColumnStyle(focusedSection.key)">
                    <div class="w-full shrink-0 rounded-2xl border border-amber-100 bg-gradient-to-b from-amber-50 to-orange-50/50 p-2">
                      <div class="flex flex-col gap-2">
                        <MatchCard v-for="m in finalMatches(focusedSection.key)" :key="m.id" :match="m"
                                   :swap-mode="swapMode && finalRound(focusedSection.key) === 1"
                                   :selected-slot="selectedSlot" @select-slot="onSelectSlot" />
                      </div>
                    </div>
                  </div>
                </div>

                <template v-for="round in rightRounds(focusedSection.key)" :key="'FR' + round">
                  <div class="flex flex-col min-w-[132px]">
                    <div class="text-center text-[10px] text-slate-400 font-bold mb-2 uppercase tracking-wider">
                      {{ roundLabel(focusedSection.key, round) }}
                    </div>
                    <div class="bracket-col bracket-col--right" :class="{ 'is-first': round === 1 }"
                         :style="columnHeightStyle(focusedSection.key)">
                      <div class="bracket-slot" v-for="m in matchesBySideRound(focusedSection.key, 'right', round)" :key="m.id">
                        <div class="match-wrap">
                          <MatchCard :match="m" :swap-mode="swapMode" :selected-slot="selectedSlot" @select-slot="onSelectSlot" />
                        </div>
                      </div>
                    </div>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useTournamentStore } from '../stores/tournament'
import MatchCard from './MatchCard.vue'
import { WEIGHT_CATEGORIES } from '../data/categories'
import {getCurrentWindow} from "@tauri-apps/api/window";

const store = useTournamentStore()

/* ==================== helpers: نرمال‌سازی و شماره وزن ==================== */

const FA_DIGITS = '۰۱۲۳۴۵۶۷۸۹'
const AR_DIGITS = '٠١٢٣٤٥٦٧٨٩'

const toFaDigits = (v: string | number) =>
    String(v).replace(/\d/g, d => FA_DIGITS[Number(d)])

const FA_ORDINALS = [
  'اول', 'دوم', 'سوم', 'چهارم', 'پنجم', 'ششم', 'هفتم', 'هشتم', 'نهم', 'دهم',
  'یازدهم', 'دوازدهم', 'سیزدهم', 'چهاردهم', 'پانزدهم', 'شانزدهم', 'هفدهم', 'هجدهم', 'نوزدهم', 'بیستم',
] as const

const ordinalFa = (n: number) =>
    (n >= 1 && n <= FA_ORDINALS.length) ? FA_ORDINALS[n - 1] : toFaDigits(n)

/**
 * هر فرمتی از وزن را به یک کلید یکتا تبدیل می‌کند.
 *   '-۴۵ کیلوگرم' | '45-' | 'زیر 45' -> '-45'
 *   '+۸۷'         | 'بالای ۸۷'       -> '+87'
 *   '45-48'                          -> '45-48'
 */
function normCat(s: unknown): string {
  const raw = String(s ?? '')
      .replace(/[\u200c\u200f\u200e\s]/g, '')
      .replace(/[۰-۹]/g, d => String(FA_DIGITS.indexOf(d)))
      .replace(/[٠-٩]/g, d => String(AR_DIGITS.indexOf(d)))
      .replace(/[−–—ـ]/g, '-')
      .replace(/کیلوگرم|کیلو|kg/gi, '')
      .toLowerCase()

  const nums = raw.match(/\d+(?:[.,]\d+)?/g)
  if (!nums) return raw

  const num = (i: number) => parseFloat(nums[i].replace(',', '.'))
  if (nums.length >= 2) return `${num(0)}-${num(1)}`

  const isPlus = raw.includes('+') || /بالا|بیشتر|over|plus/i.test(raw)
  return `${isPlus ? '+' : '-'}${num(0)}`
}

/** مقدار عددی برای مرتب‌سازی وقتی وزن در لیست رسمی پیدا نشد */
function weightSortValue(cat: string) {
  const s = normCat(cat)
  const m = s.match(/(\d+(?:\.\d+)?)/)
  const n = m ? parseFloat(m[1]) : 0
  return s.startsWith('+') ? n + 0.5 : n
}

/* ==================== لیست رسمی وزن‌ها ==================== */

/** آرایه‌ی خام (رشته یا آبجکت) را به آرایه‌ی رشته تبدیل می‌کند */
function toCatList(v: unknown): string[] | null {
  if (!Array.isArray(v)) return null
  const out = v.map(x => {
    if (x == null) return ''
    if (typeof x === 'string' || typeof x === 'number') return String(x)
    const o = x as Record<string, unknown>
    return String(o.label ?? o.name ?? o.title ?? o.value ?? o.weight ?? o.cat ?? '')
  }).filter(Boolean)
  return out.length ? out : null
}

/** کلید آبجکت را بدون حساسیت به بزرگی/کوچکی پیدا می‌کند */
function findKey(obj: Record<string, unknown>, key: string): string | null {
  if (key in obj) return key
  const lc = key.toLowerCase()
  return Object.keys(obj).find(k => k.toLowerCase() === lc) ?? null
}

const GENDER_ALIASES: Record<string, string[]> = {
  male: ['male', 'men', 'man', 'boys', 'boy', 'آقایان', 'مردان', 'پسران', 'مرد', 'پسر'],
  female: ['female', 'women', 'woman', 'girls', 'girl', 'بانوان', 'زنان', 'دختران', 'زن', 'دختر'],
}

function expandGender(g?: string): string[] {
  if (!g) return []
  const lc = String(g).toLowerCase()
  for (const list of Object.values(GENDER_ALIASES)) {
    if (list.some(x => x.toLowerCase() === lc)) return list
  }
  return [g]
}

/**
 * ساختار WEIGHT_CATEGORIES هرچه باشد (تخت، [ageGroup][gender]، یا [gender][ageGroup])
 * لیست وزن‌ها را پیدا می‌کند.
 */
function resolveList(node: unknown, keys: string[], depth = 0): string[] | null {
  if (node == null || depth > 4) return null

  const direct = toCatList(node)
  if (direct) return direct
  if (typeof node !== 'object') return null

  const obj = node as Record<string, unknown>

  // اول با کلیدهای مشخص (رده سنی / جنسیت)
  for (const k of keys) {
    const real = findKey(obj, k)
    if (real) {
      const r = resolveList(obj[real], keys, depth + 1)
      if (r) return r
    }
  }

  // اگر فقط یک شاخه دارد، همان را دنبال کن
  const values = Object.values(obj)
  if (values.length === 1) return resolveList(values[0], keys, depth + 1)

  return null
}

const officialCats = computed<string[]>(() => {
  const t = store.currentTournament as unknown as Record<string, unknown> | null
  if (!t) return []

  const ageKeys = [t.ageGroup, t.age_group, t.category, t.ageCategory]
      .filter(Boolean).map(String)
  const genderKeys = [...expandGender(t.gender as string), ...expandGender(t.sex as string)]

  return resolveList(WEIGHT_CATEGORIES, [...ageKeys, ...genderKeys]) ?? []
})

/** نگاشت کلیدِ نرمال‌شده -> { index, label } از لیست رسمی */
const officialMap = computed(() => {
  const map = new Map<string, { index: number; label: string }>()
  officialCats.value.forEach((c, i) => {
    const k = normCat(c)
    if (!map.has(k)) map.set(k, { index: i, label: c })
  })
  return map
})

/* ==================== state ==================== */

const swapMode = ref(false)
const selectedSlot = ref<{ matchId: string; slot: 1 | 2 } | null>(null)
const selectedCategory = ref<string | null>(null)   // با normCat ذخیره می‌شود
const drawType = ref<'random' | 'ranked'>('random')
const isDrawing = ref(false)
const drawingLabel = ref('')
const zoom = ref(1)

const ceremonyMode = ref(false)
const ceremonyIndex = ref(0)

interface Toast { id: number; message: string; type: 'success' | 'warning' }
const toasts = ref<Toast[]>([])
let toastId = 0
function showToast(message: string, type: 'success' | 'warning' = 'success') {
  const id = toastId++
  toasts.value.push({ id, message, type })
  setTimeout(() => { toasts.value = toasts.value.filter(t => t.id !== id) }, 4000)
}

const athletesAll = computed<any[]>(() => store.currentTournament?.athletes ?? [])
const eligibleAthletes = computed(() =>
    athletesAll.value.filter(
        athlete => athlete?.weighIn?.status === 'passed'
    )
)
const allMatches = computed<any[]>(() => store.currentTournament?.matches ?? [])
const hasMatches = computed(() => allMatches.value.length > 0)

function nextPow2(n: number) { let p = 1; while (p < n) p <<= 1; return p }

function rankOf(a: any): number | null {
  const v = a?.ranking
  if (v === null || v === undefined || v === '') return null
  const n = Number(v)
  return Number.isFinite(n) && n > 0 ? n : null
}
function hasRank(a: any) { return rankOf(a) !== null }

/* ==================== سکشن‌های وزنی ==================== */

interface WeightSection {
  key: string        // کلید نرمال‌شده (مبنای همه مقایسه‌ها)
  cat: string        // مقدار خام دیتابیس (برای فراخوانی store)
  label: string      // برچسب نمایشی (ترجیحاً از لیست رسمی)
  roster: any[]
  drawn: boolean
  seedIds: Set<string>
  rankedCount: number
  byes: number
  order: number | null
  ordinal: string
}

const weightSections = computed<WeightSection[]>(() => {
  // ۱) گروه‌بندی ورزشکارها بر اساس کلید نرمال‌شده (نه رشته خام)
  const groups = new Map<string, { rawCat: string; roster: any[] }>()
  for (const a of eligibleAthletes.value) {
    if (!a?.weightCategory) continue

    const key = normCat(a.weightCategory)

    if (!groups.has(key)) {
      groups.set(key, {
        rawCat: String(a.weightCategory),
        roster: [],
      })
    }

    groups.get(key)!.roster.push(a)
  }


  // ۲) وزن‌هایی که مسابقه دارند ولی ورزشکارشان حذف شده هم نمایش داده شوند
  for (const m of allMatches.value) {
    if (!m?.weightCategory) continue
    const key = normCat(m.weightCategory)
    if (!groups.has(key)) groups.set(key, { rawCat: String(m.weightCategory), roster: [] })
  }

  const drawnKeys = new Set(allMatches.value.map(m => normCat(m.weightCategory)))

  // ۳) ساخت سکشن‌ها
  const sections = [...groups.entries()].map(([key, g]) => {
    const official = officialMap.value.get(key)

    const roster = [...g.roster].sort(
        (a, b) => (rankOf(a) ?? 9999) - (rankOf(b) ?? 9999)
    )
    const ranked = roster.filter(hasRank)
    const seedIds = new Set<string>(ranked.slice(0, 4).map(a => a.id))

    return {
      key,
      cat: g.rawCat,
      label: official?.label ?? g.rawCat,
      roster,
      drawn: drawnKeys.has(key),
      seedIds,
      rankedCount: ranked.length,
      byes: roster.length ? nextPow2(roster.length) - roster.length : 0,
      order: official ? official.index + 1 : null,
      ordinal: '',
    } as WeightSection
  })

  // ۴) مرتب‌سازی: اول ترتیب رسمی، بعد ترتیب عددی برای وزن‌های ناشناخته
  sections.sort((a, b) => {
    const ao = a.order ?? Infinity
    const bo = b.order ?? Infinity
    return (ao - bo) || (weightSortValue(a.key) - weightSortValue(b.key))
  })

  // ۵) شماره‌گذاری: اگر در لیست رسمی بود شماره‌ی رسمی، وگرنه ترتیب نمایش
  sections.forEach((s, i) => {
    s.ordinal = ordinalFa(s.order ?? (i + 1))
  })

  return sections
})

const drawnCount = computed(() => weightSections.value.filter(s => s.drawn).length)
const progressPercent = computed(() =>
    weightSections.value.length ? (drawnCount.value / weightSections.value.length) * 100 : 0
)

/* ==================== اعتبارسنجی قرعه‌کشی رنکینگی ==================== */

const drawableSections = computed(() => weightSections.value.filter(s => s.roster.length >= 2))

/** وزن‌هایی که در حالت رنکینگی هیچ ورزشکار رنک‌داری ندارند */
const sectionsWithoutRanking = computed(() =>
    drawType.value === 'ranked'
        ? drawableSections.value.filter(s => s.rankedCount === 0)
        : []
)

function needsRanking(section: WeightSection) {
  return drawType.value === 'ranked' && section.roster.length >= 2 && section.rankedCount === 0
}

function canDraw(section: WeightSection) {
  return section.roster.length >= 2 && !needsRanking(section)
}

const sectionName = (s: WeightSection) => `وزن ${s.ordinal} (${s.label})`

/**
 * قبل از هر قرعه‌کشی رنکینگی صدا زده می‌شود.
 * نبودِ رنکینگ = توقف کامل، کم بودنِ رنکینگ = تأیید کاربر.
 */
function guardRanked(sections: WeightSection[]): boolean {
  if (drawType.value !== 'ranked') return true

  const targets = sections.filter(s => s.roster.length >= 2)

  const missing = targets.filter(s => s.rankedCount === 0)
  if (missing.length) {
    showToast(
        `قرعه‌کشی رنکینگی ممکن نیست: در ${missing.map(sectionName).join('، ')} هیچ رنکینگی ثبت نشده است.`,
        'warning'
    )
    return false
  }

  // رنکینگ هست ولی انقدر کم که نتیجه عملاً رندوم می‌شود
  const weak = targets.filter(s => s.roster.length >= 4 && s.rankedCount < 2)
  if (weak.length) {
    const list = weak.map(sectionName).join('، ')
    if (!confirm(
        `در ${list} کمتر از ۲ ورزشکار رنک‌دار وجود دارد و سیدبندی معنادار انجام نمی‌شود.\n\nبا همین شرایط ادامه می‌دهید؟`
    )) return false
  }

  return true
}

const currentSection = computed(() => weightSections.value[ceremonyIndex.value] ?? null)

const sectionsToRender = computed<WeightSection[]>(() => {
  if (ceremonyMode.value) return currentSection.value ? [currentSection.value] : []
  if (selectedCategory.value) return weightSections.value.filter(s => s.key === selectedCategory.value)
  return weightSections.value
})

// اگر لیست وزن‌ها عوض شد، ایندکس‌ها و فیلتر نامعتبر نشوند
watch(weightSections, list => {
  if (ceremonyIndex.value > list.length - 1) ceremonyIndex.value = Math.max(0, list.length - 1)
  if (selectedCategory.value && !list.some(s => s.key === selectedCategory.value)) {
    selectedCategory.value = null
  }
})

/* ==================== براکت (همه با کلید نرمال‌شده) ==================== */

const matchesByKey = computed(() => {
  const map = new Map<string, any[]>()
  for (const m of allMatches.value) {
    const k = normCat(m.weightCategory)
    const arr = map.get(k)
    if (arr) arr.push(m)
    else map.set(k, [m])
  }
  // ترتیب پایدار داخل هر وزن
  for (const arr of map.values()) {
    arr.sort((a, b) =>
        (a.round ?? 0) - (b.round ?? 0) ||
        (a.position ?? a.matchNumber ?? a.order ?? 0) - (b.position ?? b.matchNumber ?? b.order ?? 0)
    )
  }
  return map
})

function catMatches(key: string) { return matchesByKey.value.get(normCat(key)) ?? [] }

const MIRROR_SIDES = true

function matchesBySideRound(key: string, side: 'left' | 'right', round: number) {
  const target = MIRROR_SIDES ? (side === 'left' ? 'right' : 'left') : side
  return catMatches(key)
      .filter(m => m.round === round && m.side === target)
      .sort((a, b) => (a.bracketIndex ?? a.order) - (b.bracketIndex ?? b.order))
}

function finalRound(key: string) {
  const rounds = catMatches(key).map(m => m.round ?? 0)
  return rounds.length ? Math.max(...rounds) : 0
}

function finalMatches(key: string) {
  const max = finalRound(key)
  if (!max) return []
  const list = catMatches(key)
  const tagged = list.filter(m => m.round === max && m.side === 'final')
  return tagged.length ? tagged : list.filter(m => m.round === max)
}

function leftRounds(key: string) {
  const max = finalRound(key)
  return Array.from({ length: Math.max(0, max - 1) }, (_, i) => i + 1)
}
function rightRounds(key: string) { return [...leftRounds(key)].reverse() }

function roundLabel(key: string, round: number) {
  const max = finalRound(key)
  switch (max - round) {
    case 1: return 'نیمه‌نهایی'
    case 2: return 'یک‌چهارم نهایی'
    case 3: return 'یک‌هشتم نهایی'
    case 4: return 'یک‌شانزدهم نهایی'
    case 5: return 'یک‌سی‌ودوم نهایی'
    default: return round === 1 ? 'دور اول' : `دور ${ordinalFa(round)}`
  }
}

const MATCH_CARD_HEIGHT = 76
const MATCH_GAP = 28

function maxMatchesInRound1(key: string) {
  return Math.max(
      matchesBySideRound(key, 'left', 1).length,
      matchesBySideRound(key, 'right', 1).length,
      1
  )
}
function columnHeightStyle(key: string) {
  const n = maxMatchesInRound1(key)
  return { height: `${n * MATCH_CARD_HEIGHT + (n - 1) * MATCH_GAP}px` }
}

function rankBadgeClass(rank: number) {
  if (rank === 1) return 'bg-yellow-400 text-white'
  if (rank === 2) return 'bg-gray-700 text-gray-100'
  if (rank === 3 || rank === 4) return 'bg-amber-700 text-white'
  return 'bg-slate-200 text-slate-600'
}

/* ==================== زوم ==================== */

function setZoom(v: number) { zoom.value = Math.min(1.2, Math.max(0.4, Math.round(v * 10) / 10)) }

function fitZoom() {
  const inFocus = focusedKey.value !== null
  const sections = inFocus
      ? (focusedSection.value ? [focusedSection.value] : [])
      : sectionsToRender.value.filter(s => s.drawn)
  if (!sections.length) { zoom.value = 1; return }

  const COL_W = 132
  const GAP_W = 48                       // مطابق --bgap
  const cols = Math.max(1, Math.max(...sections.map(s => 2 * (finalRound(s.key) - 1) + 1)))
  const contentW = cols * COL_W + (cols - 1) * GAP_W

  const avail = inFocus
      ? window.innerWidth - 48            // p-6 دو طرف
      : Math.min(window.innerWidth, 1152) - 48

  zoom.value = Math.min(1, Math.max(0.4, Math.round((avail / contentW) * 10) / 10))
}

/* ==================== Overlay ==================== */

function particleStyle(i: number) {
  const size = 2 + (i % 4)
  return {
    position: 'absolute' as const, width: `${size}px`, height: `${size}px`, borderRadius: '50%',
    background: i % 3 === 0 ? '#818cf8' : i % 3 === 1 ? '#60a5fa' : '#a78bfa',
    left: `${(i * 37 + 11) % 100}%`, bottom: '-10px', opacity: '0.6',
    animation: `floatUp ${3 + (i % 3)}s ${(i * 0.3) % 3}s infinite ease-in`,
  }
}

async function runWithCinematic(label: string, action: () => void) {
  drawingLabel.value = label
  isDrawing.value = true
  await new Promise(r => setTimeout(r, 1800))
  action()
  await new Promise(r => setTimeout(r, 400))
  isDrawing.value = false
}

/* ==================== اکشن‌های قرعه‌کشی ==================== */

function confirmDrawAll() {
  if (!guardRanked(weightSections.value)) return
  if (hasMatches.value && !confirm('قرعه‌کشی مجدد کل جدول؟ تمام نتایج و براکت‌ها بازنشانی می‌شود.')) return
  handleDrawAll()
}

async function handleDrawAll() {
  const id = store.currentTournamentId
  if (!id) return
  await runWithCinematic('قرعه‌کشی همه وزن‌ها', () => {
    const outcome = store.drawBracket(id, drawType.value)
    if (outcome?.error) { alert(outcome.error); return }
    swapMode.value = false
    selectedSlot.value = null
  })
  fitZoom()
  showToast('قرعه‌کشی همه وزن‌ها انجام شد.')
}

function confirmDrawCategory(section: WeightSection) {
  if (!guardRanked([section])) return
  if (!confirm(`قرعه‌کشی مجدد وزن ${section.ordinal} (${section.label})؟ نتایج فعلی این وزن حذف می‌شود.`)) return
  handleDrawCategoryInternal(section)
}

async function handleDrawCategory(section: WeightSection) {
  if (!guardRanked([section])) return
  await handleDrawCategoryInternal(section)
}

/** بدون هیچ اعتبارسنجی — فقط بعد از عبور از guardRanked صدا زده شود */
async function handleDrawCategoryInternal(section: WeightSection) {
  const id = store.currentTournamentId
  if (!id) return
  await runWithCinematic(`قرعه‌کشی وزن ${section.ordinal}  ${section.label}`, () => {
    const outcome = store.drawBracketForCategory(id, section.cat, drawType.value)
    if (outcome?.error) { alert(outcome.error); return }
  })
  fitZoom()
  showToast(`قرعه‌کشی وزن ${section.ordinal} (${section.label}) انجام شد.`)
  await openFocus(section.key)
}


/* ==================== حالت مراسم ==================== */
function ceremonyPrev() { if (ceremonyIndex.value > 0) ceremonyIndex.value-- }
function ceremonyNext() { if (ceremonyIndex.value < weightSections.value.length - 1) ceremonyIndex.value++ }

/* ==================== جابه‌جایی ==================== */

function toggleSwapMode() { swapMode.value = !swapMode.value; selectedSlot.value = null }

function onSelectSlot(payload: { matchId: string; slot: 1 | 2 }) {
  if (!selectedSlot.value) { selectedSlot.value = payload; return }
  if (selectedSlot.value.matchId === payload.matchId && selectedSlot.value.slot === payload.slot) {
    selectedSlot.value = null; return
  }

  const matchA = allMatches.value.find(m => m.id === selectedSlot.value!.matchId)
  const matchB = allMatches.value.find(m => m.id === payload.matchId)

  // مقایسه با normCat تا فرمت متفاوت وزن مانع جابه‌جایی نشود
  if (!matchA || !matchB || normCat(matchA.weightCategory) !== normCat(matchB.weightCategory)) {
    showToast('جابه‌جایی فقط بین دو بازیکن از یک وزن ممکن است.', 'warning')
    selectedSlot.value = null; return
  }

  const athleteMap = new Map<string, any>(athletesAll.value.map(x => [x.id, x]))
  const getClub = (id: string | null | undefined) => id ? (athleteMap.get(id)?.club ?? null) : null

  const keyA = selectedSlot.value.slot === 1 ? 'athlete1Id' : 'athlete2Id'
  const keyB = payload.slot === 1 ? 'athlete1Id' : 'athlete2Id'
  const valA = matchA[keyA]
  const valB = matchB[keyB]
  const otherA = keyA === 'athlete1Id' ? matchA.athlete2Id : matchA.athlete1Id
  const otherB = keyB === 'athlete1Id' ? matchB.athlete2Id : matchB.athlete1Id

  if (valA && valB) {
    const clubA = getClub(valA)
    const clubB = getClub(valB)
    if ((clubB != null && clubB === getClub(otherA)) || (clubA != null && clubA === getClub(otherB))) {
      showToast('این جابه‌جایی باعث تقابل دو هم‌باشگاهی در دور اول می‌شود!', 'warning')
      selectedSlot.value = null; return
    }
  }

  store.swapAthletes(selectedSlot.value, payload)
  selectedSlot.value = null
  showToast('جابه‌جایی انجام شد.')
}

function finalColumnStyle(key: string) {
  const s = columnHeightStyle(key) as Record<string, any>
  const h = s?.height ?? s?.minHeight
  return h ? { minHeight: h } : {}
}

/* ==================== فول‌اسکرین حالت مراسم ==================== */

async function enterNativeFullscreen() {
  try { if (!document.fullscreenElement) await getCurrentWindow().setFullscreen(true) } catch {}
}
async function exitNativeFullscreen() {
  try { if (document.fullscreenElement) await document.exitFullscreen() } catch {}
}

async function toggleCeremony() {
  ceremonyMode.value = !ceremonyMode.value

  if (ceremonyMode.value) {
    selectedCategory.value = null
    swapMode.value = false
    selectedSlot.value = null
    const firstPending = weightSections.value.findIndex(s => !s.drawn)
    ceremonyIndex.value = firstPending === -1 ? 0 : firstPending

    await enterNativeFullscreen()
    requestAnimationFrame(fitZoom)
  } else {
    document.documentElement.classList.remove('overflow-hidden')
    await exitNativeFullscreen()
  }
}

/* Esc در حالت fullscreen توسط خود مرورگر مصرف می‌شود، پس با fullscreenchange سینک می‌کنیم */
function onFsChange() {
  if (document.fullscreenElement) return
  // اگر کاربر با Esc از fullscreen مرورگر خارج شد، stateها را هم برگردان
  if (focusedKey.value) {
    focusedKey.value = null
    document.documentElement.classList.remove('overflow-hidden')
  }
  if (ceremonyMode.value) {
    ceremonyMode.value = false
    document.documentElement.classList.remove('overflow-hidden')
  }
}

function onCeremonyKey(e: KeyboardEvent) {
  if (focusedKey.value) {
    if (e.key === 'Escape') closeFocus()
    return                      // در حالت focus بقیه کلیدها را نادیده بگیر
  }
  if (!ceremonyMode.value) return
  if (e.key === 'Escape') toggleCeremony()
  else if (e.key === 'ArrowLeft') ceremonyNext()
  else if (e.key === 'ArrowRight') ceremonyPrev()
}

const focusedKey = ref<string | null>(null)

const focusedSection = computed<WeightSection | null>(() =>
    focusedKey.value
        ? weightSections.value.find(s => s.key === focusedKey.value) ?? null
        : null
)

async function openFocus(key: string) {
  focusedKey.value = key
  document.documentElement.classList.add('overflow-hidden')
  await enterNativeFullscreen()
  requestAnimationFrame(fitZoom)   // زوم را با کل صفحه هماهنگ کن
}

async function closeFocus() {
  focusedKey.value = null
  document.documentElement.classList.remove('overflow-hidden')
  await exitNativeFullscreen()
}

onMounted(() => {
  window.addEventListener('keydown', onCeremonyKey)
  document.addEventListener('fullscreenchange', onFsChange)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onCeremonyKey)
  document.removeEventListener('fullscreenchange', onFsChange)
  document.documentElement.classList.remove('overflow-hidden')
})
</script>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

.toast-enter-active, .toast-leave-active { transition: all 0.3s ease; }
.toast-enter-from { opacity: 0; transform: translateX(-30px); }
.toast-leave-to { opacity: 0; transform: scale(0.9); }

.draw-overlay-enter-active { transition: opacity 0.4s ease; }
.draw-overlay-leave-active { transition: opacity 0.5s ease 0.1s; }
.draw-overlay-enter-from, .draw-overlay-leave-to { opacity: 0; }

.draw-spin { animation: spinBounce 1.2s ease-in-out infinite; }
@keyframes spinBounce {
  0% { transform: rotate(0deg) scale(1); }
  40% { transform: rotate(200deg) scale(1.1); }
  70% { transform: rotate(340deg) scale(0.95); }
  100% { transform: rotate(360deg) scale(1); }
}
.draw-ping { animation: ringPing 1.5s ease-out infinite; }
@keyframes ringPing { 0% { transform: scale(1); opacity: 0.6; } 100% { transform: scale(1.6); opacity: 0; } }

.draw-text-glow { text-shadow: 0 0 30px rgba(99,102,241,0.8), 0 0 60px rgba(99,102,241,0.4); animation: textPulse 1.5s ease-in-out infinite; }
@keyframes textPulse {
  0%, 100% { text-shadow: 0 0 30px rgba(99,102,241,0.8), 0 0 60px rgba(99,102,241,0.4); }
  50% { text-shadow: 0 0 50px rgba(139,92,246,1), 0 0 90px rgba(99,102,241,0.6); }
}
.draw-progress { animation: progressFill 1.6s ease-in-out forwards; }
@keyframes progressFill { from { width: 0%; } to { width: 100%; } }

@keyframes floatUp {
  0% { transform: translateY(0) scale(1); opacity: 0.6; }
  80% { opacity: 0.4; }
  100% { transform: translateY(-100vh) scale(0.5); opacity: 0; }
}

.bracket-reveal-enter-active { animation: revealCard 0.5s ease both; }
@keyframes revealCard {
  from { opacity: 0; transform: translateY(20px) scale(0.97); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

/* ===== خطوط اتصال و فاصله‌گذاری براکت ===== */
.bracket-canvas {
  --bgap: 3rem;
  --stub: 1.5rem;      /* دقیقاً نصف --bgap */
  --cline: #cbd5e1;
  --cw: 2px;
  gap: var(--bgap);
}

.bracket-col  { display: flex; flex-direction: column; }
.bracket-slot { flex: 1 1 0; display: flex; align-items: center; position: relative; }
.match-wrap   { width: 100%; }

/* خط افقی خروجی: از لبه‌ی کارت تا وسط فاصله‌ی ستون‌ها */
.bracket-col .bracket-slot::before {
  content: '';
  position: absolute;
  top: 50%;
  width: var(--stub);
  height: var(--cw);
  background: var(--cline);
  transform: translateY(-50%);
}

/* خط عمودی: نیمه‌ی پایین برای اسلات فرد، نیمه‌ی بالا برای اسلات زوج */
.bracket-col .bracket-slot::after {
  content: '';
  position: absolute;
  width: var(--cw);
  height: 50%;
  background: var(--cline);
}
.bracket-col .bracket-slot:nth-child(odd)::after  { top: 50%; }
.bracket-col .bracket-slot:nth-child(even)::after { bottom: 50%; }

/* جهت: ستون‌های چپ به سمت فینال (inline-end)، ستون‌های راست به سمت فینال (inline-start) */
.bracket-col--left  .bracket-slot::before,
.bracket-col--left  .bracket-slot::after  { inset-inline-end:   calc(var(--stub) * -1); }
.bracket-col--right .bracket-slot::before,
.bracket-col--right .bracket-slot::after  { inset-inline-start: calc(var(--stub) * -1); }

/* نیمه‌نهایی (تک‌مسابقه) خط عمودی ندارد */
.bracket-col .bracket-slot:only-child::after { content: none; }

/* فینال: دو خط ورودی از هر طرف */
.bracket-final { position: relative; }
.bracket-final::before,
.bracket-final::after {
  content: '';
  position: absolute;
  top: 50%;
  width: var(--stub);
  height: var(--cw);
  background: var(--cline);
  transform: translateY(-50%);
}
.bracket-final::before { inset-inline-start: calc(var(--stub) * -1); }
.bracket-final::after  { inset-inline-end:   calc(var(--stub) * -1); }
</style>
