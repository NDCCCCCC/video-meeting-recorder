/**
 * Type-level tests for transcription types.
 * These are validated by tsc --noEmit, not by a test runner.
 * If these compile without errors, the type contracts are correct.
 */
import type {
  TranscriptionMode,
  CloudTranscriptionStage,
  AnyTranscriptionStage,
  TextSegment,
  TranscriptionTextResponse,
  TranscriptionStatusResponseExtended,
  TranscriptionTriggerRequestExtended,
  TranscriptionTriggerResponseExtended,
} from '../transcription'

// Validate TranscriptionMode values
const localMode: TranscriptionMode = 'local'
const cloudMode: TranscriptionMode = 'cloud'

// Validate CloudTranscriptionStage values
const uploading: CloudTranscriptionStage = 'uploading'
const queued: CloudTranscriptionStage = 'queued'
const processing: CloudTranscriptionStage = 'cloud_processing'
const downloading: CloudTranscriptionStage = 'downloading'

// Validate AnyTranscriptionStage accepts both local and cloud stages
const anyLocal: AnyTranscriptionStage = 'extracting'
const anyCloud: AnyTranscriptionStage = 'uploading'

// Validate TextSegment has required fields
const segment: TextSegment = {
  text: 'test',
  begin_time: 1000,
  end_time: 2000,
  segment_index: 0,
}

// Validate TranscriptionTextResponse
const textResponse: TranscriptionTextResponse = {
  segments: [segment],
  total_count: 1,
}

// Validate extended status response accepts mode
const statusResp: TranscriptionStatusResponseExtended = {
  status: 'processing',
  current_stage: 'cloud_processing',
  frames_processed: 0,
  total_frames: 0,
  percentage: 50,
  error_message: '',
  result_ppt_file_id: null,
  mode: 'cloud',
}

// Validate trigger request: cloud mode does NOT need sampling_rate per D-03
const cloudRequest: TranscriptionTriggerRequestExtended = {
  mode: 'cloud',
  // sampling_rate is intentionally omitted for cloud per D-03
}

// Validate trigger request: local mode includes sampling_rate
const localRequest: TranscriptionTriggerRequestExtended = {
  mode: 'local',
  sampling_rate: 0.5,
}

// Validate trigger response
const triggerResp: TranscriptionTriggerResponseExtended = {
  video_file_id: 1,
  status: 'processing',
  mode: 'cloud',
}

// Suppress unused warnings
void localMode
void cloudMode
void uploading
void queued
void processing
void downloading
void anyLocal
void anyCloud
void segment
void textResponse
void statusResp
void cloudRequest
void localRequest
void triggerResp
