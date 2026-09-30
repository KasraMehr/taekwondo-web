<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useTournamentStore } from '../stores/tournament'
import { webApi } from '../webApi'

type Option={value:string;label:string}
const store=useTournamentStore()
const tournament=computed(()=>store.currentTournament)
const loading=ref(true),saving=ref(false),message=ref(''),error=ref('')
const options=reactive<{scheduleModes:Option[];numberingScopes:Option[];numberingOrders:Option[];dayAssignments:Option[];finalCourtPolicies:Option[]}>({scheduleModes:[],numberingScopes:[],numberingOrders:[],dayAssignments:[],finalCourtPolicies:[]})
const settings=reactive<any>({
  scoring:{pointGap:15,pointGapFromRound:1,gamJeomLimitPerRound:5,roundsToWin:2,maxRounds:3},
  weighIn:{maxAttempts:2,toleranceKg:.2},draw:{type:'ranked',separateTeams:true},
  schedule:{mode:'balanced',singleCourt:1,finalCourtPolicy:'primary',finalCourt:1,dayAssignment:'alternating',days:[],categoryDays:{}},
  numbering:{startAt:1,scope:'tournament',order:'rounds'}
})
function merge(raw:any){for(const section of ['scoring','weighIn','draw','schedule','numbering'])Object.assign(settings[section],raw?.[section]||{});if(!Array.isArray(settings.schedule.days))settings.schedule.days=[]}
async function load(){if(!tournament.value)return;loading.value=true;error.value='';try{const [opts,current]=await Promise.all([webApi().call<any>('/tournament-settings-options'),webApi().call<any>(`/tournaments/${tournament.value.id}/settings`)]);Object.assign(options,opts||{});merge(current)}catch(e:any){error.value=e?.message||'دریافت تنظیمات ناموفق بود'}finally{loading.value=false}}
function ensureDays(){const base=tournament.value?.date||new Date().toISOString().slice(0,10);const second=new Date(`${base}T00:00:00`);second.setDate(second.getDate()+1);settings.schedule.days=[{day:1,date:base,label:'روز اول'},{day:2,date:second.toISOString().slice(0,10),label:'روز دوم'}]}
function clearSecondDay(){settings.schedule.days=settings.schedule.days.slice(0,1);settings.schedule.categoryDays={}}
async function save(){if(!tournament.value)return;saving.value=true;error.value='';message.value='';try{await webApi().call(`/tournaments/${tournament.value.id}/settings`,'PUT',settings);message.value='تنظیمات مسابقه ذخیره شد'}catch(e:any){error.value=e?.message||'ذخیره تنظیمات ناموفق بود'}finally{saving.value=false}}
onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-5 sm:flex-row sm:items-center sm:justify-between"><div><h3 class="text-lg font-black text-slate-900">تنظیمات برگزاری</h3><p class="mt-1 text-sm text-slate-500">قوانین مسابقه، قرعه، زمین‌ها و شماره‌گذاری بازی‌ها</p></div><button :disabled="saving||loading" class="rounded-xl bg-blue-600 px-6 py-2.5 text-sm font-bold text-white shadow hover:bg-blue-700 disabled:opacity-50" @click="save">{{saving?'در حال ذخیره…':'ذخیره تنظیمات'}}</button></div>
    <div v-if="loading" class="rounded-2xl bg-blue-50 p-5 text-sm text-blue-700">در حال دریافت تنظیمات…</div>
    <template v-else>
      <p v-if="error" class="rounded-xl bg-rose-50 p-3 text-sm text-rose-700">{{error}}</p><p v-if="message" class="rounded-xl bg-emerald-50 p-3 text-sm text-emerald-700">{{message}}</p>
      <div class="grid gap-5 lg:grid-cols-2">
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۱</span><div><h4>قوانین مبارزه</h4><p>راندها، اختلاف امتیاز و گمجوم</p></div></div><div class="settings-grid"><label>راند لازم برای برد<input v-model.number="settings.scoring.roundsToWin" type="number" min="1"></label><label>حداکثر راند<input v-model.number="settings.scoring.maxRounds" type="number" min="1"></label><label>اختلاف امتیاز<input v-model.number="settings.scoring.pointGap" type="number" min="1"></label><label>شروع اختلاف از راند<input v-model.number="settings.scoring.pointGapFromRound" type="number" min="1"></label><label class="sm:col-span-2">سقف گمجوم در هر راند<input v-model.number="settings.scoring.gamJeomLimitPerRound" type="number" min="1"></label></div></section>
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۲</span><div><h4>وزن‌کشی و قرعه</h4><p>تعداد تلاش و جداسازی باشگاه‌ها</p></div></div><div class="settings-grid"><label>حداکثر تلاش وزن‌کشی<input v-model.number="settings.weighIn.maxAttempts" type="number" min="1"></label><label>تلورانس وزن (کیلوگرم)<input v-model.number="settings.weighIn.toleranceKg" type="number" min="0" step="0.1"></label><label class="sm:col-span-2 toggle-row"><input v-model="settings.draw.separateTeams" type="checkbox">ورزشکاران هم‌باشگاهی در قرعه از هم جدا شوند</label></div></section>
        <section class="settings-card lg:col-span-2"><div class="settings-title"><span class="settings-index">۳</span><div><h4>چیدمان زمین‌ها</h4><p>روش توزیع بازی‌ها و محل برگزاری فینال</p></div></div><div class="settings-grid lg:grid-cols-3"><label>روش پخش بازی‌ها<select v-model="settings.schedule.mode"><option v-for="o in options.scheduleModes" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label>زمین حالت تک‌زمین<input v-model.number="settings.schedule.singleCourt" type="number" min="1" :max="tournament?.courts||1"></label><label>محل فینال<select v-model="settings.schedule.finalCourtPolicy"><option v-for="o in options.finalCourtPolicies" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label v-if="settings.schedule.finalCourtPolicy==='fixed'">زمین ثابت فینال<input v-model.number="settings.schedule.finalCourt" type="number" min="1" :max="tournament?.courts||1"></label></div></section>
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۴</span><div><h4>روزهای برگزاری</h4><p>اجرای یک‌روزه یا تقسیم اوزان در دو روز</p></div></div><div class="mb-4 flex gap-2"><button class="choice-button" :class="settings.schedule.days.length<2?'choice-active':''" @click="clearSecondDay">یک روز</button><button class="choice-button" :class="settings.schedule.days.length>=2?'choice-active':''" @click="ensureDays">دو روز</button></div><div v-if="settings.schedule.days.length>=2" class="settings-grid"><label>روش تقسیم اوزان<select v-model="settings.schedule.dayAssignment"><option v-for="o in options.dayAssignments" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label v-for="day in settings.schedule.days" :key="day.day">{{day.label}}<input v-model="day.date" type="date"></label></div></section>
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۵</span><div><h4>شماره‌گذاری بازی‌ها</h4><p>پس از تکمیل قرعه همه اوزان اعمال می‌شود</p></div></div><div class="settings-grid"><label>شماره شروع<input v-model.number="settings.numbering.startAt" type="number" min="1"></label><label>محدوده شماره<select v-model="settings.numbering.scope"><option v-for="o in options.numberingScopes" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label class="sm:col-span-2">ترتیب شماره‌گذاری<select v-model="settings.numbering.order"><option v-for="o in options.numberingOrders" :key="o.value" :value="o.value">{{o.label}}</option></select></label></div></section>
      </div>
    </template>
  </div>
