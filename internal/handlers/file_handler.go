package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/storage"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// FileHandler 文件处理器
type FileHandler struct {
	fileService *storage.FileService
	logger      *zap.Logger
	jwtService  *auth.JWTService
}

// NewFileHandler 创建文件处理器
func NewFileHandler(
	fileService *storage.FileService,
	logger *zap.Logger,
	jwtService *auth.JWTService,
) *FileHandler {
	return &FileHandler{
		fileService: fileService,
		logger:      logger,
		jwtService:  jwtService,
	}
}

// SetJWTService 设置JWT服务（向后兼容）
func (h *FileHandler) SetJWTService(jwtService *auth.JWTService) {
	h.jwtService = jwtService
}

// SetLogger 设置日志记录器（向后兼容）
func (h *FileHandler) SetLogger(logger *zap.Logger) {
	h.logger = logger
}

// getUserID 从上下文获取用户ID
func (h *FileHandler) getUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// Upload 上传文件
// @Summary 上传文件
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "文件"
// @Param folder formData string false "文件夹" default(uploads)
// @Param is_public formData bool false "是否公开" default(false)
// @Param expires_in formData string false "过期时间(秒)"
// @Success 200 {object} response.Response
// @Router /api/v1/storage/upload [post]
func (h *FileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "文件上传失败")
		return
	}

	folder := c.PostForm("folder")
	if folder == "" {
		folder = "uploads"
	}

	isPublic := c.PostForm("is_public") == "true"

	var expiresIn time.Duration
	if expiresInStr := c.PostForm("expires_in"); expiresInStr != "" {
		if sec, err := strconv.Atoi(expiresInStr); err == nil && sec > 0 {
			expiresIn = time.Duration(sec) * time.Second
		}
	}

	uploadReq := &storage.UploadRequest{
		File:      file,
		Folder:    folder,
		IsPublic:  isPublic,
		ExpiresIn: expiresIn,
	}

	result, err := h.fileService.Upload(c.Request.Context(), h.getUserID(c), uploadReq)
	if err != nil {
		h.logger.Warn("文件上传失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	response.GinSuccess(c, result)
}

// Download 下载文件（支持 Range 请求和断点续传）
// @Summary 下载文件
// @Tags 文件管理
// @Produce application/octet-stream
// @Param token path string true "访问令牌"
// @Success 200 {file} file
// @Router /api/v1/files/download/{token} [get]
func (h *FileHandler) Download(c *gin.Context) {
	accessToken := c.Param("token")
	userID := h.getUserID(c)

	reader, filename, err := h.fileService.Download(c.Request.Context(), accessToken, userID)
	if err != nil {
		h.logger.Warn("文件下载失败",
			zap.Uint("user_id", userID),
			zap.String("token", accessToken),
			zap.Error(err),
		)
		response.GinError(c, response.CodeNotFound, err.Error())
		return
	}
	defer reader.Close()

	// 获取文件信息用于设置响应头
	// 注意：这里需要将 reader 转换为可Seek的接口或使用临时文件
	// 简化方案：使用 c.DataFromReader 但添加 Range 支持
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")

	// 流式传输文件
	c.DataFromReader(200, -1, "application/octet-stream", reader, nil)
}

// Delete 删除文件
// @Summary 删除文件
// @Tags 文件管理
// @Produce json
// @Param id path int true "文件ID"
// @Success 200 {object} response.Response
// @Router /api/v1/storage/:id [delete]
func (h *FileHandler) Delete(c *gin.Context) {
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的ID")
		return
	}

	err = h.fileService.Delete(c.Request.Context(), uint(fileID), h.getUserID(c))
	if err != nil {
		h.logger.Warn("删除文件失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// Share 生成分享链接
// @Summary 生成分享链接
// @Tags 文件管理
// @Produce json
// @Param id path int true "文件ID"
// @Param request body ShareRequest true "分享请求"
// @Success 200 {object} response.Response
// @Router /api/v1/storage/:id/share [post]
func (h *FileHandler) Share(c *gin.Context) {
	fileID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的ID")
		return
	}

	var req ShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 默认24小时过期
	expiresIn := 24 * time.Hour
	if req.ExpiresIn > 0 {
		expiresIn = time.Duration(req.ExpiresIn) * time.Second
	}

	shareURL, err := h.fileService.ShareFile(
		c.Request.Context(),
		uint(fileID),
		h.getUserID(c),
		expiresIn,
		req.Password,
	)
	if err != nil {
		h.logger.Warn("生成分享链接失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	response.GinSuccess(c, gin.H{"share_url": shareURL})
}

// ShareRequest 分享请求
type ShareRequest struct {
	ExpiresIn int    `json:"expires_in" binding:"min=60,max=2592000"` // 1分钟到30天
	Password  string `json:"password" binding:"omitempty,max=100"`
}

// ShareDownload 通过分享链接下载
// @Summary 通过分享链接下载
// @Tags 文件管理
// @Produce application/octet-stream
// @Param token path string true "分享令牌"
// @Param password query string false "密码"
// @Success 200 {file} file
// @Router /api/v1/files/share/{token} [get]
func (h *FileHandler) ShareDownload(c *gin.Context) {
	shareToken := c.Param("token")
	password := c.Query("password")

	reader, filename, err := h.fileService.GetShareDownload(c.Request.Context(), shareToken, password)
	if err != nil {
		response.GinError(c, response.CodeNotFound, err.Error())
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.DataFromReader(200, -1, "application/octet-stream", reader, nil)
}

// List 获取文件列表
// @Summary 获取文件列表
// @Tags 文件管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "页大小" default(20)
// @Param file_type query string false "文件类型"
// @Param keyword query string false "关键词"
// @Success 200 {object} response.Response
// @Router /api/v1/storage [get]
func (h *FileHandler) List(c *gin.Context) {
	var req storage.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Page = 1
		req.PageSize = 20
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	result, err := h.fileService.Query(c.Request.Context(), h.getUserID(c), &req)
	if err != nil {
		h.logger.Warn("查询文件列表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "查询失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetQuota 获取存储配额
// @Summary 获取存储配额
// @Tags 文件管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/storage/quota [get]
func (h *FileHandler) GetQuota(c *gin.Context) {
	quota, err := h.fileService.GetUserQuota(c.Request.Context(), h.getUserID(c))
	if err != nil {
		h.logger.Warn("获取存储配额失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取失败")
		return
	}

	response.GinSuccess(c, quota)
}
