package models

// MergeRequest 合并幻灯片请求
type MergeRequest struct {
	Slides      []MergeSlideItem `json:"slides" binding:"required"`        // Ordered list of slides to merge
	OutputName  string           `json:"output_name"`                      // Output filename (optional, default: "合并PPT.pptx")
	VideoFileID uint             `json:"video_file_id" binding:"required"` // For ownership validation and PPTFile association
}

// MergeSlideItem 单个幻灯片选择项
type MergeSlideItem struct {
	PptFileID   uint `json:"ppt_file_id" binding:"required"`  // Source PPT file ID
	SlideNumber int  `json:"slide_number" binding:"required"` // 1-based slide number within that PPT
}
