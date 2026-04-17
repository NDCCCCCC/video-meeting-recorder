# Technology Stack

**Project:** Record V2 - Video Splitting, Aliyun Tingwu Transcription, and PPT Extraction
**Researched:** 2026-04-17
**Mode:** Ecosystem - NEW capabilities for existing Go/Gin + React app

## Recommended Stack

### Core Additions

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `alibabacloud-oss-go-sdk-v2` | Latest (v2) | Aliyun OSS file upload | Official OSS SDK v2, supports Go 1.18+, environment-based credentials, modern API design |
| `net/http` (stdlib) | Built-in | Tingwu API calls | No official Go SDK for Tingwu; REST API requires manual HTTP requests with SignatureV4 signing |
| `crypto/hmac` + `crypto/sha256` (stdlib) | Built-in | Alibaba Cloud signature signing | Required for manual API authentication to Tingwu service |
| FFmpeg (CLI) | Already integrated | Video splitting | `-ss` seek + `-codec copy` for fast, lossless multi-point splitting |
| `Muprprpr/Go-pptx` | Latest | PPTX file generation | Pure Go, no license restrictions, actively maintained, full PPTX support |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` (stdlib) | Built-in | Tingwu API JSON parsing | All API request/response handling |
| `io` + `os` (stdlib) | Built-in | File upload streaming | Large video file upload to OSS |
| `context` (stdlib) | Built-in | Cancellation & timeouts | Long-running transcription tasks |

## Installation

```bash
# OSS SDK v2
go get github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss

# PPTX generation
go get github.com/Muprprpr/Go-pptx

# Note: Tingwu API uses stdlib net/http + manual signing
# FFmpeg CLI should already be installed on the system
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| OSS SDK | `alibabacloud-oss-go-sdk-v2` | `aliyun-oss-go-sdk` (v1) | V1 is legacy; v2 has modern API, better Go idioms, active maintenance |
| PPTX Generation | `Muprprpr/Go-pptx` | `unidoc/unioffice` | UniOffice requires PAID commercial license; Go-pptx is free/open-source |
| Video Splitting | FFmpeg CLI | `u2takey/ffmpeg-go` | FFmpeg already integrated via CLI; wrapper adds complexity without benefit |
| HTTP Client | `net/http` | `gin.Context` | Gin's context is for handlers; use stdlib for external API calls |
| Tingwu SDK | Manual REST | Wait for official SDK | No official Go SDK exists; manual HTTP is straightforward |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `unidoc/unioffice` | Commercial license required, adds cost | `Muprprpr/Go-pptx` (free, MIT/Apache) |
| `aliyun-oss-go-sdk` (v1) | Legacy, superseded by v2 | `alibabacloud-oss-go-sdk-v2` |
| `u2takey/ffmpeg-go` | Unnecessary abstraction; FFmpeg CLI already works | Direct `exec.Command()` calls to FFmpeg |
| Third-party Tingwu SDKs | Unofficial, unmaintained, security risk | Manual `net/http` with SignatureV4 |
| Heavy HTTP clients (resty, etc.) | Overkill for simple REST API calls | `net/http` stdlib |

## Stack Patterns by Variant

**If uploading large videos (>500MB):**
- Use OSS multipart upload via `client.AbortMultipartUpload()` / `CompleteMultipartUpload()`
- Because single-part upload fails on network interruption and memory constraints

**If Tingwu API response is large (>10MB):**
- Use `json.Decoder` streaming instead of `json.Unmarshal`
- Because full-buffer unmarshaling consumes excessive memory

**If FFmpeg split points are at non-keyframes:**
- Add `-re` (read input at native frame rate) or `-accurate_seek` before `-ss`
- Because default seek may be inaccurate; `accurate_seek` ensures precise timing

**If PPTX generation needs complex formatting:**
- Use Go-pptx's raw XML access via `.X()` method
- Because high-level API may not cover all OOXML specification features

## Integration Points

### 1. OSS Upload Flow
```
Local Video → OSS Client → Multipart Upload → Public URL → Tingwu API
```
- Credentials: Environment variables `OSS_ACCESS_KEY_ID`, `OSS_ACCESS_KEY_SECRET`
- Region: `cn-beijing` (match Tingwu service region)
- Lifecycle: Delete file after Tingwu transcription completes

### 2. Tingwu API Flow
```
POST /api/v1/files → Tingwu returns taskId
Poll GET /api/v1/tasks/{taskId} → Wait for status=COMPLETE
GET /api/v1/tasks/{taskId}/result → Fetch transcription + PPT URLs
Download results → Store in SQLite → Delete OSS file
```
- Authentication: Custom `Authorization` header with HMAC-SHA256 signature
- Domain: `tingwu.cn-beijing.aliyuncs.com`
- API Version: `2023-09-30`
- Polling interval: 5-10 seconds

### 3. Video Splitting Flow
```
User marks split points → FFmpeg -ss {time} -i input -codec copy -to {next_time} output
```
- Command pattern: `ffmpeg -ss {start} -i {input} -codec copy -to {end} {output}`
- Multiple segments: Execute sequentially or parallel (depending on CPU)
- No re-encoding: Preserves quality, fast execution

### 4. PPTX Generation Flow
```
Tingwu PPT images downloaded → Go-pptx creates slides → Insert images → Save .pptx
```
- One slide per Tingwu-extracted PPT frame
- Preserve timestamps in slide notes
- Auto-layout: Center image, add timestamp text

## Version Compatibility

| Package | Go Version | Notes |
|-----------|-----------------|-------|
| `alibabacloud-oss-go-sdk-v2` | Go 1.18+ | Project uses Go 1.24, fully compatible |
| `Muprprpr/Go-pptx` | Go 1.16+ | Compatible with Go 1.24 |
| FFmpeg CLI | Any (external) | System binary, not Go version dependent |

## Sources

### HIGH Confidence
- [Alibaba Cloud OSS Go SDK v2 GitHub](https://github.com/aliyun/alibabacloud-oss-go-sdk-v2) — Official OSS SDK documentation, installation, examples
- [Muprprpr/Go-pptx GitHub](https://github.com/Muprprpr/Go-pptx) — PPTX library features, installation
- [UniDoc UniOffice License](https://github.com/unidoc/unioffice) — Commercial license requirement (avoid)
- [FFmpeg Splitting Video](https://stackoverflow.com/questions/22952639) — Command-line split syntax with codec copy
- [FFmpeg Codecs](https://ffmpeg.org/ffmpeg-codecs.html) — Copy codec for lossless splitting

### MEDIUM Confidence
- [Aliyun Tingwu Official Docs](https://help.aliyun.com/zh/tingwu) — Service overview, API endpoints (verified via webReader)
- [Alibaba Cloud Credentials Guide](https://www.alibabacloud.com/help/en/sdk/developer-reference/v2-manage-go-access-credentials) — Credential management for Go SDKs

### LOW Confidence
- WebSearch for "Tingwu Go SDK" — No official Go SDK found; manual REST required
- WebSearch for "ffmpeg-go" — Wrapper exists but unnecessary given existing FFmpeg CLI integration

---
*Stack research for: Record V2 - Video Splitting & Aliyun Tingwu Integration*
*Researched: 2026-04-17*
