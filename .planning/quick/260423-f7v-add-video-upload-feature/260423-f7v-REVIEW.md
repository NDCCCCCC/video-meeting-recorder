---
phase: 260423-f7v-add-video-upload-feature
reviewed: 2026-04-23T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - frontend/src/api/video-file.ts
  - frontend/src/components/VideoUploadModal.tsx
  - frontend/src/pages/files/index.tsx
findings:
  critical: 4
  warning: 6
  info: 4
  total: 14
status: issues_found
---

# Phase 260423-f7v: Code Review Report

**Reviewed:** 2026-04-23T00:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Reviewed video upload feature implementation including API client, upload modal component, and file management page integration. The implementation handles file uploads with progress tracking and validation, but contains several critical security vulnerabilities and code quality issues that should be addressed before deployment.

## Critical Issues

### CR-01: Token exposed in URL query parameter (Security)

**File:** `frontend/src/api/video-file.ts:42-44`
**Issue:** Authentication token is exposed in URL query parameter during file download, which can be logged in server access logs, browser history, and network intermediaries.

**Fix:**
```typescript
// Instead of passing token in URL, use Authorization header
export function downloadVideoFile(id: number, fileName?: string): void {
  const token = getToken()
  const url = `${API_BASE_URL}/api/v1/files/${id}/download`

  fetch(url, {
    headers: {
      ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    }
  })
  .then(response => response.blob())
  .then(blob => {
    const blobUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = fileName || `video_${id}.mp4`
    link.click()
    URL.revokeObjectURL(blobUrl)
  })
}
```

### CR-02: Missing error response handling for upload (Data Loss)

**File:** `frontend/src/api/video-file.ts:157-167`
**Issue:** Upload success check only validates status code 200-299, but doesn't validate the response structure. Malformed responses could cause silent failures.

**Fix:**
```typescript
xhr.addEventListener('load', () => {
  if (xhr.status >= 200 && xhr.status < 300) {
    try {
      const response = JSON.parse(xhr.responseText)
      // Validate response structure
      if (!response || !response.data || !response.data.file_id) {
        reject(new Error('服务器返回的响应格式无效'))
        return
      }
      resolve(response)
    } catch (error) {
      reject(new Error('解析响应失败'))
    }
  } else {
    // Try to parse error message from server
    let errorMsg = `上传失败: ${xhr.status} ${xhr.statusText}`
    try {
      const errorResponse = JSON.parse(xhr.responseText)
      if (errorResponse.message) {
        errorMsg = errorResponse.message
      }
    } catch {
      // Use default error message
    }
    reject(new Error(errorMsg))
  }
})
```

### CR-03: Type assertion bypasses type safety (Bug)

**File:** `frontend/src/components/VideoUploadModal.tsx:72`
**Issue:** Using `as VideoFile` type assertion without runtime validation. If the API returns incorrect data structure, this will cause runtime errors.

**Fix:**
```typescript
// Add proper validation
interface VideoFileValidation {
  id?: number
  file_name?: string
  file_path?: string
  file_size?: number
  mime_type?: string
  format?: string
}

const validateVideoFile = (data: unknown): data is VideoFile => {
  const file = data as VideoFileValidation
  return !!(file?.id && file?.file_name && file?.file_path)
}

// In handleUpload:
if (result.data) {
  if (!validateVideoFile(result.data)) {
    message.error('服务器返回的数据格式无效')
    setFileList([])
    return
  }
  message.success(`${file.name} 上传成功`)
  setFileList([])
  setProgress(0)
  onUploadSuccess(result.data)
}
```

### CR-04: MIME type validation bypass (Security)

**File:** `frontend/src/components/VideoUploadModal.tsx:54-58`
**Issue:** MIME type validation can be easily bypassed by modifying file metadata. The MIME type is client-controlled and not trustworthy. Additionally, some operating systems don't provide MIME types for all video formats, causing false negatives.

**Fix:**
```typescript
// Remove MIME type check - it's unreliable and security should be enforced server-side
// Only validate file extension and enforce server-side validation

const handleUpload = useCallback(
  async (file: File) => {
    // Validate file size
    if (file.size > MAX_FILE_SIZE) {
      message.error(`文件大小超过限制 (最大 5GB)`)
      return false
    }

    // Validate file type by extension (more reliable than MIME type)
    const hasValidExtension = ACCEPTED_VIDEO_FORMATS.some((ext) =>
      file.name.toLowerCase().endsWith(ext)
    )
    if (!hasValidExtension) {
      message.error(`不支持的文件格式，仅支持: ${ACCEPTED_VIDEO_FORMATS.join(', ')}`)
      return false
    }

    // Security: Rely on server-side MIME type validation
    // Client-side MIME type can be spoofed

    setUploading(true)
    setProgress(0)

    try {
      const result = await uploadVideoFile(file, (percent) => {
        setProgress(percent)
      })

      if (result.data) {
        if (!validateVideoFile(result.data)) {
          message.error('服务器返回的数据格式无效')
          setFileList([])
          return
        }
        message.success(`${file.name} 上传成功`)
        setFileList([])
        setProgress(0)
        onUploadSuccess(result.data)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '上传失败')
      setFileList([])
    } finally {
      setUploading(false)
    }

    return false // Prevent auto upload
  },
  [onUploadSuccess]
)
```

