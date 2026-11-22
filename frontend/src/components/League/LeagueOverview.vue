<script setup lang="ts">
import { computed, ref } from 'vue'
import { jsPDF } from 'jspdf'
import { domToCanvas } from 'modern-screenshot'
import { useLeagueStore } from '../../stores/league'
import { loadWebLeagues } from '../../webApi'

const store=useLeagueStore(); const league=computed(()=>store.currentLeague); const report=ref<HTMLElement|null>(null); const busy=ref(false); const error=ref('')
const teams=computed(()=>{
  if(!league.value)return[]
  const scored=store.getTeamRankings(league.value.id).filter(row=>row.club)
  const rows=league.value.clubs.filter(c=>c.active!==false).map(club=>scored.find(row=>row.club===club.name)??{rank:0,club:club.name,totalPoints:0,gold:0,silver:0,bronze:0,weighInPoints:0,winPoints:0,roundDiff:0,wins2_0:0,wins2_1:0,losses0_2:0,losses1_2:0,totalWins:0,totalLosses:0,athleteCount:league.value!.athletes.filter(a=>a.isActive&&a.clubId===club.id).length,groupId:club.groupId,groupName:league.value!.season.groups.find(g=>g.id===club.groupId)?.name})
  return rows.sort((a,b)=>b.totalPoints-a.totalPoints||b.gold-a.gold||b.silver-a.silver||a.club.localeCompare(b.club,'fa')).map((row,i)=>({...row,rank:i+1}))
})
const players=computed(()=>{
  if(!league.value)return[]
  const scored=store.getPlayerRankings(league.value.id)
  const rows=league.value.athletes.filter(a=>a.isActive).map(a=>{
    const profileId=(a as typeof a&{profileId?:string}).profileId
    return scored.find(row=>row.athleteId===profileId||row.name===a.name&&row.weightCategory===a.weightCategory)??{rank:0,athleteId:a.id,name:a.name,club:a.club||a.coachName||'آزاد',weightCategory:a.weightCategory??'',totalPoints:0,gold:0,silver:0,bronze:0,weighInPoints:0,winPoints:0,roundDiff:0,wins2_0:0,wins2_1:0,losses0_2:0,losses1_2:0,totalWins:0,totalLosses:0,groupId:a.groupId,groupName:league.value!.season.groups.find(g=>g.id===a.groupId)?.name}
  })
  return rows.sort((a,b)=>b.totalPoints-a.totalPoints||b.gold-a.gold||b.silver-a.silver||a.name.localeCompare(b.name,'fa')).map((row,i)=>({...row,rank:i+1}))
})
const rankingSections=computed(()=>{
  if(!league.value)return[]
  const groups=league.value.season.groups
  if(groups.length<=1)return[{id:groups[0]?.id??'all',name:groups[0]?.name??'رده‌بندی کل لیگ',teams:teams.value,players:players.value}]
  const sections=groups.map(g=>({id:g.id,name:g.name,teams:teams.value.filter(t=>t.groupId===g.id).map((t,i)=>({...t,rank:i+1})),players:players.value.filter(p=>p.groupId===g.id).map((p,i)=>({...p,rank:i+1}))}))
  const ungroupedPlayers=players.value.filter(p=>!p.groupId)
  if(ungroupedPlayers.length)sections.push({id:'free',name:'بدون گروه',teams:[],players:ungroupedPlayers.map((p,i)=>({...p,rank:i+1}))})
  return sections
})
const medalSections=computed(()=>rankingSections.value.map(section=>{
  const byWeight=new Map<string,typeof section.players>()
  for(const player of section.players){
    if(!player.gold&&!player.silver&&!player.bronze)continue
    const weight=player.weightCategory||'وزن نامشخص'
    if(!byWeight.has(weight))byWeight.set(weight,[])
    byWeight.get(weight)!.push(player)
  }
  return {...section,weights:[...byWeight].sort(([a],[b])=>a.localeCompare(b,'fa',{numeric:true})).map(([weight,medalists])=>({weight,medalists:medalists.sort((a,b)=>b.gold-a.gold||b.silver-a.silver||b.bronze-a.bronze||b.totalPoints-a.totalPoints).map((player,i)=>({...player,medalRank:i+1}))}))}
}))
const stats=computed(()=>({teams:league.value?.clubs.filter(c=>c.active!==false).length??0,players:league.value?.athletes.filter(a=>a.isActive).length??0,weeks:league.value?.season.stages.reduce((n,s)=>n+s.weeks.length,0)??0,medals:players.value.reduce((n,p)=>n+p.gold+p.silver+p.bronze,0)}))
async function refresh(){busy.value=true;error.value='';try{store.replaceFromBackend(await loadWebLeagues())}catch(e:any){error.value=e?.message||'دریافت گزارش ناموفق بود'}finally{busy.value=false}}
async function exportPdf(){
  if(!report.value||!league.value)return
  busy.value=true;error.value=''
  try{
    const canvas=await domToCanvas(report.value,{scale:1.5,backgroundColor:'#ffffff'})
    const pdf=new jsPDF({orientation:'portrait',unit:'mm',format:'a4',compress:true})
    const pageW=210,pageH=297,margin=10,contentW=pageW-margin*2
    const pagePixels=Math.floor(canvas.width*(pageH-margin*2)/contentW)
    for(let top=0,page=0;top<canvas.height;top+=pagePixels,page++){
      if(page)pdf.addPage()
      const sliceHeight=Math.min(pagePixels,canvas.height-top)
      const slice=document.createElement('canvas')
      slice.width=canvas.width;slice.height=sliceHeight
      const ctx=slice.getContext('2d')
      if(!ctx)throw new Error('ساخت تصویر گزارش ممکن نشد')
      ctx.fillStyle='#ffffff';ctx.fillRect(0,0,slice.width,slice.height)
      ctx.drawImage(canvas,0,top,canvas.width,sliceHeight,0,0,canvas.width,sliceHeight)
      pdf.addImage(slice.toDataURL('image/jpeg',0.78),'JPEG',margin,margin,contentW,sliceHeight*contentW/canvas.width,undefined,'FAST')
    }
    pdf.save(`${league.value.name}-league-report.pdf`)
  }catch(e:any){error.value=e?.message||'ساخت PDF ناموفق بود'}finally{busy.value=false}
}
</script>

