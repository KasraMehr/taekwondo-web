// Generate portable fixtures from actual TypeScript code, with no copied algorithm.
import { build } from 'esbuild'
import { writeFileSync, rmSync } from 'node:fs'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { resolve } from 'node:path'

const root = fileURLToPath(new URL('../', import.meta.url))
const output = fileURLToPath(new URL('./.parity.mjs', import.meta.url))
try {
  await build({ stdin: { contents: `export * from './src/utils/bracket'; export * from './src/utils/weighIn'; export * from './src/utils/scoring'; export * from './src/data/categories'`, resolveDir: root }, outfile: output, bundle: true, platform: 'node', format: 'esm' })
  const source = await import(pathToFileURL(output).href)
  const fixtures = { weights: [], brackets: [], rounds: [], categories: source.WEIGHT_CATEGORIES }
  for (const [category, values] of [['-58', [54, 54.001, 58, 58.2, 58.201]], ['+87', [87, 87.001]]]) {
    for (const weight of values) {
      const result = source.checkWeight(weight, category, source.weightOptions('بزرگسالان', 'male'))
      fixtures.weights.push({ category, weight, ok: result.ok, tolerance: result.withTolerance })
    }
  }
  for (const n of [2, 3, 5, 8, 16]) {
    const athletes = Array.from({ length: n }, (_, i) => ({ id: String(i + 1), name: String(i + 1), club: String(i + 1), weightCategory: '-54', ranking: i + 1 }))
    const matches = source.generateBracket(athletes, '-54', 1, 'ranked')
    const index = new Map(matches.map((m, i) => [m.id, i]))
    fixtures.brackets.push({ count: n, matches: matches.map(m => ({ a: m.athlete1Id, b: m.athlete2Id, winner: m.winnerId ?? null, round: m.round, side: m.side, bye: !!m.isBye, index: m.bracketIndex, next: m.nextMatchId ? index.get(m.nextMatchId) : -1, slot: m.nextSlot ?? null })) })
  }
  for (const scores of [{ blue: { bodyKick: 2 } }, { red: { headKick: 5 } }, { blue: { gamJeom: 5 } }, { blue: { punch: 2 }, red: { bodyKick: 1 } }, {}]) {
    const round = source.emptyRound(1); Object.assign(round.blue, scores.blue); Object.assign(round.red, scores.red)
    fixtures.rounds.push({ round, outcome: source.resolveRound(round) })
  }
  const destination = process.argv[2] ? resolve(process.argv[2]) : fileURLToPath(new URL('./desktop-parity.json', import.meta.url))
  writeFileSync(destination, JSON.stringify(fixtures, null, 2) + '\n')
  console.log(`Wrote parity fixtures: ${destination}`)
} finally { rmSync(output, { force: true }) }
