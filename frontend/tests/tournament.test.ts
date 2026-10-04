import test from 'node:test'
import './poomsae.test'
import assert from 'node:assert/strict'
import { createPinia, setActivePinia } from 'pinia'
import { generateBracket, recordResult } from '../src/utils/bracket'
import { checkWeight, emptyWeighIn, recordAttempt, migrateAthleteWeighIn, sign } from '../src/utils/weighIn'
import { emptyRound, resolveMatch } from '../src/utils/scoring'
import { useTournamentStore } from '../src/stores/tournament'
import type { Athlete } from '../src/types'
import { tournamentSeedsForGroup } from '../src/utils/leagueRoster'

const categories = ['-54', '-58', '-63', '-68', '-74', '-80', '-87', '+87']
const athlete = (i: number): Athlete => ({ id: crypto.randomUUID(), name: `Athlete ${i}`, club: `Club ${i}`, weightCategory: '-54', ranking: i })

test('weight boundaries agree with the desktop contract', () => {
  for (const [weight, ok] of [[54, false], [54.001, true], [58, true], [58.2, true], [58.201, false]] as const)
    assert.equal(checkWeight(weight, '-58', categories).ok, ok)
  assert.equal(checkWeight(87, '+87', categories).ok, false)
  assert.equal(checkWeight(87.001, '+87', categories).ok, true)
  assert.equal(checkWeight(58.2, '-58', categories).withTolerance, true)
})
test('league roster, including inherited club group, is the tournament source of truth', () => {
  const league: any = { clubs: [{ id: 'c1', name: 'Club A', groupId: 'g1' }], athletes: [
    { id: 'la1', name: 'Active', clubId: 'c1', club: 'old name', groupId: null, isActive: true, weightCategory: '-54' },
    { id: 'la2', name: 'Inactive', clubId: 'c1', groupId: null, isActive: false, weightCategory: '-58' },
    { id: 'la3', name: 'No weight', clubId: 'c1', groupId: null, isActive: true, weightCategory: null },
  ] }
  assert.deepEqual(tournamentSeedsForGroup(league, 'g1'), [{ sourceLeagueAthleteId: 'la1', name: 'Active', club: 'Club A', weightCategory: '-54', ranking: undefined }])
})
test('invalid inputs do not consume weigh-in attempts', () => {
  const initial = emptyWeighIn()
  for (const weight of [NaN, Infinity, -1, 0]) assert.equal(recordAttempt(initial, weight, '-58', categories), initial)
  assert.equal(recordAttempt(initial, 58, '-999', categories), initial)
  const first = recordAttempt(initial, 60, '-58', categories)
  assert.equal(first.status, 'pending')
  const second = recordAttempt(first, 60, '-58', categories)
  assert.equal(second.status, 'failed')
  assert.equal(recordAttempt(second, 58, '-58', categories), second)
})
test('legacy approved weigh-ins survive migration without invented measurements', () => {
  const migrated = migrateAthleteWeighIn({ ...athlete(1), weighedIn: true })
  assert.equal(migrated.weighIn?.status, 'passed')
  assert.equal(migrated.weighIn?.weightKg, null)
  assert.deepEqual(migrated.weighIn?.attempts, [])
  const conflict = migrateAthleteWeighIn({ ...athlete(2), weighedIn: true, weighIn: emptyWeighIn() })
  assert.equal(conflict.weighedIn, false)
})
test('blank signatures do not consume the signature slot', () => {
  const passed = recordAttempt(emptyWeighIn(), 54, '-54', categories)
  assert.equal(sign(passed, '   '), passed)
})
test('seed order, automatic byes and single final shape', () => {
  const athletes = Array.from({ length: 8 }, (_, i) => athlete(i + 1))
  const bracket = generateBracket(athletes, '-54', 1, 'ranked')
  const ranking = new Map(athletes.map(a => [a.id, a.ranking]))
  assert.deepEqual(bracket.filter(m => m.round === 1).flatMap(m => [ranking.get(m.athlete1Id!), ranking.get(m.athlete2Id!)]), [1, 8, 5, 4, 3, 6, 7, 2])
  assert.equal(generateBracket(athletes.slice(0, 2), '-54', 1, 'ranked')[0].side, 'final')
  for (let n = 2; n <= 33; n++) {
    const entries = Array.from({ length: n }, (_, i) => athlete(i + 1))
    const matches = generateBracket(entries, '-54', 1, 'ranked')
    assert.equal(matches.filter(m => !m.isBye).length, n - 1)
    for (const bye of matches.filter(m => m.isBye)) {
      assert.equal(bye.status, 'completed'); assert.equal(bye.court, 0); assert.equal(bye.order, 0); assert.ok(bye.winnerId)
    }
  }
})
test('an unrelated athlete cannot win; completed descendants prevent undo', () => {
  const entries = Array.from({ length: 4 }, (_, i) => athlete(i + 1))
  const matches = generateBracket(entries, '-54', 1, 'ranked')
  assert.equal(recordResult(matches, matches[0].id, 'unrelated'), matches)
  let result = recordResult(matches, matches[0].id, matches[0].athlete1Id!)
  result = recordResult(result, matches[1].id, matches[1].athlete1Id!)
  const final = result.find(m => m.side === 'final')!
  result = recordResult(result, final.id, final.athlete2Id!)
  assert.equal(recordResult(result, matches[0].id, matches[0].athlete1Id!), result)
})
test('rounds beyond the configured maximum cannot decide the match', () => {
  const rounds = Array.from({ length: 5 }, (_, i) => emptyRound(i + 1))
  rounds[3].blue.punch = 1; rounds[4].blue.punch = 1
  assert.equal(resolveMatch(rounds).decided, false)
})

