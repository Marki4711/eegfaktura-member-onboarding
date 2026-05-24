// Test-data helpers for Playwright specs.
//
// Reuse this module to avoid the "fixed-string-collision" anti-pattern
// where every test uses the same email (`test@example.at`) and hits
// `email_already_exists`-Flakes after the first run. Each test gets a
// unique-per-run email via UUID-prefix.
//
// Convention: any e-mail used in a SUBMIT must be unique-per-run. Reads
// (filter the admin list by email) can use a fixed string if you're sure
// no other test populates it. When in doubt: unique.

import { randomUUID } from 'node:crypto'

// uniqueEmail returns "test-<uuid>@e2e.local". The `@e2e.local` TLD is
// reserved (RFC 6761) and cannot resolve — so a misconfigured mailer
// won't accidentally exfiltrate test data to a real recipient.
export function uniqueEmail(prefix = 'test'): string {
  return `${prefix}-${randomUUID()}@e2e.local`
}

// uniqueRef returns a unique reference number safe to use in form fields
// that need a stable user-provided string per test run (e.g. company
// register-number, member-name, comment field).
export function uniqueRef(prefix = 'ref'): string {
  return `${prefix}-${randomUUID().slice(0, 8)}`
}

// TEST_RC_NUMBER is the well-known test EEG seeded by the dev-seed script
// (`POST /api/admin/sync` against the operator's test cluster). Override
// via TEST_RC_NUMBER env var when running against a non-default seed.
export const TEST_RC_NUMBER = process.env.TEST_RC_NUMBER ?? 'RC123456'
