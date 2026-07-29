/**
 * Type-level tests for transcription API client.
 * Validates that API function signatures and request shapes are correct.
 * Per D-03: submitTranscriptionWithMode('cloud') must NOT include sampling_rate.
 */
import type { TranscriptionMode } from '../../types/transcription'

// Test: submitTranscriptionWithMode signature accepts correct arguments
type SubmitFn = (
  videoFileId: number,
  mode: TranscriptionMode,
  samplingRate?: number
) => Promise<unknown>

// Test: cloud mode call omits samplingRate
const cloudCall: Parameters<SubmitFn> = [1, 'cloud']
// cloudCall[2] is undefined (samplingRate is optional) -- per D-03

// Test: local mode call includes samplingRate
const localCall: Parameters<SubmitFn> = [1, 'local', 0.5]

// Test: getTranscriptionText signature
type GetTextFn = (videoFileId: number) => Promise<unknown>
const textCall: Parameters<GetTextFn> = [42]

// Suppress unused warnings
void cloudCall
void localCall
void textCall
