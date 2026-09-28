<script setup lang="ts">
import { computed, ref } from 'vue'
import { useLeagueStore } from '../../stores/league'
import { loadWebLeagues, webApi } from '../../webApi'
import AthleteFormModal from './AthleteFormModal.vue'
import type { AthleteDraft, LeagueAthlete, LeagueClub } from '../../types'

type WebClub=LeagueClub&{teamId?:string}
type WebAthlete=LeagueAthlete&{profileId?:string;teamId?:string|null;coachName?:string;ranking?:number|null}
const store=useLeagueStore(); const league=computed(()=>store.currentLeague)
const teams=computed(()=>(league.value?.clubs??[]) as WebClub[])
const athletes=computed(()=>(league.value?.athletes??[]).filter(a=>a.isActive) as WebAthlete[])
const free=computed(()=>athletes.value.filter(a=>!a.teamId))
const search=ref(''); const formOpen=ref(false); const selectedTeam=ref<WebClub|null>(null); const editing=ref<WebAthlete|null>(null); const error=ref('')
const groups=computed(()=>league.value?.season.groups??[])
const shownTeams=computed(()=>teams.value.filter(t=>!search.value.trim()||t.name.includes(search.value.trim())||athletes.value.some(a=>a.teamId===t.teamId&&a.name.includes(search.value.trim()))))
const members=(team:WebClub)=>athletes.value.filter(a=>a.teamId===team.teamId)
async function refresh(){store.replaceFromBackend(await loadWebLeagues())}
function openAdd(team:WebClub|null){selectedTeam.value=team;editing.value=null;formOpen.value=true;error.value=''}
function openEdit(a:WebAthlete){editing.value=a;selectedTeam.value=teams.value.find(t=>t.teamId===a.teamId)??null;formOpen.value=true;error.value=''}
function payload(a:WebAthlete,teamId:string|null,groupId:string|null,weight=a.weightCategory){return{teamId,groupId,weightCategory:weight,coachName:a.coachName??'',ranking:a.ranking??null,active:true}}
async function save(draft:AthleteDraft){
  if(!league.value)return
  try{
    const club=teams.value.find(t=>t.id===draft.clubId)??selectedTeam.value
    if(editing.value){
      if(!editing.value.profileId)throw new Error('پروفایل ورزشکار پیدا نشد')
      await webApi().call(`/athletes/${editing.value.profileId}`,'PUT',{name:draft.name,gender:draft.gender,clubId:club?.id??null})
      await webApi().call(`/leagues/${league.value.id}/athletes/${editing.value.id}`,'PUT',payload(editing.value,club?.teamId??null,club?.groupId??editing.value.groupId,draft.weightCategory))
    }else await webApi().call(`/leagues/${league.value.id}/athletes`,'POST',{name:draft.name,gender:draft.gender,clubId:club?.id??null,teamId:club?.teamId??null,groupId:club?.groupId??null,weightCategory:draft.weightCategory,coachName:'',ranking:null})
    await refresh();formOpen.value=false
  }catch(e:any){error.value=e?.message||'ذخیره ورزشکار ناموفق بود'}
}
async function move(a:WebAthlete,value:string){
  if(!league.value)return
  const team=value==='free'?null:teams.value.find(t=>t.teamId===value)??null
  try{await webApi().call(`/leagues/${league.value.id}/athletes/${a.id}`,'PUT',payload(a,team?.teamId??null,team?.groupId??a.groupId));await refresh()}
  catch(e:any){error.value=e?.message||'انتقال ورزشکار ناموفق بود'}
}
async function moveFreeGroup(a:WebAthlete,groupId:string){if(!league.value)return;try{await webApi().call(`/leagues/${league.value.id}/athletes/${a.id}`,'PUT',payload(a,null,groupId||null));await refresh()}catch(e:any){error.value=e?.message||'تخصیص گروه ناموفق بود'}}
async function remove(a:WebAthlete){if(!league.value||!confirm(`«${a.name}» از لیگ حذف شود؟`))return;try{await webApi().call(`/leagues/${league.value.id}/athletes/${a.id}`,'DELETE');await refresh()}catch(e:any){error.value=e?.message||'حذف ناموفق بود'}}
</script>

