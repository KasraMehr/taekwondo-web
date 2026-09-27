import { Api, auth, organizations, type Session } from './api'

const KEY = 'tkd_session'

export function savedSession(): Session | null {
  try { return JSON.parse(localStorage.getItem(KEY) || 'null') } catch { return null }
}

export function saveSession(value: Session | null) {
  if (value) localStorage.setItem(KEY, JSON.stringify(value))
  else localStorage.removeItem(KEY)
}

export async function loginWeb(email: string, password: string) {
  const result = await auth('login', { email, password })
  const orgs = await organizations(result.token)
  if (!orgs.length) throw new Error('این حساب عضو هیچ سازمانی نیست')
  const session = { token: result.token, userId: result.userId, orgId: orgs[0].id }
  saveSession(session)
  return session
}

export function webApi() {
  const session = savedSession()
  if (!session) throw new Error('ابتدا وارد شوید')
  return new Api(session)
}

export async function loadWebTournaments() {
  const api = webApi()
  const list = await api.call<any[]>('/tournaments?limit=200')
  return Promise.all(list.map(item => api.call(`/tournaments/${item.id}`)))
}

export async function loadWebLeagues() {
  const api = webApi()
  const leagues = await api.call<any[]>('/leagues')
  return Promise.all(leagues.map(async league => ({
    league,
    standings: await api.call<any>(`/leagues/${league.id}/standings`),
  })))
}
