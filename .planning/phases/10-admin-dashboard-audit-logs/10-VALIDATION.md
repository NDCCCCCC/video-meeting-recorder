---
phase: 10
slug: admin-dashboard-audit-logs
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-24
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest (to be installed - Wave 0 gap) |
| **Config file** | frontend/vitest.config.ts (to be created) |
| **Quick run command** | `npm test` (to be configured) |
| **Full suite command** | `npm test:coverage` (to be configured) |
| **Estimated runtime** | ~30 seconds (after Wave 0 setup) |

**Current Status:** ⚠️ No test runner configured. Project has test files (*.test.ts, *.test.tsx) but no test runner configuration. This is a **Wave 0 gap**.

---

## Sampling Rate

- **After every task commit:** Manual browser testing (no automated quick run until Wave 0 complete)
- **After every plan wave:** Manual smoke test (dashboard loads, audit log table works, export generates file)
- **Before `/gsd-verify-work`:** Full manual verification of all user decisions (D-01 to D-39) in development environment
- **Max feedback latency:** Manual verification latency (depends on task complexity)

**Note:** This phase is primarily UI-focused (React components, visualizations). Testing requires:
1. Component test runner (Vitest or React Testing Library)
2. Mock API responses for dashboard stats and audit logs
3. Theme testing utilities for design token verification
4. Visual regression testing for chart rendering (optional)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | D-01 to D-12 (Dashboard stats, charts) | T-10-01 (Unauthorized access) | Admin role check via middleware | Manual | Manual browser test | ❌ W0 | ⬜ pending |
| 10-02-01 | 02 | 1 | D-13 to D-26 (Audit logs, filters, export) | T-10-02 (CSV injection) | CSV escape values starting with `=`, `+`, `-`, `@` | Manual | Export test with malicious data | ❌ W0 | ⬜ pending |
| 10-03-01 | 03 | 1 | D-27 to D-39 (Design tokens, loading, errors) | — | Theme tokens propagate to all components | Manual | Visual inspection across pages | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Before implementation begins, install and configure Vitest:

1. **Install dependencies:**
   ```bash
   cd frontend
   npm install --save-dev vitest @testing-library/react @testing-library/jest-dom @testing-library/user-event @vitejs/plugin-react
   ```

2. **Create `frontend/vitest.config.ts`:**
   ```typescript
   import { defineConfig } from 'vitest/config'
   import react from '@vitejs/plugin-react'

   export default defineConfig({
     plugins: [react()],
     test: {
       globals: true,
       environment: 'jsdom',
       setupFiles: './src/test/setup.ts',
     },
   })
   ```

3. **Create test setup file `frontend/src/test/setup.ts`:**
   ```typescript
   import '@testing-library/jest-dom'
   import { cleanup } from '@testing-library/react'
   import { afterEach } from 'vitest'

   afterEach(() => {
     cleanup()
   })
   ```

4. **Update `frontend/package.json` scripts:**
   ```json
   {
     "scripts": {
       "test": "vitest",
       "test:ui": "vitest --ui",
       "test:coverage": "vitest --coverage"
     }
   }
   ```

5. **Create initial test stubs:**
   - [ ] `frontend/src/pages/dashboard/__tests__/StatCards.test.tsx` — Dashboard statistics cards
   - [ ] `frontend/src/pages/dashboard/__tests__/ChartsSection.test.tsx` — Chart rendering
   - [ ] `frontend/src/pages/audit/__tests__/AuditTable.test.tsx` — Audit log table
   - [ ] `frontend/src/pages/audit/__tests__/DiffModal.test.tsx` — Diff view modal
   - [ ] `frontend/src/styles/__tests__/theme.test.ts` — Design token verification

**If Wave 0 is skipped:** Phase relies entirely on manual testing. Planner should include manual testing tasks for each feature (dashboard stats, audit log filters, diff view, export, design tokens).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dashboard chart rendering | D-07 to D-09 | Visual regression (charts need human verification) | 1. Open dashboard page. 2. Verify line chart shows trend data. 3. Verify column chart shows comparison data. 4. Verify pie chart shows distribution data. 5. Check tooltips appear on hover. |
| Audit log diff view | D-21 to D-23 | Complex UI interaction (side-by-side comparison) | 1. Open audit logs page. 2. Click "查看详情" on a log with changes. 3. Verify modal opens with side-by-side view. 4. Verify OldData on left, NewData on right. 5. Verify differences are highlighted with background colors. |
| CSV export functionality | D-24 | File download verification (browser behavior) | 1. Apply filters to audit logs. 2. Click export dropdown, select CSV. 3. Verify browser downloads `audit_logs.csv`. 4. Open file in Excel, verify data integrity. 5. Test with malicious data (values starting with `=`, `+`, `-`, `@`). |
| Design token consistency | D-27 to D-39 | Visual inspection across multiple pages | 1. Open dashboard, audit, and existing pages. 2. Verify consistent spacing (use designTokens.spacing.*). 3. Verify consistent colors (primary, success, warning, error). 4. Verify consistent border radius on cards and modals. |
| Loading states (Skeleton) | D-34 to D-36 | Perceived performance (needs human evaluation) | 1. Open dashboard page with slow network (DevTools throttling). 2. Verify Skeleton appears before stats load. 3. Verify smooth transition from Skeleton to content. 4. Repeat for audit logs table. |
| Error handling (Toast) | D-37 to D-39 | User experience (needs human evaluation) | 1. Trigger API error (disconnect backend or invalid request). 2. Verify Toast notification appears with error message. 3. Verify message is user-friendly (not technical stack trace). 4. Verify error doesn't break UI. |

---

## Security Validation

### Threat Mitigations to Verify

| Threat ID | Pattern | Mitigation | Verification Step |
|-----------|---------|------------|-------------------|
| T-10-01 | Unauthorized dashboard access | Backend permission check: `middleware.RequirePermission("dashboard:view")` | 1. Login as non-admin user. 2. Attempt to access `/dashboard`. 3. Verify 403 Forbidden or redirect. |
| T-10-02 | CSV injection (formula injection) | Escape audit log values starting with `=`, `+`, `-`, `@` | 1. Create audit log with OldData containing `=SUM(1+1)`. 2. Export to CSV. 3. Open in Excel. 4. Verify formula does NOT execute (value appears as text). |
| T-10-03 | Information disclosure via audit logs | Apply data_scope filtering, redact sensitive fields | 1. Login as regular user (not admin). 2. Access audit logs page. 3. Verify only own logs visible. 4. Verify OldData/NewData don't show other users' sensitive data. |
| T-10-04 | Denial of service via large export | Enforce max export limit (10k rows) | 1. Create >10k audit log entries (or mock large dataset). 2. Attempt export. 3. Verify export returns max 10k rows or shows error message. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (Vitest setup, test stubs)
- [ ] No watch-mode flags (manual testing only for this phase)
- [ ] Feedback latency acceptable (manual verification latency)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

---

*Validation strategy created: 2026-04-24*