<template>
  <div v-if="league" class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-slate-900 p-4 text-white"><div><h3 class="font-bold">گزارش و رده‌بندی لیگ</h3><p class="text-sm text-slate-300">نسخه آماده انتشار شامل آمار، تیم‌ها و ورزشکاران</p></div><div class="flex gap-2"><button :disabled="busy" class="rounded-xl bg-white/10 px-4 py-2 text-sm hover:bg-white/20 disabled:opacity-50" @click="refresh">↻ به‌روزرسانی</button><button :disabled="busy" class="rounded-xl bg-rose-500 px-4 py-2 text-sm font-bold hover:bg-rose-400 disabled:opacity-50" @click="exportPdf">{{busy?'در حال آماده‌سازی…':'📄 خروجی PDF'}}</button></div></div>
    <p v-if="error" class="rounded-xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{{error}}</p>
    <div ref="report" dir="rtl" class="mx-auto space-y-6 rounded-2xl bg-white p-6 text-slate-900 shadow-sm" style="max-width: 980px">
      <header class="border-b-2 border-slate-900 pb-4 text-center"><div class="text-sm text-slate-500">گزارش رسمی مسابقات تکواندو</div><h1 class="mt-1 text-2xl font-black">{{league.name}}</h1><p class="mt-1 text-sm text-slate-600">{{league.ageCategory}} · {{league.gender==='male'?'پسران / مردان':'دختران / زنان'}} · تاریخ گزارش {{new Date().toLocaleDateString('fa-IR')}}</p></header>
      <section class="grid grid-cols-4 gap-3"><div class="rounded-xl bg-slate-100 p-4 text-center"><div class="text-2xl font-black">{{stats.teams}}</div><div class="text-xs text-slate-500">تیم</div></div><div class="rounded-xl bg-slate-100 p-4 text-center"><div class="text-2xl font-black">{{stats.players}}</div><div class="text-xs text-slate-500">ورزشکار</div></div><div class="rounded-xl bg-slate-100 p-4 text-center"><div class="text-2xl font-black">{{stats.weeks}}</div><div class="text-xs text-slate-500">هفته</div></div><div class="rounded-xl bg-slate-100 p-4 text-center"><div class="text-2xl font-black">{{stats.medals}}</div><div class="text-xs text-slate-500">مدال ثبت‌شده</div></div></section>
      <section v-for="section in medalSections" :key="section.id" class="space-y-5 rounded-xl border border-slate-200 p-4"><h2 class="border-b pb-2 text-lg font-black">{{section.name}}</h2><div><div class="mb-3 flex items-center justify-between"><h3 class="font-bold">رده‌بندی تیم‌ها</h3><span class="text-xs text-slate-500">بر اساس مجموع امتیاز</span></div><div v-if="!section.teams.length" class="rounded-xl border border-dashed p-5 text-center text-sm text-slate-400">تیمی در این گروه نیست.</div><table v-else class="w-full border-collapse overflow-hidden text-sm"><thead><tr class="bg-slate-900 text-white"><th class="p-2">رتبه</th><th class="p-2 text-right">تیم</th><th class="p-2">امتیاز</th><th class="p-2">طلا</th><th class="p-2">نقره</th><th class="p-2">برنز</th><th class="p-2">برد / باخت</th></tr></thead><tbody><tr v-for="t in section.teams" :key="t.club" class="border-b border-slate-200" :class="t.rank<=3?'bg-amber-50':''"><td class="p-2 text-center font-bold">{{t.rank}}</td><td class="p-2 font-medium">{{t.club}}</td><td class="p-2 text-center font-black text-emerald-700">{{t.totalPoints}}</td><td class="p-2 text-center">{{t.gold}}</td><td class="p-2 text-center">{{t.silver}}</td><td class="p-2 text-center">{{t.bronze}}</td><td class="p-2 text-center">{{t.totalWins}} / {{t.totalLosses}}</td></tr></tbody></table></div><div><h3 class="mb-3 font-bold">مدال‌آوران به تفکیک وزن</h3><p v-if="!section.weights.length" class="rounded-xl border border-dashed p-5 text-center text-sm text-slate-500">هنوز مدالی در این گروه ثبت نشده است.</p><div v-for="weight in section.weights" :key="weight.weight" class="mb-4 break-inside-avoid"><h4 class="mb-2 rounded-lg bg-slate-100 px-3 py-2 font-bold">وزن {{weight.weight}}</h4><table class="w-full border-collapse text-sm"><thead><tr class="bg-slate-50"><th class="p-2">ردیف</th><th class="p-2 text-right">ورزشکار</th><th class="p-2 text-right">تیم / مربی</th><th class="p-2">امتیاز</th><th class="p-2">مدال‌ها</th></tr></thead><tbody><tr v-for="p in weight.medalists" :key="p.athleteId" class="border-b border-slate-200"><td class="p-2 text-center">{{p.medalRank}}</td><td class="p-2 font-medium">{{p.name}}</td><td class="p-2">{{p.club||'آزاد'}}</td><td class="p-2 text-center">{{p.totalPoints}}</td><td class="p-2 text-center">🥇{{p.gold}} · 🥈{{p.silver}} · 🥉{{p.bronze}}</td></tr></tbody></table></div></div></section>
      <footer class="border-t pt-3 text-center text-xs text-slate-400">سامانه مدیریت لیگ تکواندو · این گزارش به‌صورت خودکار از نتایج ثبت‌شده تولید شده است.</footer>
    </div>
  </div>
</template>
