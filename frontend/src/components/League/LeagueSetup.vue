<script setup lang="ts">
import { computed, ref } from 'vue'
import { useLeagueStore } from '../../stores/league'
import { loadWebLeagues, webApi } from '../../webApi'
import type { LeagueClub } from '../../types'

const store = useLeagueStore()
const league = computed(() => store.currentLeague)
const names = ref('')
const groupCount = ref(2)
const busy = ref(false)
const notice = ref('')
const error = ref('')
const teams = computed(() => (league.value?.clubs ?? []) as Array<LeagueClub & { teamId?: string }>)
const groups = computed(() => [...(league.value?.season.groups ?? [])].sort((a,b) => a.order-b.order))

async function refresh(){ store.replaceFromBackend(await loadWebLeagues()) }
function parsedNames(){ return [...new Set(names.value.split(/[\n,،;]/).map(v=>v.trim()).filter(Boolean))] }

async function addTeams(){
  if(!league.value || !parsedNames().length) return
  busy.value=true; error.value=''; notice.value=''
  try{
    const allClubs = await webApi().call<any[]>('/clubs')
    const existingTeams = new Set(teams.value.map(t=>t.name))
    let added=0
    for(const name of parsedNames()){
      if(existingTeams.has(name)) continue
      let club=allClubs.find(c=>c.name===name)
      if(!club) club=await webApi().call<any>('/clubs','POST',{name})
      await webApi().call(`/leagues/${league.value.id}/teams`,'POST',{clubId:club.id,groupId:null})
      added++
    }
    await refresh(); names.value=''; notice.value=`${added} تیم ثبت شد`
  }catch(e:any){error.value=e?.message||'ثبت تیم‌ها ناموفق بود'; await refresh()}
  finally{busy.value=false}
}

async function ensureGroups(count:number){
  if(!league.value) return
  for(let i=groups.value.length;i<count;i++) await webApi().call(`/leagues/${league.value.id}/groups`,'POST',{name:`گروه ${String.fromCharCode(65+i)}`})
  await refresh()
}

async function draw(){
  if(!league.value || teams.value.length<2) return error.value='برای قرعه‌کشی حداقل دو تیم لازم است'
  busy.value=true; error.value=''; notice.value=''
  try{
    const count=Math.max(1,Math.min(groupCount.value,teams.value.length))
    await ensureGroups(count)
    const target=groups.value.slice(0,count)
    const shuffled=[...teams.value].sort(()=>Math.random()-.5)
    await Promise.all(shuffled.map((team,i)=>webApi().call(`/leagues/${league.value!.id}/teams/${team.teamId}`,'PUT',{groupId:target[i%count].id,active:team.active!==false})))
    await refresh(); notice.value='قرعه‌کشی انجام شد و تیم‌ها به‌صورت متوازن در گروه‌ها قرار گرفتند'
  }catch(e:any){error.value=e?.message||'قرعه‌کشی ناموفق بود'}finally{busy.value=false}
}

async function moveTeam(team:LeagueClub & {teamId?:string},groupId:string){
  if(!league.value||!team.teamId)return
  try{await webApi().call(`/leagues/${league.value.id}/teams/${team.teamId}`,'PUT',{groupId:groupId||null,active:team.active!==false});await refresh()}
  catch(e:any){error.value=e?.message||'جابه‌جایی تیم ناموفق بود'}
}
</script>

<template>
  <div v-if="league" class="space-y-5">
    <section class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-4 flex items-center gap-3"><span class="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-100 text-xl">1</span><div><h3 class="font-bold text-slate-900">ثبت تیم‌های لیگ</h3><p class="text-sm text-slate-500">نام هر تیم را در یک خط وارد کنید.</p></div></div>
        <textarea v-model="names" rows="7" class="w-full rounded-xl border border-slate-300 p-3 text-sm focus:border-emerald-500 focus:outline-none" placeholder="مثال:&#10;باشگاه آزادی&#10;باشگاه قهرمانان"></textarea>
        <button :disabled="busy||!parsedNames().length" class="mt-3 w-full rounded-xl bg-emerald-600 px-4 py-2.5 font-medium text-white hover:bg-emerald-700 disabled:opacity-50" @click="addTeams">ثبت تیم‌ها</button>
      </div>
      <div class="rounded-2xl bg-slate-900 p-5 text-white shadow-sm">
        <div class="mb-4 flex items-center gap-3"><span class="flex h-10 w-10 items-center justify-center rounded-xl bg-white/10 text-xl">2</span><div><h3 class="font-bold">قرعه‌کشی گروهی</h3><p class="text-sm text-slate-300">توزیع تصادفی و متوازن تیم‌ها</p></div></div>
        <label class="text-sm text-slate-300">تعداد گروه‌ها<input v-model.number="groupCount" type="number" min="1" :max="Math.max(1,teams.length)" class="mt-2 w-full rounded-xl border border-slate-600 bg-slate-800 px-3 py-2 text-white"></label>
        <div class="mt-4 rounded-xl bg-white/5 p-3 text-sm"><div class="flex justify-between"><span>تیم‌های آماده</span><b>{{ teams.filter(t=>t.active!==false).length }}</b></div><div class="mt-2 flex justify-between"><span>گروه‌های فعلی</span><b>{{ groups.length }}</b></div></div>
        <button :disabled="busy||teams.length<2" class="mt-4 w-full rounded-xl bg-amber-400 px-4 py-2.5 font-bold text-slate-900 hover:bg-amber-300 disabled:opacity-50" @click="draw">🎲 انجام قرعه‌کشی</button>
      </div>
    </section>
    <p v-if="notice" class="rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{{notice}}</p><p v-if="error" class="rounded-xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{{error}}</p>
    <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="mb-4 flex items-center justify-between"><div><h3 class="font-bold text-slate-900">تیم‌های ثبت‌شده</h3><p class="text-sm text-slate-500">نتیجه قرعه را ببینید یا گروه هر تیم را دستی اصلاح کنید.</p></div><span class="rounded-full bg-slate-100 px-3 py-1 text-sm">{{teams.length}} تیم</span></div>
      <div v-if="!teams.length" class="rounded-xl border-2 border-dashed border-slate-200 py-12 text-center text-sm text-slate-400">هنوز تیمی ثبت نشده است.</div>
      <div v-else class="grid gap-3 md:grid-cols-2 xl:grid-cols-3"><article v-for="(team,i) in teams" :key="team.id" class="flex items-center gap-3 rounded-xl border border-slate-200 p-3"><span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 font-bold text-slate-600">{{i+1}}</span><div class="min-w-0 flex-1"><div class="truncate font-medium">{{team.name}}</div><select :value="team.groupId||''" class="mt-1 w-full rounded-lg border border-slate-200 bg-slate-50 px-2 py-1 text-xs" @change="moveTeam(team,($event.target as HTMLSelectElement).value)"><option value="">بدون گروه</option><option v-for="g in groups" :key="g.id" :value="g.id">{{g.name}}</option></select></div></article></div>
    </section>
  </div>
</template>
