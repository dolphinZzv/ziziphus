import { FullConfig } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

async function globalSetup(_config: FullConfig) {
  const coverageDir = path.resolve(__dirname, '../../../../coverage')
  if (fs.existsSync(coverageDir)) {
    fs.rmSync(coverageDir, { recursive: true, force: true })
  }
  fs.mkdirSync(path.join(coverageDir, 'backend'), { recursive: true })
  console.log(`Coverage directory: ${coverageDir}`)
}

export default globalSetup
