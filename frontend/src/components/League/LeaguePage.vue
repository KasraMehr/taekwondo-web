<script setup lang="ts">
import { computed, ref } from 'vue'
import { useUiStore, type LeagueTab } from '../../stores/ui'
import { useLeagueStore } from '../../stores/league'
import { loadWebLeagues } from '../../webApi'
import { GENDER_LABELS } from '../../data/categories'
import LeagueSetup from './LeagueSetup.vue'
import LeagueRoster from './LeagueRoster.vue'
import LeagueWeeks from './LeagueWeeks.vue'
import LeagueOverview from './LeagueOverview.vue'
import LeagueSettings from './LeagueSettings.vue'

const ui=useUiStore();const store=useLeagueStore();const league=computed(()=>store.currentLeague);const refreshing=ref(false);const error=ref('')
const tabs:Array<{key:LeagueTab;step?:number;label:string;hint:string}>=[
  {key:'setup',step:1,label:'تیم‌ها و قرعه‌کشی',hint:'ثبت و گروه‌بندی'},
  {key:'roster',step:2,label:'فهرست ورزشکاران',hint:'اعضا و انتقال‌ها'},
  {key:'weeks',step:3,label:'برگزاری هفته‌ها',hint:'ساخت تورنمنت'},
  {key:'overview',step:4,label:'گزارش و رده‌بندی',hint:'نتایج و PDF'},
  {key:'settings',label:'تنظیمات',hint:'قوانین امتیاز'},
]
async function refresh(){refreshing.value=true;error.value='';try{store.replaceFromBackend(await loadWebLeagues())}catch(e:any){error.value=e?.message||'به‌روزرسانی ناموفق بود'}finally{refreshing.value=false}}
</script>

<template>
  <div v-if="league" class="space-y-5">
    <header class="overflow-hidden rounded-2xl bg-gradient-to-l from-slate-950 to-slate-800 text-white shadow-lg">
      <div class="flex flex-col gap-4 p-6 lg:flex-row lg:items-center lg:justify-between"><div><div class="mb-2 flex flex-wrap items-center gap-2"><span class="rounded-full bg-emerald-400/15 px-3 py-1 text-xs text-emerald-300">لیگ فعال</span><span class="text-xs text-slate-400">شناسه {{league.id.slice(0,8)}}</span></div><h2 class="text-2xl font-black">{{league.name}}</h2><p class="mt-2 text-sm text-slate-300">{{league.ageCategory}} · {{GENDER_LABELS[league.gender]}} · {{league.season.stages.length}} مرحله · {{league.season.stages.reduce((n,s)=>n+s.weeks.length,0)}} هفته</p></div><div class="flex gap-2"><button :disabled="refreshing" class="rounded-xl bg-white/10 px-4 py-2 text-sm hover:bg-white/20 disabled:opacity-50" @click="refresh">{{refreshing?'در حال دریافت…':'↻ به‌روزرسانی'}}</button><button class="rounded-xl bg-white px-4 py-2 text-sm font-medium text-slate-900 hover:bg-slate-100" @click="ui.goHome()">بازگشت به مسابقات</button></div></div>
      <div class="grid grid-cols-3 border-t border-white/10 bg-white/5"><div class="p-3 text-center"><b class="text-xl">{{league.clubs.filter(c=>c.active!==false).length}}</b><div class="text-xs text-slate-400">تیم</div></div><div class="border-x border-white/10 p-3 text-center"><b class="text-xl">{{league.athletes.filter(a=>a.isActive).length}}</b><div class="text-xs text-slate-400">ورزشکار</div></div><div class="p-3 text-center"><b class="text-xl">{{league.season.groups.length}}</b><div class="text-xs text-slate-400">گروه</div></div></div>
    </header>
    <p v-if="error" class="rounded-xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{{error}}</p>
    <nav class="grid gap-2 rounded-2xl border border-slate-200 bg-white p-2 shadow-sm sm:grid-cols-2 lg:grid-cols-5"><button v-for="tab in tabs" :key="tab.key" class="flex items-center gap-3 rounded-xl px-3 py-3 text-right transition" :class="ui.leagueTab===tab.key?'bg-emerald-600 text-white shadow-sm':'text-slate-600 hover:bg-slate-50'" @click="ui.leagueTab=tab.key"><span v-if="tab.step" class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-sm font-bold" :class="ui.leagueTab===tab.key?'bg-white/20':'bg-slate-100'">{{tab.step}}</span><span><b class="block text-sm">{{tab.label}}</b><small :class="ui.leagueTab===tab.key?'text-emerald-100':'text-slate-400'">{{tab.hint}}</small></span></button></nav>
    <LeagueSetup v-if="ui.leagueTab==='setup'" />
    <LeagueRoster v-else-if="ui.leagueTab==='roster'" />
    <LeagueWeeks v-else-if="ui.leagueTab==='weeks'" />
    <LeagueOverview v-else-if="ui.leagueTab==='overview'" />
    <LeagueSettings v-else-if="ui.leagueTab==='settings'" />
  </div>
  <div v-else class="rounded-2xl border border-dashed border-slate-300 bg-white p-12 text-center text-slate-500">لیگ پیدا نشد.</div>
</template>
