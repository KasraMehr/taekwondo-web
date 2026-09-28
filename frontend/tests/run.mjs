import { build } from 'esbuild'
import { spawnSync } from 'node:child_process'
import { rmSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const output = fileURLToPath(new URL('./.generated.test.mjs', import.meta.url))
try {
  await build({ entryPoints: [fileURLToPath(new URL('./tournament.test.ts', import.meta.url))], outfile: output,
    bundle: true, platform: 'node', format: 'esm', packages: 'external', sourcemap: 'inline' })
  const result = spawnSync(process.execPath, ['--test', output], { stdio: 'inherit' })
  process.exitCode = result.status ?? 1
} finally { rmSync(output, { force: true }) }
