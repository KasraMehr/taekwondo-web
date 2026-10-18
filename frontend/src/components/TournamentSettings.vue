<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useTournamentStore } from '../stores/tournament'
import { webApi } from '../webApi'
import { weightOptions, type AgeCategory, type Gender } from '../data/categories'
import { isoFromJalali, jalaliFromIso } from '../utils/poomsae'

type Option={value:string;label:string}
const store=useTournamentStore()
const tournament=computed(()=>store.currentTournament)
const loading=ref(true),saving=ref(false),message=ref(''),error=ref('')
const dateInput=ref(''),savingDate=ref(false)
const dayDateInputs=reactive<Record<number,string>>({})
const options=reactive<{scheduleModes:Option[];numberingScopes:Option[];numberingOrders:Option[];dayAssignments:Option[];finalCourtPolicies:Option[]}>({scheduleModes:[],numberingScopes:[],numberingOrders:[],dayAssignments:[],finalCourtPolicies:[]})
const settings=reactive<any>({
  scoring:{pointGap:15,pointGapFromRound:1,gamJeomLimitPerRound:5,roundsToWin:2,maxRounds:3},
  weighIn:{maxAttempts:2,toleranceKg:.2},draw:{type:'ranked',separateTeams:true},
  schedule:{matchMinutes:5,restMinutes:10,mode:'balanced',singleCourt:1,finalCourtPolicy:'primary',finalCourt:1,dayAssignment:'manual',days:[],categoryDays:{}},
  numbering:{startAt:1,scope:'court_day',order:'category'},
  hogu:{enabled:false,courtSizes:{},categorySizes:{}}
})
const hoguSizes=['1','2','3','4']
const categories=computed(()=>tournament.value?weightOptions(tournament.value.ageCategory as AgeCategory,tournament.value.gender as Gender):[])
function defaultHoguSize(category:string){const weight=Number(category.replace(/[+-]/g,''));if(!Number.isFinite(weight))return '1';if(weight<=40)return '1';if(weight<=55)return '2';if(weight<=70)return '3';return '4'}
function fillDefaultHoguSizes(){for(const category of categories.value)if(!hoguSizes.includes(String(settings.hogu.categorySizes[category]||'')))settings.hogu.categorySizes[category]=defaultHoguSize(category)}
function resetHoguDefaults(){for(const category of categories.value)settings.hogu.categorySizes[category]=defaultHoguSize(category)}
function merge(raw:any){raw=JSON.parse(JSON.stringify(raw||{}));for(const section of ['scoring','weighIn','draw','schedule','numbering','hogu'])Object.assign(settings[section],raw?.[section]||{});if(!settings.schedule.matchMinutes){settings.schedule.matchMinutes=5;settings.schedule.restMinutes=10};if(!Array.isArray(settings.schedule.days))settings.schedule.days=[];for(const day of settings.schedule.days)dayDateInputs[day.day]=jalaliFromIso(day.date);if(!settings.hogu.courtSizes)settings.hogu.courtSizes={};if(!settings.hogu.categorySizes)settings.hogu.categorySizes={};for(const court of Object.keys(settings.hogu.courtSizes))settings.hogu.courtSizes[court]=(settings.hogu.courtSizes[court]||[]).filter((size:string)=>hoguSizes.includes(String(size)));fillDefaultHoguSizes()}
function courtHasSize(court:number,size:string){return (settings.hogu.courtSizes[court]??[]).includes(size)}
function toggleCourtSize(court:number,size:string){const list=[...(settings.hogu.courtSizes[court]??[])];settings.hogu.courtSizes[court]=list.includes(size)?list.filter((x:string)=>x!==size):[...list,size]}
async function load(){if(!tournament.value)return;loading.value=true;error.value='';dateInput.value=jalaliFromIso(tournament.value.date);try{const [opts,current]=await Promise.all([webApi().call<any>('/tournament-settings-options'),webApi().call<any>(`/tournaments/${tournament.value.id}`)]);Object.assign(options,opts||{});store.replaceFromServer(current);merge(current.settings)}catch(e:any){error.value=e?.message||'دریافت تنظیمات ناموفق بود'}finally{loading.value=false}}
async function saveDate(){const t=tournament.value;if(!t||!dateInput.value)return;savingDate.value=true;error.value='';message.value='';try{const isoDate=isoFromJalali(dateInput.value);const updated=await webApi().call<any>(`/tournaments/${t.id}`,'PUT',{name:t.name,date:isoDate,courts:t.courts,gender:t.gender,ageCategory:t.ageCategory,format:t.format,leagueId:t.leagueId??null,stageOrder:t.stageOrder??null,weekId:t.weekId??null,groupId:t.groupId??null,teamTournamentId:t.teamTournamentId??null});store.replaceFromServer(updated);await load();message.value='تاریخ مسابقه ذخیره شد'}catch(e:any){error.value=e?.message||'تغییر تاریخ ناموفق بود'}finally{savingDate.value=false}}
function normalizeDateInput(){try{dateInput.value=jalaliFromIso(isoFromJalali(dateInput.value))}catch{/* show validation when saving */}}
function ensureDays(){const base=tournament.value?.date.slice(0,10)||new Date().toISOString().slice(0,10);const second=new Date(`${base}T00:00:00Z`);second.setUTCDate(second.getUTCDate()+1);settings.schedule.days=[{day:1,date:base,label:'روز اول'},{day:2,date:second.toISOString().slice(0,10),label:'روز دوم'}];for(const day of settings.schedule.days)dayDateInputs[day.day]=jalaliFromIso(day.date)}
function clearSecondDay(){settings.schedule.days=settings.schedule.days.slice(0,1);settings.schedule.dayAssignment='manual';settings.schedule.categoryDays={}}
async function save(){
  const event = tournament.value
  if(!event || saving.value) return
  saving.value=true;error.value='';message.value=''
  fillDefaultHoguSizes()
  try {
    const days=settings.schedule.days.map((day:any)=>({...day,date:isoFromJalali(dayDateInputs[day.day]||'')}))
    const payload={scoring:{...settings.scoring},weighIn:{...settings.weighIn},draw:{...settings.draw},schedule:{...settings.schedule,days,categoryDays:{...settings.schedule.categoryDays}},numbering:{...settings.numbering},hogu:{enabled:Boolean(settings.hogu.enabled),courtSizes:{...settings.hogu.courtSizes},categorySizes:{...settings.hogu.categorySizes}}}
    const updated=await store.saveTournamentSettings(event.id,payload)
    merge(updated.settings)
    const games=updated.matches.filter(m=>!m.isBye)
    message.value=games.length && games.every(m=>m.matchNumber)
      ? 'تنظیمات، زمین‌ها و شمارهٔ بازی‌ها ذخیره شد.'
      : 'تنظیمات ذخیره شد؛ شماره‌گذاری پس از تکمیل وزن‌کشی و قرعهٔ همهٔ وزن‌ها انجام می‌شود.'
  } catch(e:any) { error.value=e?.message||'ذخیره تنظیمات ناموفق بود' }
  finally { saving.value=false }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-5 sm:flex-row sm:items-center sm:justify-between"><div><h3 class="text-lg font-black text-slate-900">تنظیمات برگزاری</h3><p class="mt-1 text-sm text-slate-500">قوانین مسابقه، قرعه، زمین‌ها و شماره‌گذاری بازی‌ها</p></div><button :disabled="saving||loading" class="rounded-xl bg-blue-600 px-6 py-2.5 text-sm font-bold text-white shadow hover:bg-blue-700 disabled:opacity-50" @click="save">{{saving?'در حال ذخیره…':'ذخیره تنظیمات'}}</button></div>
    <div class="flex flex-wrap items-end gap-3 rounded-2xl border border-slate-200 bg-white p-5"><label class="text-sm font-bold text-slate-700">تاریخ برگزاری مسابقه (شمسی)<input v-model="dateInput" type="text" inputmode="numeric" dir="ltr" placeholder="۱۴۰۵/۰۷/۱۱" class="mt-2 block rounded-xl border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900" @blur="normalizeDateInput"></label><button :disabled="savingDate||loading||!dateInput||dateInput===jalaliFromIso(tournament?.date)" class="rounded-xl bg-slate-900 px-5 py-2.5 text-sm font-bold text-white disabled:opacity-40" @click="saveDate">{{savingDate?'در حال ذخیره…':'ثبت تاریخ جدید'}}</button><p class="w-full text-xs text-slate-500">با تغییر تاریخ، روزهای برگزاری نیز به همان تعداد روز جابه‌جا می‌شوند.</p></div>
    <div v-if="loading" class="rounded-2xl bg-blue-50 p-5 text-sm text-blue-700">در حال دریافت تنظیمات…</div>
    <template v-else>
      <p v-if="error" class="rounded-xl bg-rose-50 p-3 text-sm text-rose-700">{{error}}</p><p v-if="message" class="rounded-xl bg-emerald-50 p-3 text-sm text-emerald-700">{{message}}</p>
      <div class="grid gap-5 lg:grid-cols-2">
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۱</span><div><h4>قوانین مبارزه</h4><p>راندها، اختلاف امتیاز و گمجوم</p></div></div><div class="settings-grid"><label>راند لازم برای برد<input v-model.number="settings.scoring.roundsToWin" type="number" min="1"></label><label>حداکثر راند<input v-model.number="settings.scoring.maxRounds" type="number" min="1"></label><label>اختلاف امتیاز<input v-model.number="settings.scoring.pointGap" type="number" min="1"></label><label>شروع اختلاف از راند<input v-model.number="settings.scoring.pointGapFromRound" type="number" min="1"></label><label class="sm:col-span-2">سقف گمجوم در هر راند<input v-model.number="settings.scoring.gamJeomLimitPerRound" type="number" min="1"></label></div></section>
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۲</span><div><h4>وزن‌کشی و قرعه</h4><p>تعداد تلاش و جداسازی باشگاه‌ها</p></div></div><div class="settings-grid"><label>حداکثر تلاش وزن‌کشی<input v-model.number="settings.weighIn.maxAttempts" type="number" min="1"></label><label>تلورانس وزن (کیلوگرم)<input v-model.number="settings.weighIn.toleranceKg" type="number" min="0" step="0.1"></label><label class="sm:col-span-2 toggle-row"><input v-model="settings.draw.separateTeams" type="checkbox">ورزشکاران هم‌باشگاهی در قرعه از هم جدا شوند</label></div></section>
        <section class="settings-card lg:col-span-2"><div class="settings-title"><span class="settings-index">۳</span><div><h4>چیدمان زمین‌ها</h4><p>تغییر روش پخش، هوگو یا محل فینال، چیدمان دستی قبلی را با تنظیمات جدید جایگزین می‌کند.</p></div></div><div class="settings-grid lg:grid-cols-3"><label>روش پخش بازی‌ها<select v-model="settings.schedule.mode"><option v-for="o in options.scheduleModes" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label>زمین حالت تک‌زمین<input v-model.number="settings.schedule.singleCourt" type="number" min="1" :max="tournament?.courts||1"></label><label>محل فینال<select v-model="settings.schedule.finalCourtPolicy"><option v-for="o in options.finalCourtPolicies" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label v-if="settings.schedule.finalCourtPolicy==='fixed'">زمین ثابت فینال<input v-model.number="settings.schedule.finalCourt" type="number" min="1" :max="tournament?.courts||1"></label></div>
          <div class="mt-5 rounded-xl border border-blue-100 bg-blue-50/50 p-4"><label class="toggle-row"><input v-model="settings.hogu.enabled" type="checkbox">چیدمان بازی‌ها بر اساس سایز هوگو</label><p class="mt-2 text-xs leading-6 text-slate-500">سایز هر رده وزنی به‌صورت خودکار تعیین می‌شود. فقط مشخص کنید در هر زمین چه سایزهایی موجود است.</p>
            <div v-if="settings.hogu.enabled" class="mt-4 space-y-4"><div class="grid gap-3 md:grid-cols-2"><div v-for="court in (tournament?.courts||1)" :key="court" class="rounded-lg border border-slate-200 bg-white p-3"><div class="mb-2 text-sm font-bold">هوگوهای موجود در زمین {{court}}</div><div class="flex flex-wrap gap-2"><label v-for="size in hoguSizes" :key="size" class="size-chip"><input type="checkbox" :checked="courtHasSize(court,size)" @change="toggleCourtSize(court,size)">سایز {{size}}</label></div></div></div>
              <div class="rounded-lg border border-slate-200 bg-white p-3"><div class="mb-3 flex items-center justify-between gap-3"><div><div class="text-sm font-bold text-slate-800">سایز خودکار رده‌های وزنی</div><div class="mt-1 text-xs text-slate-500">در صورت نیاز می‌توانید یک مورد خاص را تغییر دهید.</div></div><button type="button" class="defaults-button" @click="resetHoguDefaults">بازگردانی پیش‌فرض</button></div><div class="hogu-category-grid"><label v-for="category in categories" :key="category" class="hogu-category"><span>وزن {{category}}</span><select v-model="settings.hogu.categorySizes[category]"><option v-for="size in hoguSizes" :key="size" :value="size">هوگو {{size}}</option></select></label></div></div>
            </div>
          </div>
        </section>
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۴</span><div><h4>روزهای برگزاری</h4><p>اجرای یک‌روزه یا تقسیم اوزان در دو روز</p></div></div><div class="mb-4 flex gap-2"><button class="choice-button" :class="settings.schedule.days.length<2?'choice-active':''" @click="clearSecondDay">یک روز</button><button class="choice-button" :class="settings.schedule.days.length>=2?'choice-active':''" @click="ensureDays">دو روز</button></div><div v-if="settings.schedule.days.length>=2" class="settings-grid"><label>روش تقسیم اوزان<select v-model="settings.schedule.dayAssignment"><option v-for="o in options.dayAssignments" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label v-for="day in settings.schedule.days" :key="day.day">{{day.label}} (شمسی)<input v-model="dayDateInputs[day.day]" type="text" inputmode="numeric" dir="ltr" placeholder="۱۴۰۵/۰۷/۱۱"></label></div></section>
        <section class="settings-card"><div class="settings-title"><span class="settings-index">۵</span><div><h4>شماره‌گذاری بازی‌ها</h4><p>پس از تکمیل قرعه همه اوزان اعمال می‌شود</p></div></div><div class="settings-grid"><label>مدت تخمینی هر بازی (دقیقه)<input v-model.number="settings.schedule.matchMinutes" type="number" min="1" max="60"></label><label>حداقل استراحت بین دو بازی (دقیقه)<input v-model.number="settings.schedule.restMinutes" type="number" min="0" max="120"></label><label>شماره شروع<input v-model.number="settings.numbering.startAt" type="number" min="1"></label><label>محدوده شماره<select v-model="settings.numbering.scope"><option v-for="o in options.numberingScopes" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label class="sm:col-span-2">ترتیب شماره‌گذاری<select v-model="settings.numbering.order"><option v-for="o in options.numberingOrders" :key="o.value" :value="o.value">{{o.label}}</option></select></label></div></section>
      </div>
    </template>
  </div>
</template>

<style scoped>
.settings-card{border:1px solid #e2e8f0;border-radius:1rem;background:white;padding:1.25rem;box-shadow:0 1px 2px #0f172a0a}.settings-title{display:flex;align-items:center;gap:.75rem;margin-bottom:1.25rem}.settings-title h4{font-weight:900;color:#0f172a}.settings-title p{margin-top:.15rem;font-size:.75rem;color:#94a3b8}.settings-index{display:grid;height:2.5rem;width:2.5rem;flex:none;place-items:center;border-radius:.75rem;background:#eff6ff;font-weight:900;color:#2563eb}.settings-grid{display:grid;gap:1rem}@media(min-width:640px){.settings-grid{grid-template-columns:repeat(2,minmax(0,1fr))}}.settings-grid label{font-size:.75rem;font-weight:700;color:#475569}.settings-grid input:not([type=checkbox]),.settings-grid select{display:block;width:100%;margin-top:.4rem;border:1px solid #cbd5e1;border-radius:.65rem;background:white;padding:.65rem .75rem;font-size:.875rem;color:#0f172a;outline:none}.settings-grid input:focus,.settings-grid select:focus{border-color:#3b82f6;box-shadow:0 0 0 3px #dbeafe}.toggle-row{display:flex;align-items:center;gap:.65rem;border-radius:.75rem;background:#f8fafc;padding:.8rem}.toggle-row input{height:1rem;width:1rem}.size-chip{display:flex;align-items:center;gap:.4rem;border:1px solid #e2e8f0;border-radius:.6rem;padding:.45rem .65rem;font-size:.75rem}.size-chip input{height:1rem;width:1rem}.hogu-category-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.65rem}@media(min-width:640px){.hogu-category-grid{grid-template-columns:repeat(5,minmax(0,1fr))}}.hogu-category{display:flex;align-items:center;justify-content:space-between;gap:.5rem;border-radius:.65rem;background:#f8fafc;padding:.5rem .65rem;font-size:.75rem;font-weight:700;color:#475569}.hogu-category select{min-width:5rem;border:1px solid #cbd5e1;border-radius:.5rem;background:white;padding:.35rem;font-size:.75rem;color:#0f172a}.defaults-button{flex:none;border:1px solid #bfdbfe;border-radius:.6rem;background:#eff6ff;padding:.45rem .65rem;font-size:.7rem;font-weight:800;color:#1d4ed8}.choice-button{flex:1;border:1px solid #cbd5e1;border-radius:.7rem;padding:.6rem;font-size:.8rem;font-weight:700;color:#64748b}.choice-active{border-color:#3b82f6;background:#eff6ff;color:#1d4ed8}
</style>
