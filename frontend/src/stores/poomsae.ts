import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { webApi } from '../webApi'
import { copyScores, scoreKey } from '../utils/poomsae'
import type { DivisionInput, DivisionSummary, EventInput, PoomsaeDivision, PoomsaeEvent, PoomsaeOptions, ScoreDraft } from '../types/poomsae'

export const usePoomsaeStore = defineStore('poomsae', () => {
  const event = ref<PoomsaeEvent | null>(null)
  const divisions = ref<DivisionSummary[]>([])
  const division = ref<PoomsaeDivision | null>(null)
  const options = ref<PoomsaeOptions | null>(null)
  const busy = ref(false)
  const drafts = reactive<Record<string, ScoreDraft>>({})
  const dirtyCount = computed(() => Object.values(drafts).filter(d => scoreKey(d.value) !== scoreKey(d.base)).length)
  let generation = 0
  async function settings() { options.value = await webApi().call<PoomsaeOptions>('/poomsae-settings-options') }
  async function all<T>(path: string): Promise<T[]> {
    const result: T[] = []
    for (let offset = 0; ; offset += 200) { const page = await webApi().call<T[]>(`${path}?limit=200&offset=${offset}`); result.push(...page); if (page.length < 200) return result }
  }
  async function open(id: string, divisionId?: string) {
    const ticket = ++generation
    event.value = null; division.value = null; divisions.value = []
    const [e, ds] = await Promise.all([webApi().call<PoomsaeEvent>(`/poomsae-events/${id}`), all<DivisionSummary>(`/poomsae-events/${id}/divisions`), settings()])
    if (ticket !== generation) return
    event.value = e; divisions.value = ds
    const selected = divisionId || ds[0]?.id
    if (selected) await selectDivision(selected)
  }
  function divisionPath() { if (!event.value || !division.value) throw new Error('ابتدا یک رده انتخاب کنید.'); return `/poomsae-events/${event.value.id}/divisions/${division.value.id}` }
  async function selectDivision(id: string) {
    if (!event.value) return
    const ticket = ++generation; const eventId = event.value.id; division.value = null
    const d = await webApi().call<PoomsaeDivision>(`/poomsae-events/${eventId}/divisions/${id}`)
    if (ticket === generation) division.value = d
  }
  function accept(d: PoomsaeDivision) {
    division.value = d; const i = divisions.value.findIndex(v => v.id === d.id); if (i >= 0) divisions.value[i] = d
  }
  async function refreshDivision() { const path = divisionPath(); const d = await webApi().call<PoomsaeDivision>(path); if (division.value?.id === d.id) accept(d) }
  async function mutate(suffix: string, method: string, body?: unknown) {
    if (busy.value) throw new Error('لطفاً تا پایان ذخیرهٔ قبلی صبر کنید.')
    const path = divisionPath(); const revision = division.value!.revision; busy.value = true
    try { const d = await webApi().call<PoomsaeDivision>(path + suffix, method, body, { revision }); if (event.value?.id === d.eventId && division.value?.id === d.id) accept(d); return d } finally { busy.value = false }
  }
  async function saveEvent(input: EventInput) {
    const e = event.value; const result = await webApi().call<PoomsaeEvent>(e ? `/poomsae-events/${e.id}` : '/poomsae-events', e ? 'PUT' : 'POST', input, e ? { revision: e.revision } : undefined)
    event.value = result; return result
  }
  async function createDivision(input: DivisionInput) {
    const e = event.value!; const response = await webApi().call<{ division: PoomsaeDivision; eventRevision: number }>(`/poomsae-events/${e.id}/divisions`, 'POST', input, { revision: e.revision })
    e.revision = response.eventRevision; divisions.value.push(response.division); await selectDivision(response.division.id)
  }
  function draftFor(id: string) {
    const d = division.value!; const key = `${d.id}:${id}`; const entry = d.entries.find(a => a.id === id)!
    const value = copyScores(entry.scores); if (event.value?.rules.formSelection === 'division') { value.form1.code = d.form1Code; value.form2.code = d.form2Code }
    const existing = drafts[key]
    if (!existing || (!existing.conflict && scoreKey(existing.value) === scoreKey(existing.base) && scoreKey(existing.base) !== scoreKey(value))) drafts[key] = { value, base: copyScores(value), error: '', conflict: false }
    return drafts[key]
  }
  function clearDraft(id: string, divisionId = division.value?.id) { if (divisionId) delete drafts[`${divisionId}:${id}`] }
  function reset() { generation++; event.value = null; division.value = null; divisions.value = []; options.value = null; for (const key of Object.keys(drafts)) delete drafts[key] }
  return { event, divisions, division, options, busy, drafts, dirtyCount, settings, all, open, selectDivision, refreshDivision, mutate, saveEvent, createDivision, draftFor, clearDraft, reset, divisionPath }
})
