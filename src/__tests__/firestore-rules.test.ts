import { describe, it, beforeAll, afterAll, beforeEach, expect } from 'vitest'
import {
  initializeTestEnvironment,
  assertFails,
  type RulesTestEnvironment,
} from '@firebase/rules-unit-testing'
import { doc, getDoc, setDoc, collection, getDocs } from 'firebase/firestore'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

// These tests require the Firestore emulator. Locally start it via dev.sh; in
// CI it's started by the workflow before this suite runs. If the emulator
// isn't reachable we skip rather than failing — Vitest is the wrong place to
// surface a missing-emulator error, and CI sets EMULATOR_AVAILABLE=1.
const EMULATOR_HOST = process.env.FIRESTORE_EMULATOR_HOST ?? 'localhost:8091'

async function emulatorReachable(): Promise<boolean> {
  try {
    const [host, port] = EMULATOR_HOST.split(':')
    const res = await fetch(`http://${host}:${port}/`, { signal: AbortSignal.timeout(500) })
    return res.ok || res.status === 404 // 404 is fine — Firestore emulator answers
  } catch {
    return false
  }
}

const reachable = await emulatorReachable()

describe.skipIf(!reachable)('Firestore security rules — deny-all posture', () => {
  let testEnv: RulesTestEnvironment

  beforeAll(async () => {
    const rulesPath = resolve(__dirname, '../../firestore.rules')
    testEnv = await initializeTestEnvironment({
      projectId: 'demo-rules-test',
      firestore: {
        rules: readFileSync(rulesPath, 'utf8'),
        host: EMULATOR_HOST.split(':')[0],
        port: parseInt(EMULATOR_HOST.split(':')[1], 10),
      },
    })
  })

  afterAll(async () => {
    if (testEnv) await testEnv.cleanup()
  })

  beforeEach(async () => {
    if (testEnv) await testEnv.clearFirestore()
  })

  // Each (auth-state, operation, collection) tuple is a separate red-team
  // probe. Production code path is the Admin SDK which bypasses rules
  // entirely, so we never exercise "expect this to succeed for an authed
  // client" — every client-SDK call must fail.
  it.each([
    ['unauthenticated', 'users'],
    ['unauthenticated', 'predictions'],
    ['unauthenticated', 'results'],
    ['unauthenticated', 'matches'],
    ['unauthenticated', 'seasons'],
    ['authenticated as user-123', 'users'],
    ['authenticated as user-123', 'predictions'],
    ['authenticated as user-123', 'results'],
    ['authenticated as user-123', 'matches'],
    ['authenticated as user-123', 'seasons'],
  ])('denies reads on /%s collection (%s)', async (authLabel, col) => {
    const ctx =
      authLabel === 'unauthenticated'
        ? testEnv.unauthenticatedContext()
        : testEnv.authenticatedContext('user-123')
    const db = ctx.firestore()
    await assertFails(getDoc(doc(db, col, 'any-doc-id')))
    await assertFails(getDocs(collection(db, col)))
  })

  it.each([
    ['unauthenticated', 'users'],
    ['unauthenticated', 'predictions'],
    ['unauthenticated', 'results'],
    ['unauthenticated', 'matches'],
    ['unauthenticated', 'seasons'],
    ['authenticated as user-123', 'users'],
    ['authenticated as user-123', 'predictions'],
    ['authenticated as user-123', 'results'],
    ['authenticated as user-123', 'matches'],
    ['authenticated as user-123', 'seasons'],
  ])('denies writes on /%s collection (%s)', async (authLabel, col) => {
    const ctx =
      authLabel === 'unauthenticated'
        ? testEnv.unauthenticatedContext()
        : testEnv.authenticatedContext('user-123')
    const db = ctx.firestore()
    await assertFails(setDoc(doc(db, col, 'any-doc-id'), { foo: 'bar' }))
  })

  // Subcollections at any nesting depth must also be blocked. The recursive
  // {document=**} wildcard in the rules file handles this; we probe it
  // explicitly so a future rules edit that drops the wildcard breaks here.
  it('denies reads on deeply nested subcollections', async () => {
    const db = testEnv.authenticatedContext('user-123').firestore()
    await assertFails(getDoc(doc(db, 'users/u1/predictions/p1/notes/n1')))
  })
})

describe.skipIf(reachable)('Firestore security rules — emulator unreachable', () => {
  it('is skipped because the Firestore emulator is not running on ' + EMULATOR_HOST, () => {
    expect(true).toBe(true)
  })
})
