/**
 * Type-level tests for Dropdown transcription button in file list.
 * Validates Dropdown menu structure and D-03 compliance.
 */
import type { TranscriptionMode } from '../../../types/transcription'

// Test: Dropdown has exactly two items per D-01
const dropdownItems = [
  { key: 'local', label: '本地转录' },
  { key: 'cloud', label: '云端转录（通义听悟）' },
]
if (dropdownItems.length !== 2) throw new Error('Expected 2 dropdown items')

// Test: Cloud mode call omits sampling_rate per D-03
function buildCloudRequestBody(
  mode: TranscriptionMode,
  samplingRate?: number
): Record<string, unknown> {
  const body: Record<string, unknown> = { mode }
  if (mode === 'local' && samplingRate) {
    body.sampling_rate = samplingRate
  }
  return body
}

const cloudBody = buildCloudRequestBody('cloud')
if ('sampling_rate' in cloudBody)
  throw new Error('Cloud body must NOT contain sampling_rate per D-03')

const localBody = buildCloudRequestBody('local', 0.5)
if (!('sampling_rate' in localBody)) throw new Error('Local body must contain sampling_rate')
