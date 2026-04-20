// PPT 幻灯片图片
export interface SlideImage {
  slide_number: number
  thumbnail_url: string
  fullsize_url: string
}

// PPT 结果
export interface PPTResult {
  id: number
  file_name: string
  file_path: string
  file_size: number
  page_count: number
  format: string
  source_type: 'transcription' | 'merge'
  slide_cache_path: string
  merged_from: string
  source_video_file_id: number
  transcription_task_id: number | null
  created_at: string
  updated_at: string
}

// 幻灯片获取响应
export interface SlidesResponse {
  slide_count: number
  slides: SlideImage[]
  status: 'ready' | 'extracting'
}

// PPT 列表响应
export interface PPTListResponse {
  ppts: PPTResult[]
}

// 合并幻灯片项
export interface MergeSlideItem {
  ppt_file_id: number
  slide_number: number
}

// 合并请求
export interface MergeRequest {
  slides: MergeSlideItem[]
  output_name?: string
  video_file_id: number
}

// 合并响应
export interface MergeResponse {
  ppt_file_id: number
  file_name: string
  page_count: number
}

// 已选择的幻灯片（用于合并模式）
export interface SelectedSlide {
  id: string           // Format: "{pptFileId}_{slideNumber}"
  ppt_file_id: number
  slide_number: number
  thumbnail_url: string
  source_name: string   // Display name for source PPT
}

// SlideCapturePanel props
export interface SlideCapturePanelProps {
  pptFileId: number
  videoFileId: number
  currentSlide: number
  totalSlides: number
  onSlideInserted?: (newSlideNumber: number) => void
  onCancel?: () => void
  open?: boolean
}
