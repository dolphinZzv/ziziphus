import { test as base, expect } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const WEB_ROOT = path.resolve(__dirname, '../../..')
const NYC_OUTPUT = path.join(WEB_ROOT, '.nyc_output')

/** Transform a Vite-served URL to a file:// URL that c8 understands. */
function toFileUrl(viteUrl: string): string | null {
  if (!viteUrl.startsWith('http://localhost:5173/src/')) return null
  const rel = viteUrl.replace('http://localhost:5173/', '')
  const abs = path.resolve(WEB_ROOT, rel)
  return 'file://' + abs
}

// Extend base test with coverage tracking (c8-compatible V8 output)
export const test = base.extend<{
  collectCoverage: void
}>({
  collectCoverage: [async ({ page }, use) => {
    await page.coverage.startJSCoverage()
    await use()
    const coverage = await page.coverage.stopJSCoverage()

    if (coverage.length === 0) return

    // Filter to our source files only
    const srcCoverage = coverage
      .filter(e => e.url.startsWith('http://localhost:5173/src/'))
      .map(e => ({
        scriptId: '0',
        url: toFileUrl(e.url)!,
        functions: e.functions,
        source: e.text,
      }))

    if (srcCoverage.length === 0) return

    fs.mkdirSync(NYC_OUTPUT, { recursive: true })

    const testName = base.info().title.replace(/[^a-zA-Z0-9]/g, '_')
    const filePath = path.join(NYC_OUTPUT, `coverage-${testName}.json`)
    fs.writeFileSync(filePath, JSON.stringify({ result: srcCoverage }, null, 2))
  }, { auto: true }],
})

export { expect }
