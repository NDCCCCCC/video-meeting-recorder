---
phase: 2
slug: local-transcription
plan: 04
type: frontend
wave: 2
status: completed
started: 2026-04-17T06:17:19Z
completed: 2026-04-17T06:17:45Z
duration_seconds: 26
tasks: 3
commits: 3
deviations: 0

# Phase 2 Plan 04: Frontend Transcription UI

## One-Liner

Complete user-facing transcription experience with trigger modal, real-time progress tracking with three-stage flow, and file list integration.

## Summary

Delivered the complete frontend layer for local transcription functionality. Users can now trigger transcription from the file list, select sampling rates (1s/2s/5s with 2s default per D-02), monitor real-time progress through a three-stage flow (extracting, detecting, generating per D-14), and download PPT files upon completion (per D-15). The implementation follows the UI-SPEC visual contract and existing codebase patterns.

## Key Files

### Created

- `frontend/src/types/transcription.ts` - TypeScript interfaces for transcription API contracts
- `frontend/src/api/transcription.ts` - API client functions (submitTranscription, getTranscriptionStatus)
- `frontend/src/components/TranscriptionProgressModal.tsx` - Real-time progress modal with staged phases

### Modified

- `frontend/src/utils/permissions.ts` - Added FILE_TRANSCRIBE permission constant
- `frontend/src/pages/files/index.tsx` - Extended file list with transcription button, trigger modal, and progress modal integration

## Commits

| Hash | Message | Files |
|------|---------|-------|
| 787df12 | feat(02-04): add transcription types and API client | frontend/src/types/transcription.ts, frontend/src/api/transcription.ts |
| 58caf2d | feat(02-04): add TranscriptionProgressModal component | frontend/src/components/TranscriptionProgressModal.tsx |
| 2836258 | feat(02-04): add transcription button and trigger modal to file list | frontend/src/utils/permissions.ts, frontend/src/pages/files/index.tsx |

## Deviations from Plan

None - plan executed exactly as written.

## Auth Gates

None encountered.

## Known Stubs

None - all functionality is fully implemented. The TranscriptionProgressModal polls the backend API (built in Plan 02-03) for real-time status updates. The "下载PPT" button uses the existing `/api/v1/files/{id}/download` endpoint.

## Threat Flags

None introduced - all new frontend code follows existing security patterns. Backend validation of sampling_rate values (whitelist: 1.0, 0.5, 0.2) prevents tampering threats (T-02-13). Polling interval of 5 seconds per D-16 mitigates denial-of-service concerns (T-02-12).

## Key Decisions

### Decision 1: Modal Non-Blocking Behavior
**Choice:** Made progress modal closeable via X button, transcription continues in background
**Rationale:** Per CONTEXT.md decision D-14, users should not be blocked from using the system during transcription
**Outcome:** Progress modal can be closed, polling continues, completion shown via notification or modal re-open

### Decision 2: Sampling Rate Default
**Choice:** Default to 2 seconds per frame (0.5 fps value)
**Rationale:** Per CONTEXT.md decision D-02, 2s/frame is recommended as balance between quality and speed
**Outcome:** Trigger modal pre-selects "2秒/帧" option

### Decision 3: Stage Display Format
**Choice:** Vertical list with icons (✓ completed, ● active, ○ pending)
**Rationale:** Per UI-SPEC, clear visual hierarchy shows three-stage flow (extracting, detecting, generating)
**Outcome:** Users see real-time progress through all transcription stages

## Technical Notes

### Polling Implementation
- 5-second interval per CONTEXT.md D-16
- Cleanup on component unmount prevents memory leaks
- Exponential backoff not implemented (would require API error detection)
- Polling continues even when modal closed (background transcription)

### File List Integration
- "转录" button only visible for mp4 files in 'ready' status per UI-SPEC
- Button placed after split button, before download button per D-13
- Uses PermissionGuard with FILE_TRANSCRIBE permission
- File list auto-refreshes on completion via loadFiles() callback

### Modal State Management
- Two separate modals: trigger (sampling rate selection) and progress (real-time updates)
- State stored in parent component (files/index.tsx) for simplicity
- Could be extracted to dedicated transcription state store if complexity increases

## Verification Results

### Automated Checks
- ✅ TypeScript compilation passes (no new errors)
- ✅ All three files created and importable
- ✅ API paths match backend contract from Plan 03:
  - POST `/api/v1/videos/{id}/transcribe`
  - GET `/api/v1/videos/{id}/transcription-status`

### Manual Verification Required
- [ ] Start dev server: `cd frontend && npm run dev`
- [ ] Navigate to `/files` page
- [ ] Verify "转录" button appears for mp4 files in 'ready' status
- [ ] Click "转录" button, verify trigger modal opens with sampling rate options
- [ ] Select sampling rate, click "开始转录", verify progress modal opens
- [ ] Verify progress modal shows three stages: extracting, detecting, generating
- [ ] Close progress modal (X button), verify it's closeable (non-blocking)
- [ ] Wait for completion, verify "下载PPT" button appears
- [ ] Click "下载PPT", verify browser downloads file

## Next Steps

This plan completes the frontend transcription UI layer. The backend transcription service (Plan 02-03) must be deployed for end-to-end testing. Future enhancements could include:
- Notification banner when transcription completes while modal is closed
- Batch transcription for multiple files
- Transcription history and retry logic
- PPT preview before download

## Dependencies

- **Depends on:** Plan 02-03 (backend transcription API endpoints)
- **Enables:** Plan 02-04 verification (manual UI testing with live backend)

## Self-Check: PASSED

- ✅ All three files created: frontend/src/types/transcription.ts, frontend/src/api/transcription.ts, frontend/src/components/TranscriptionProgressModal.tsx
- ✅ All three commits exist: 787df12, 58caf2d, 2836258
- ✅ TypeScript compilation passes (no new errors introduced)
- ✅ Files modified: frontend/src/utils/permissions.ts, frontend/src/pages/files/index.tsx
- ✅ No deviations from plan
- ✅ SUMMARY.md created
