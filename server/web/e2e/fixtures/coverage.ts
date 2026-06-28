import { test as base, expect } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// Extend base test with coverage tracking
export const test = base.extend<{
  collectCoverage: void
}>({
  collectCoverage: [async ({ page }, use) => {
    await page.coverage.startJSCoverage()
    await use()
    const coverage = await page.coverage.stopJSCoverage()

    if (coverage.length > 0) {
      const coverageDir = path.resolve(__dirname, '../../../../../coverage/frontend')
      fs.mkdirSync(coverageDir, { recursive: true })

      const testName = base.info().title.replace(/[^a-zA-Z0-9]/g, '_')
      const filePath = path.join(coverageDir, `${testName}.json`)

      const aggregated = coverage.map(entry => ({
        url: entry.url,
        functions: entry.functions?.map(f => ({
          name: f.functionName,
          ranges: f.ranges?.filter(r => r.count > 0).length || 0,
          total: f.ranges?.length || 0,
        })),
        totalBytes: entry.text?.length || 0,
      }))

      fs.writeFileSync(filePath, JSON.stringify(aggregated, null, 2))
    }
  }, { auto: true }],
})

export { expect }