## Warnings

### WR-01: Non-serializable file upload breaks API abstraction (Code Quality)

**File:** `frontend/src/api/video-file.ts:136-187`
**Issue:** The `uploadVideoFile` function uses XMLHttpRequest directly instead of the `apiRequest` pattern used elsewhere in the codebase. This creates inconsistency and makes it harder to add centralized logging, error handling, or request interceptors.

**Fix:**
```typescript
// Consider using fetch with proper error handling to maintain consistency
// Or extend apiRequest to support progress callbacks
export async function uploadVideoFile(
  file: File,
  onProgress?: (percent: number) => void
): Promise<ApiResponse<FileUploadResult>> {
  const token = getToken()
  const formData = new FormData()
  formData.append('file', file)
  formData.append('folder', 'videos')

  const xhr = new XMLHttpRequest()

  return new Promise((resolve, reject) => {
    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress((event.loaded / event.total) * 100)
      }
    })

    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const response = JSON.parse(xhr.responseText)
          if (response.data?.file_id) {
            resolve(response)
          } else {
            reject(new Error('服务器返回的响应格式无效'))
          }
        } catch (error) {
          reject(new Error('解析响应失败'))
        }
      } else {
        reject(new Error(`上传失败: ${xhr.status} ${xhr.statusText}`))
      }
    })

    xhr.addEventListener('error', () => reject(new Error('网络错误，上传失败')))
    xhr.addEventListener('abort', () => reject(new Error('上传已取消')))

    xhr.open('POST', `${API_BASE_URL}/api/v1/storage/upload`)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }
    xhr.send(formData)
  })
}
```

### WR-02: Missing cleanup on unmount (Resource Leak)

**File:** `frontend/src/components/VideoUploadModal.tsx:37-84`
**Issue:** The upload promise is not tracked or cancelled when the modal closes. If the user closes the modal during upload, the upload continues in the background and may cause state updates on unmounted component.

**Fix:**
```typescript
import { useState, useCallback, useRef } from 'react'

export default function VideoUploadModal({
  visible,
  onCancel,
  onUploadSuccess,
}: VideoUploadModalProps) {
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [fileList, setFileList] = useState<any[]>([])
  const xhrRef = useRef<XMLHttpRequest | null>(null)

  const handleUpload = useCallback(
    async (file: File) => {
      // ... validation code ...

      setUploading(true)
      setProgress(0)

      try {
        const result = await uploadVideoFile(file, (percent) => {
          setProgress(percent)
        }, (xhr) => {
          xhrRef.current = xhr
        })

        if (result.data) {
          message.success(`${file.name} 上传成功`)
          setFileList([])
          setProgress(0)
          onUploadSuccess(result.data)
        }
      } catch (error) {
        message.error(error instanceof Error ? error.message : '上传失败')
        setFileList([])
      } finally {
        setUploading(false)
        xhrRef.current = null
      }

      return false
    },
    [onUploadSuccess]
  )

  const handleCancel = useCallback(() => {
    if (xhrRef.current) {
      xhrRef.current.abort()
      xhrRef.current = null
    }
    onCancel()
  }, [onCancel])

  return (
    <Modal
      title="上传视频"
      open={visible}
      onCancel={handleCancel}
      // ... rest of modal props
    >
      {/* ... */}
    </Modal>
  )
}
```

### WR-03: Missing loading state for upload button (UX)

**File:** `frontend/src/components/VideoUploadModal.tsx:104-111`
**Issue:** The cancel button is disabled during upload, but there's no visual feedback on the modal itself that an upload is in progress before the progress bar appears.

**Fix:**
```typescript
<Modal
  title="上传视频"
  open={visible}
  onCancel={handleCancel}
  okButtonProps={{ style: { display: 'none' } }}
  cancelButtonProps={{ disabled: uploading }}
  cancelText={uploading ? "上传中..." : "关闭"}
  width={600}
  closable={!uploading}
  maskClosable={!uploading}
>
```

### WR-04: Inconsistent null handling in activeTranscriptions (Bug)

**File:** `frontend/src/pages/files/index.tsx:447-451`
**Issue:** Non-null assertion operator (!) is used when getting task info from Map, but the Map could have been modified between the has() check and get() call.

