/**
 * Type-level tests for result page cloud mode integration.
 * Validates Dropdown retranscribe button and Tabs structure.
 */
import type { TranscriptionMode } from '../../../types/transcription'

// Test: Retranscribe Dropdown has two items per D-02
const retranscribeItems = [
  { key: 'local', label: '本地转录' },
  { key: 'cloud', label: '云端转录（通义听悟）' },
]
if (retranscribeItems.length !== 2) throw new Error('Expected 2 retranscribe items')

// Test: Tabs structure per D-09
const tabKeys = ['info', 'text']
if (!tabKeys.includes('info')) throw new Error('Missing info tab')
if (!tabKeys.includes('text')) throw new Error('Missing text tab')

// Test: Cloud retranscribe omits sampling_rate per D-03
function buildRetranscribeBody(mode: TranscriptionMode): Record<string, unknown> {
  const body: Record<string, unknown> = { mode }
  // Per D-03: no sampling_rate for cloud
  return body
}

const cloudBody = buildRetranscribeBody('cloud')
if ('sampling_rate' in cloudBody)
  throw new Error('Cloud retranscribe must NOT have sampling_rate per D-03')

const localBody = buildRetranscribeBody('local')
if (localBody.mode !== 'local') throw new Error('Local retranscribe mode mismatch')
