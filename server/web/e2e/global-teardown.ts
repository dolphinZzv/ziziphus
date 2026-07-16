import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'
import { execSync } from 'child_process'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

async function globalTeardown() {
  if (!process.env.COVERAGE) return

  const webRoot = path.resolve(__dirname, '..')
  const root = path.resolve(__dirname, '../../../..')

  // --- Go backend coverage ---
  const coverDir = path.join(root, 'coverage/backend')
  const goCovFiles = fs.existsSync(coverDir) ? fs.readdirSync(coverDir).filter(f => f.endsWith('.cov')) : []

  if (goCovFiles.length > 0) {
    console.log(`\n📊 Merging ${goCovFiles.length} Go coverage files...`)
    try {
      const textFmtOut = path.join(root, 'coverage/coverage.out')
      execSync(`go tool covdata textfmt -i=${coverDir} -o=${textFmtOut}`, {
        cwd: root,
        stdio: 'inherit',
      })
      console.log(`✓ Go text coverage: ${textFmtOut}`)

      const htmlOut = path.join(root, 'coverage/coverage.html')
      execSync(`go tool cover -html=${textFmtOut} -o=${htmlOut}`, {
        cwd: root,
        stdio: 'inherit',
      })
      console.log(`✓ Go HTML coverage: ${htmlOut}`)
    } catch (err) {
      console.error('Go coverage processing failed:', (err as Error).message)
    }
  }

  // --- Frontend JS coverage (V8 → lcov via c8) ---
  const nycOutputDir = path.join(webRoot, '.nyc_output')
  if (fs.existsSync(nycOutputDir)) {
    const v8Files = fs.readdirSync(nycOutputDir).filter(f => f.endsWith('.json'))
    if (v8Files.length > 0) {
      console.log(`\n📊 Generating frontend lcov from ${v8Files.length} V8 coverage files...`)
      try {
        // c8 report reads from .nyc_output/ in cwd and writes to --report-dir
        const frontendCovDir = path.join(root, 'coverage/frontend')
        fs.mkdirSync(frontendCovDir, { recursive: true })
        execSync(
          `npx c8 report --reporter=lcov --reporter=text --report-dir=${frontendCovDir}`,
          { cwd: webRoot, stdio: 'inherit' },
        )
        console.log(`✓ Frontend lcov: ${path.join(frontendCovDir, 'lcov.info')}`)
      } catch (err) {
        console.error('Frontend coverage processing failed:', (err as Error).message)
      } finally {
        // Clean up raw V8 data
        fs.rmSync(nycOutputDir, { recursive: true, force: true })
      }
    }
  }
}

export default globalTeardown
