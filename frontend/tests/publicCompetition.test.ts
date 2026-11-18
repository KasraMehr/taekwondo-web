import test from 'node:test'
import assert from 'node:assert/strict'
import { competitionPeriod, tehranDate } from '../src/utils/publicCompetition'

test('today follows Tehran midnight rather than UTC or the visitor timezone', () => {
  assert.equal(tehranDate(new Date('2026-11-18T20:29:59Z')), '2026-11-18')
  assert.equal(tehranDate(new Date('2026-11-18T20:30:00Z')), '2026-11-19')
  assert.equal(competitionPeriod('2026-11-19', '2026-11-19'), 'today')
  assert.equal(competitionPeriod('2026-11-19T00:00:00Z', '2026-11-19'), 'today')
  assert.equal(competitionPeriod('2026-11-20', '2026-11-19'), 'upcoming')
  assert.equal(competitionPeriod('2026-11-18', '2026-11-19'), 'past')
})
