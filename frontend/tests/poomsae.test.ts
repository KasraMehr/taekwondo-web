import test from 'node:test'
import assert from 'node:assert/strict'
import { createPinia, setActivePinia } from 'pinia'
import { Api, ApiError } from '../src/api'
import { usePoomsaeStore } from '../src/stores/poomsae'
import { copyScores, hundredths, isoFromJalali, jalaliFromIso, normalizeScore, sum } from '../src/utils/poomsae'
import type { PoomsaeDivision, PoomsaeEvent, Scores } from '../src/types/poomsae'

const scores = (): Scores => ({ form1: { code: 2, accuracy: null, presentation: null }, form2: { code: 3, accuracy: null, presentation: null } })
const event: PoomsaeEvent = { id: 'event', name: 'جام', date: '2026-10-04', revision: 1, createdAt: '', rules: { accuracyMax: '3.00', presentationMax: '7.00', precision: 2, tiePolicy: 'shared', formSelection: 'division', allowRepeatedForm: false, version: 'test' } }
function division(id = 'division'): PoomsaeDivision {
  return { id, eventId: event.id, name: 'زیر ۱۲', competitionType: 'individual', gender: 'male', belt: '', birthDateFrom: '2014-01-01', birthDateTo: null, allowedFormCodes: [2, 3], form1Code: 2, form2Code: 3, status: 'drawn', everScored: false, revision: 4, entries: [{ id: 'athlete', athleteId: 'profile', firstName: 'علی', lastName: 'نمونه', teamName: 'تیم', birthDate: '2016-01-01', gender: 'male', belt: '', status: 'active', updatedAt: '', scores: scores() }], draws: [], results: [], standings: { revision: 4, final: false, ranked: [], unranked: [] } }
}
test('poomsae handles Persian/Arabic digits, decimal precision and missing values exactly', () => {
  assert.equal(normalizeScore('۲٫۵', '3.00'), '2.50')
  assert.equal(normalizeScore('٢.٦٠', '3.00'), '2.60')
  assert.equal(sum('۰٫۱۰', '۰٫۲۰'), '0.30')
  assert.equal(sum('2.50', '6.00', '2.60', '6.10'), '17.20')
  assert.equal(sum('0', '0', '0', '0'), '0.00')
  assert.equal(sum('0', null), null)
  assert.equal(normalizeScore('', '3.00'), null)
  for (const bad of ['-1', 'NaN', '1e2', '1.234', '3.01', '1,2']) assert.throws(() => normalizeScore(bad, '3.00'))
  assert.equal(hundredths('0002.01'), 201)
})
test('poomsae dates round-trip and reject invalid Persian calendar dates', () => {
  assert.equal(isoFromJalali('۱۴۰۵/۰۷/۱۲'), '2026-10-04')
  assert.equal(jalaliFromIso('2026-10-04'), '۱۴۰۵/۰۷/۱۲')
  assert.equal(isoFromJalali('۱۳۹۹/۱۲/۳۰'), '2021-03-20')
  assert.throws(() => isoFromJalali('۱۴۰۰/۱۲/۳۰'))
  assert.throws(() => isoFromJalali('۱۴۰۵/۱۳/۰۱'))
})
test('API sends optimistic revision and preserves conflict HTTP status', async () => {
  const original = globalThis.fetch
  globalThis.fetch = async (_url, init) => {
    assert.equal((init!.headers as Record<string, string>)['If-Match'], '"4"')
    assert.equal(init!.method, 'PUT')
    return new Response(JSON.stringify({ message: 'resource changed' }), { status: 409 })
  }
  try { await assert.rejects(new Api({ token: 'test', userId: 'user', orgId: 'org' }).call('/poomsae-events/event', 'PUT', {}, { revision: 4 }), e => e instanceof ApiError && e.status === 409) } finally { globalThis.fetch = original }
})
test('score drafts survive division switches and conflicting server refreshes', () => {
  setActivePinia(createPinia()); const store = usePoomsaeStore(); store.event = event; store.division = division()
  const draft = store.draftFor('athlete'); draft.value.form1.accuracy = '۲٫۵۰'
  assert.equal(store.dirtyCount, 1)
  assert.equal(store.division.entries[0].scores.form1.accuracy, null)
  store.division = division('other'); assert.equal(store.draftFor('athlete').value.form1.accuracy, null)
  store.division = division(); store.division.entries[0].scores.form1.accuracy = '2.00'
  assert.equal(store.draftFor('athlete').value.form1.accuracy, '۲٫۵۰')
  assert.equal(store.draftFor('athlete').base.form1.accuracy, null)
  store.clearDraft('athlete'); assert.equal(store.draftFor('athlete').value.form1.accuracy, '2.00')
  assert.equal(store.dirtyCount, 0)
  store.division.entries[0].scores.form1.accuracy = '2.10'
  assert.equal(store.draftFor('athlete').value.form1.accuracy, '2.10')
})
test('copying saved scores does not mutate their correction audit or inputs', () => {
  const saved = scores(); saved.reason = 'correction'; const draft = copyScores(saved); draft.form1.accuracy = '1.00'
  assert.equal(saved.form1.accuracy, null); assert.equal(saved.reason, 'correction'); assert.equal(draft.reason, '')
})