</template>

<style scoped>
.settings-card{border:1px solid #e2e8f0;border-radius:1rem;background:white;padding:1.25rem;box-shadow:0 1px 2px #0f172a0a}.settings-title{display:flex;align-items:center;gap:.75rem;margin-bottom:1.25rem}.settings-title h4{font-weight:900;color:#0f172a}.settings-title p{margin-top:.15rem;font-size:.75rem;color:#94a3b8}.settings-index{display:grid;height:2.5rem;width:2.5rem;flex:none;place-items:center;border-radius:.75rem;background:#eff6ff;font-weight:900;color:#2563eb}.settings-grid{display:grid;gap:1rem}@media(min-width:640px){.settings-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}.settings-grid label{font-size:.75rem;font-weight:700;color:#475569}.settings-grid input:not([type=checkbox]),.settings-grid select{display:block;width:100%;margin-top:.4rem;border:1px solid #cbd5e1;border-radius:.65rem;background:white;padding:.65rem .75rem;font-size:.875rem;color:#0f172a;outline:none}.settings-grid input:focus,.settings-grid select:focus{border-color:#3b82f6;box-shadow:0 0 0 3px #dbeafe}.toggle-row{display:flex;align-items:center;gap:.65rem;border-radius:.75rem;background:#f8fafc;padding:.8rem}.toggle-row input{height:1rem;width:1rem}.choice-button{flex:1;border:1px solid #cbd5e1;border-radius:.7rem;padding:.6rem;font-size:.8rem;font-weight:700;color:#64748b}.choice-active{border-color:#3b82f6;background:#eff6ff;color:#1d4ed8}
</style>