**Fix:**
```typescript
moreMenuItems.push({
  key: 'view-transcription',
  icon: <CloudOutlined />,
  label: '查看转录进度',
  onClick: () => {
    const taskInfo = activeTranscriptions.get(record.id)
    if (!taskInfo) {
      message.error('转录任务信息未找到')
      return
    }
    setTranscriptionVideoFile(record)
    setCloudTranscriptionMode(taskInfo.mode as TranscriptionMode)
    setSelectedSamplingRate(taskInfo.samplingRate)
    setTranscriptionModalOpen(true)
  },
})
```

### WR-05: Silent error handling hides issues (Maintainability)

**File:** `frontend/src/pages/files/index.tsx:162-165`
**Issue:** Stats loading failure is silently ignored, which can mask real issues like network failures or API changes.

**Fix:**
```typescript
// 加载统计信息
const loadStats = useCallback(async () => {
  try {
    const response = await videoFileApi.getVideoFileStats()
    if (response.data) {
      setStats(response.data)
    }
  } catch (error) {
    console.warn('Failed to load stats:', error)
    // Only show user-facing message for critical errors
    if (error instanceof Error && !error.message.includes('404')) {
      message.warning('无法加载统计信息')
    }
  }
}, [])
```

### WR-06: File extension stripping logic fails for multiple dots (Bug)

**File:** `frontend/src/pages/files/index.tsx:319`
**Issue:** The regex `/\.[^/.]+$/` fails for filenames like "my.video.file.mp4" - it will strip ".mp4" correctly but the logic assumes only one extension.

**Fix:**
```typescript
const handleRename = useCallback((file: VideoFile) => {
  setRenamingFile(file)
  // Find the last dot and split from there
  const lastDotIndex = file.file_name.lastIndexOf('.')
  const nameWithoutExt = lastDotIndex > 0
    ? file.file_name.substring(0, lastDotIndex)
    : file.file_name
  setNewFileName(nameWithoutExt)
  setRenameModalVisible(true)
}, [])

// And in confirmRename:
const lastDotIndex = renamingFile.file_name.lastIndexOf('.')
const nameWithoutExt = lastDotIndex > 0
  ? renamingFile.file_name.substring(0, lastDotIndex)
  : renamingFile.file_name
if (newFileName.trim() === nameWithoutExt) {
  message.info('文件名未改变')
  setRenameModalVisible(false)
  return
}
```

## Info

### IN-01: Hardcoded magic number for file size limit

**File:** `frontend/src/components/VideoUploadModal.tsx:26`
**Issue:** The 5GB file size limit is hardcoded. This should match the backend configuration and be configurable.

**Fix:**
```typescript
// Define in a config file or import from backend
const MAX_FILE_SIZE = import.meta.env.VITE_MAX_UPLOAD_SIZE || 5 * 1024 * 1024 * 1024

// Or fetch from backend API
const getMaxFileSize = async (): Promise<number> => {
  try {
    const response = await fetch('/api/v1/storage/config')
    const config = await response.json()
    return config.maxFileSize || 5 * 1024 * 1024 * 1024
  } catch {
    return 5 * 1024 * 1024 * 1024 // fallback
  }
}
```

### IN-02: Duplicate file extension extraction logic

**File:** `frontend/src/pages/files/index.tsx:319, 331`
**Issue:** File extension extraction logic is duplicated in two places. Extract to a utility function.

**Fix:**
```typescript
// Add utility function
const getFileExtension = (fileName: string): string => {
  const lastDotIndex = fileName.lastIndexOf('.')
  return lastDotIndex > 0 ? fileName.substring(lastDotIndex + 1) : ''
}

const getFileNameWithoutExtension = (fileName: string): string => {
  const lastDotIndex = fileName.lastIndexOf('.')
  return lastDotIndex > 0 ? fileName.substring(0, lastDotIndex) : fileName
}

// Use in both places
const nameWithoutExt = getFileNameWithoutExtension(file.file_name)
const extension = getFileExtension(file.file_name)
```

### IN-03: Missing TypeScript strict null checks

**File:** `frontend/src/components/VideoUploadModal.tsx:72`
**Issue:** The code accesses `result.data` without checking if `result` is null first.

**Fix:**
```typescript
if (result?.data) {
  if (!validateVideoFile(result.data)) {
    message.error('服务器返回的数据格式无效')
    setFileList([])
    return
  }
  message.success(`${file.name} 上传成功`)
  setFileList([])
  setProgress(0)
  onUploadSuccess(result.data)
}
```

### IN-04: Inconsistent error message language

**File:** `frontend/src/api/video-file.ts:163, 166`
**Issue:** Error messages mix Chinese and English. Should be consistent.

**Fix:**
```typescript
// Use Chinese consistently (matches UI language)
reject(new Error('解析响应失败'))
reject(new Error(`上传失败: ${xhr.status} ${xhr.statusText}`))
reject(new Error('网络错误，上传失败'))
reject(new Error('上传已取消'))
```

---

_Reviewed: 2026-04-23T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