<template>
  <div v-if="league" class="space-y-5">
    <div class="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm sm:flex-row sm:items-center"><div class="flex-1"><h3 class="font-bold text-slate-900">فهرست تیم‌ها و ورزشکاران</h3><p class="text-sm text-slate-500">اعضای هر تیم را ببینید و بدون حذف سابقه بین تیم‌ها جابه‌جا کنید.</p></div><input v-model="search" type="search" class="rounded-xl border border-slate-300 px-3 py-2 text-sm" placeholder="جستجوی تیم یا ورزشکار"><button class="rounded-xl bg-slate-900 px-4 py-2 text-sm font-medium text-white" @click="openAdd(null)">+ بازیکن آزاد</button></div>
    <p v-if="error" class="rounded-xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{{error}}</p>
    <section v-if="free.length" class="rounded-2xl border border-amber-200 bg-amber-50/60 p-5"><div class="mb-3 flex items-center justify-between"><div><h3 class="font-bold text-amber-900">بازیکنان آزاد</h3><p class="text-sm text-amber-700">می‌توانند بدون عضویت تیم در یک گروه قرار بگیرند.</p></div><span class="rounded-full bg-amber-100 px-3 py-1 text-sm text-amber-800">{{free.length}} نفر</span></div><div class="grid gap-2 lg:grid-cols-2"><div v-for="a in free" :key="a.id" class="flex flex-wrap items-center gap-2 rounded-xl bg-white p-3"><div class="min-w-[8rem] flex-1"><b class="text-sm">{{a.name}}</b><div class="text-xs text-slate-500">{{a.weightCategory}}</div></div><select :value="a.groupId||''" class="rounded-lg border px-2 py-1 text-xs" @change="moveFreeGroup(a,($event.target as HTMLSelectElement).value)"><option value="">بدون گروه</option><option v-for="g in groups" :key="g.id" :value="g.id">{{g.name}}</option></select><select value="free" class="rounded-lg border px-2 py-1 text-xs" @change="move(a,($event.target as HTMLSelectElement).value)"><option value="free">آزاد</option><option v-for="t in teams" :key="t.teamId" :value="t.teamId">{{t.name}}</option></select><button class="text-xs text-blue-600" @click="openEdit(a)">ویرایش</button><button class="text-xs text-rose-600" @click="remove(a)">حذف</button></div></div></section>
    <div v-if="!shownTeams.length" class="rounded-2xl border-2 border-dashed border-slate-200 bg-white py-16 text-center text-slate-400">ابتدا تیم‌ها را در مرحله قبل ثبت کنید.</div>
    <section v-else class="grid gap-4 xl:grid-cols-2"><article v-for="team in shownTeams" :key="team.id" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"><header class="flex items-center gap-3 border-b border-slate-100 bg-slate-50 p-4"><span class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-100">🥋</span><div class="min-w-0 flex-1"><h3 class="truncate font-bold">{{team.name}}</h3><p class="text-xs text-slate-500">{{groups.find(g=>g.id===team.groupId)?.name||'بدون گروه'}} · {{members(team).length}} ورزشکار</p></div><button class="rounded-lg bg-emerald-600 px-3 py-2 text-xs font-medium text-white" @click="openAdd(team)">+ ورزشکار</button></header><div v-if="!members(team).length" class="p-8 text-center text-sm text-slate-400">هنوز ورزشکاری ثبت نشده است.</div><div v-else class="divide-y divide-slate-100"><div v-for="a in members(team)" :key="a.id" class="flex flex-wrap items-center gap-2 p-3"><div class="min-w-[9rem] flex-1"><b class="text-sm">{{a.name}}</b><div class="text-xs text-slate-500">وزن {{a.weightCategory}}</div></div><select :value="a.teamId||'free'" class="max-w-40 rounded-lg border border-slate-200 px-2 py-1 text-xs" @change="move(a,($event.target as HTMLSelectElement).value)"><option value="free">بازیکن آزاد</option><option v-for="t in teams" :key="t.teamId" :value="t.teamId">{{t.name}}</option></select><button class="text-xs text-blue-600" @click="openEdit(a)">ویرایش</button><button class="text-xs text-rose-600" @click="remove(a)">حذف</button></div></div></article></section>
    <AthleteFormModal :open="formOpen" :gender="league.gender" :age-category="league.ageCategory" :athlete="editing" :clubs="teams" :error="error" @submit="save" @close="formOpen=false" />
  </div>
</template>
