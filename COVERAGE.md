
## Collecting coverage during E2E

### Frontend JS coverage
Every test using `e2e/fixtures/coverage.ts` automatically collects JS coverage:
```bash
npx playwright test --reporter=list
# Coverage data → /coverage/frontend/*.json
# Coverage report → /coverage/frontend-report.json
```

### Backend Go coverage (from E2E)
```bash
cd server && bash web/scripts/coverage-e2e.sh
```

| Type | Tool | Status |
|------|------|--------|
| Go unit tests | `go test ./...` | ✅ 69.9% |
| Go coverage from E2E | `GOCOVERDIR` + script | ✅ |
| Frontend JS coverage | Playwright `startJSCoverage()` | ✅ ~45% |

## Rules for adding tests
1. **Every new API endpoint** → Go unit test + API e2e test
2. **Every new UI feature** → Playwright e2e test with coverage fixture
3. **MFA/Email flows** → unit test + API e2e + UI e2e
4. **Before commit:**
   ```bash
   cd server && go test ./... -count=1
   cd web && npx playwright test --reporter=list
   ```
