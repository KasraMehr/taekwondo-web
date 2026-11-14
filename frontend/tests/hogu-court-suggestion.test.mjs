import assert from 'node:assert/strict'
import test from 'node:test'
import { suggestHoguCourts } from '../src/utils/hoguCourtSuggestion.ts'

test('one pair of a size stays on one court', () => {
  const result = suggestHoguCourts([
    { category: '-40', size: '1', matches: 8 },
    { category: '-44', size: '1', matches: 5 },
    { category: '-58', size: '2', matches: 20 },
  ], 3, { '1': 1, '2': 1 }, true)

  assert.deepEqual(result['-40'], result['-44'])
  assert.equal(result['-40'].length, 1)
  assert.equal(result['-58'].length, 1)
  assert.notDeepEqual(result['-40'], result['-58'])
})

test('a busy weight uses two courts only with two available pairs', () => {
  const weights = [{ category: '-58', size: '2', matches: 20 }]
  assert.equal(suggestHoguCourts(weights, 4, { '2': 1 }, true)['-58'].length, 1)
  assert.equal(suggestHoguCourts(weights, 4, { '2': 2 }, true)['-58'].length, 2)
  assert.equal(suggestHoguCourts(weights, 1, { '2': 2 }, true)['-58'].length, 1)
})

test('missing required hogu size is never assigned a court', () => {
  const result = suggestHoguCourts([{ category: '-58', size: '2', matches: 3 }], 2, { '2': 0 }, true)
  assert.deepEqual(result, {})
})
