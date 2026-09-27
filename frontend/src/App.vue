<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useUiStore } from './stores/ui'
import { useTournamentStore } from './stores/tournament'
import { useLeagueStore } from './stores/league'
import HomePage from './components/HomePage.vue'
import TournamentPage from './components/TournamentPage.vue'
import LeaguePage from './components/League/LeaguePage.vue'
import TeamTournamentPage from './components/TeamTournoment/TeamTournamentPage.vue'
import { loadWebLeagues, loadWebTournaments, loginWeb, savedSession, saveSession } from './webApi'
import PublicOVR from './components/PublicOVR.vue'

const ui = useUiStore()
const tournamentStore = useTournamentStore()
const leagueStore = useLeagueStore()
const session = ref(savedSession())
const loading = ref(false)
const error = ref('')
const form = reactive({ email: '', password: '' })
const isPublicOVR = location.pathname.toLowerCase() === '/ovr'
function applyUrl(){
  const parts=location.pathname.split('/').filter(Boolean)
  if(parts[0]==='tournaments'&&parts[1]&&tournamentStore.tournaments.some(t=>t.id===parts[1])){tournamentStore.selectTournament(parts[1]);ui.page='tournament';ui.tab=(parts[2]||'athletes') as any;return}
  if(parts[0]==='leagues'&&parts[1]&&leagueStore.leagues.some(l=>l.id===parts[1])){leagueStore.selectLeague(parts[1]);ui.page='league';ui.leagueTab=(parts[2]||'setup') as any;return}
  ui.goHome()
}
function currentUrl(){if(ui.page==='tournament'&&tournamentStore.currentTournamentId)return`/tournaments/${tournamentStore.currentTournamentId}${ui.tab==='athletes'?'':`/${ui.tab}`}`;if(ui.page==='league'&&leagueStore.currentLeagueId)return`/leagues/${leagueStore.currentLeagueId}${ui.leagueTab==='setup'?'':`/${ui.leagueTab}`}`;return'/'}
async function hydrate(){loading.value=true;error.value='';try{const [items,leagues]=await Promise.all([loadWebTournaments(),loadWebLeagues()]);tournamentStore.tournaments.splice(0,tournamentStore.tournaments.length,...items as any);tournamentStore.save();leagueStore.replaceFromBackend(leagues);applyUrl()}catch(e:any){const message=e?.message||'دریافت اطلاعات ناموفق بود';if(/session expired|authentication required|invalid session/i.test(message)){saveSession(null);session.value=null;ui.goHome();error.value=''}else{error.value=message}}finally{loading.value=false}}
async function submitLogin(){loading.value=true;error.value='';try{session.value=await loginWeb(form.email,form.password);await hydrate()}catch(e:any){error.value=e?.message||'ورود ناموفق بود'}finally{loading.value=false}}
function logout(){saveSession(null);session.value=null;error.value='';ui.goHome()}
onMounted(()=>{if(session.value)hydrate()})
onMounted(()=>window.addEventListener('popstate',applyUrl))
watch([()=>ui.page,()=>ui.tab,()=>ui.leagueTab,()=>tournamentStore.currentTournamentId,()=>leagueStore.currentLeagueId],()=>{const url=currentUrl();if(location.pathname!==url)history.pushState({},'',url)})
</script>

<template>
  <PublicOVR v-if="isPublicOVR" />
  <div v-else-if="!session" class="flex min-h-screen items-center justify-center bg-slate-100 px-4 text-slate-800">
    <form class="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-7 shadow-xl" @submit.prevent="submitLogin">
      <div class="mb-6 text-center"><div class="text-5xl">🥋</div><h1 class="mt-3 text-xl font-bold">قرعه‌کشی مسابقات تکواندو</h1><p class="mt-2 text-sm text-slate-500">ورود به پنل هیئت برگزاری</p></div>
      <label class="mb-4 block text-sm font-medium">ایمیل<input v-model="form.email" class="input mt-1" type="email" required></label>
      <label class="mb-4 block text-sm font-medium">رمز عبور<input v-model="form.password" class="input mt-1" type="password" required></label>
      <p v-if="error" class="mb-3 rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ error }}</p>
      <button class="w-full rounded-lg bg-blue-600 py-2.5 font-medium text-white hover:bg-blue-700 disabled:opacity-60" :disabled="loading">{{ loading?'در حال ورود…':'ورود' }}</button>
    </form>
  </div>
  <div v-else class="min-h-screen bg-slate-100 text-slate-800">
    <header class="bg-slate-900 text-white shadow-lg">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <h1 class="text-xl font-bold">🥋 قرعه‌کشی مسابقات تکواندو</h1>
        <div class="flex items-center gap-2"><button v-if="ui.page !== 'home'" @click="ui.goHome()"
                class="rounded-lg bg-slate-700 px-4 py-2 text-sm transition hover:bg-slate-600">
          → بازگشت به لیست مسابقات
        </button><button @click="logout" class="rounded-lg border border-slate-600 px-3 py-2 text-sm hover:bg-slate-800">خروج</button></div>
      </div>
    </header>
    <main class="mx-auto max-w-6xl px-6 py-8">
      <div v-if="loading" class="mb-4 rounded-lg bg-blue-50 p-3 text-sm text-blue-700">در حال همگام‌سازی با سرور…</div>
      <div v-if="error" class="mb-4 rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>
      <HomePage v-if="ui.page === 'home'" />
      <TournamentPage v-if="ui.page === 'tournament'" />
      <TeamTournamentPage v-else-if="ui.page === 'teamTournament'" />
      <LeaguePage v-else-if="ui.page === 'league'" />
    </main>
  </div>
</template>