function storeFixture() {
  const values = new Map<string, string>()
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: {
    getItem: (key: string) => values.get(key) ?? null, setItem: (key: string, value: string) => values.set(key, value),
  } })
  setActivePinia(createPinia())
  const store = useTournamentStore()
  const tournament = store.createTournament('Test', '2026-10-01', 2, 'male', 'بزرگسالان')
  return { store, tournament }
}
test('store rejects a weight-category edit after a weigh-in', () => {
  const { store, tournament } = storeFixture()
  assert.deepEqual(store.addAthlete(tournament.id, { name: 'A', club: 'Club', weightCategory: '-54' }), {})
  const a = tournament.athletes[0]
  assert.equal(store.recordWeighIn(tournament.id, a.id, 54).error, undefined)
  assert.ok(store.updateAthlete(tournament.id, { ...a, weightCategory: '-58' }).error)
  assert.equal(tournament.athletes[0].weightCategory, '-54')
})
test('store prevents deleting a bracket participant', () => {
  const { store, tournament } = storeFixture()
  for (let i = 0; i < 2; i++) {
    store.addAthlete(tournament.id, { name: `A ${i}`, club: '', weightCategory: '-54' })
    store.recordWeighIn(tournament.id, tournament.athletes[i].id, 54)
  }
  store.drawBracket(tournament.id, 'ranked')
  assert.ok(store.removeAthlete(tournament.id, tournament.athletes[0].id).error)
  assert.equal(tournament.athletes.length, 2)
})

test('store refuses partial weigh-in draws and simultaneous starts on one court', () => {
  const { store, tournament } = storeFixture()
  for (let i = 0; i < 4; i++) store.addAthlete(tournament.id, { name: `A ${i}`, club: `Club ${i}`, weightCategory: '-54' })
  for (let i = 0; i < 2; i++) store.recordWeighIn(tournament.id, tournament.athletes[i].id, 54)
  assert.ok(store.drawBracket(tournament.id)?.error)
  assert.equal(tournament.matches.length, 0)
  for (let i = 2; i < 4; i++) store.recordWeighIn(tournament.id, tournament.athletes[i].id, 54)
  store.drawBracket(tournament.id, 'ranked')
  const games = tournament.matches.filter(m => m.round === 1)
  store.startMatch(tournament.id, games[0].id)
  store.startMatch(tournament.id, games[1].id)
  assert.equal(tournament.matches.filter(m => m.status === 'ongoing').length, 1)
})

test('store preserves a completed final when correcting an earlier result', () => {
  const { store, tournament } = storeFixture()
  for (let i = 0; i < 4; i++) {
    store.addAthlete(tournament.id, { name: `A ${i}`, club: `Club ${i}`, weightCategory: '-54' })
    store.recordWeighIn(tournament.id, tournament.athletes[i].id, 54)
  }
  store.drawBracket(tournament.id, 'ranked')
  const source = tournament.matches[0]
  for (const match of tournament.matches.filter(m => m.round === 1))
    assert.equal(store.setMatchResult(tournament.id, match.id, { winType: 'WDR', winnerId: match.athlete1Id }).error, undefined)
  const final = tournament.matches.find(m => m.side === 'final')!
  store.setMatchResult(tournament.id, final.id, { winType: 'WDR', winnerId: final.athlete2Id })
  const before = JSON.stringify(tournament.matches)
  assert.ok(store.clearMatchResult(tournament.id, source.id).error)
  assert.ok(store.setMatchResult(tournament.id, source.id, { winType: 'WDR', winnerId: source.athlete2Id }).error)
  assert.equal(JSON.stringify(tournament.matches), before)
})
