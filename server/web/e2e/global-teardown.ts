import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'
import { execSync } from 'child_process'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

async function globalTeardown() {
  if (!process.env.COVERAGE) return

  const root = path.resolve(__dirname, '../../../..')
  const coverDir = path.join(root, 'coverage/backend')

  if (!fs.existsSync(coverDir)) {
    console.log('No coverage data directory found — skipping coverage merge.')
    return
  }

  const covFiles = fs.readdirSync(coverDir).filter(f => f.endsWith('.cov'))
  if (covFiles.length === 0) {
    console.log('No .cov files found — skipping coverage merge.')
    return
  }

  console.log(`\n📊 Merging ${covFiles.length} coverage data files...`)

  try {
    const textFmtOut = path.join(root, 'coverage/coverage.out')
    execSync(`go tool covdata textfmt -i=${coverDir} -o=${textFmtOut}`, {
      cwd: root,
      stdio: 'inherit',
    })
    console.log(`✓ Text coverage: ${textFmtOut}`)

    const htmlOut = path.join(root, 'coverage/coverage.html')
    execSync(`go tool cover -html=${textFmtOut} -o=${htmlOut}`, {
      cwd: root,
      stdio: 'inherit',
    })
    console.log(`✓ HTML coverage: ${htmlOut}`)
  } catch (err) {
    console.error('Coverage processing failed:', (err as Error).message)
  }
}

export default globalTeardown
