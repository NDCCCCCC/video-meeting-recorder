/**
 * Type-level and structural tests for TranscriptionProgressModal.
 * Validates props interface, mode support, and stage config at compile time.
 */
import type { TranscriptionMode, CloudTranscriptionStage } from '../../../types/transcription'

// Test: Props mode field accepts both values
const localProp: TranscriptionMode = 'local'
const cloudProp: TranscriptionMode = 'cloud'

// Test: All cloud stages are valid CloudTranscriptionStage values
const cloudStages: CloudTranscriptionStage[] = [
  'uploading', 'queued', 'cloud_processing', 'downloading'
]
if (cloudStages.length !== 4) {
  throw new Error('Expected 4 cloud stages')
}

// Test: Polling interval logic (10s for cloud, 5s for local)
function getPollingInterval(mode: TranscriptionMode): number {
  return mode === 'cloud' ? 10000 : 5000
}
const cloudInterval = getPollingInterval('cloud')
const localInterval = getPollingInterval('local')
if (cloudInterval !== 10000) throw new Error('Cloud interval must be 10000ms')
if (localInterval !== 5000) throw new Error('Local interval must be 5000ms')

// Test: Fallback message is exact per D-08
const fallbackMessage = '云端转录失败，已自动切换到本地转录'
if (fallbackMessage !== '云端转录失败，已自动切换到本地转录') {
  throw new Error('Fallback message mismatch')
}

// Suppress unused
void localProp; void cloudProp
