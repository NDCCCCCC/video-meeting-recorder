package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VideoFileHandler 视频文件处理器
type VideoFileHandler struct {
	fileService  *services.VideoFileService
	auditService *audit.AuditLogService
	logger       *zap.Logger
}

// NewVideoFileHandler 创建视频文件处理器
func NewVideoFileHandler(
	fileService *services.VideoFileService,
	auditService *audit.AuditLogService,
	logger *zap.Logger,
) *VideoFileHandler {
	return &VideoFileHandler{
		fileService:  fileService,
		auditService: auditService,
		logger:       logger,
	}
}

// ListFiles 获取文件列表
func (h *VideoFileHandler) ListFiles(c *gin.Context) {
	var req services.ListFilesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 设置数据范围过滤参数
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	req.UserID = userID
	req.IsAdmin = middleware.GetIsAdmin(c)
	req.ApplyDataScope = true
	req.RoleIDs = middleware.GetRoleIDs(c)

	result, err := h.fileService.ListFiles(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("获取文件列表失败", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, result)
}

// GetFile 获取文件详情
func (h *VideoFileHandler) GetFile(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
		return
	}

	file, err := h.fileService.GetFileByID(c.Request.Context(),id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "文件不存在")
		return
	}

	response.GinSuccess(c, file)
}

// DownloadFile 下载文件（支持 Authorization 头或 token 查询参数）
func (h *VideoFileHandler) DownloadFile(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
		return
	}

	// 注意：token 验证由 SM4Auth 中间件处理（支持 Authorization 头和 token 查询参数）

	file, err := h.fileService.GetFileByID(c.Request.Context(),id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "文件不存在")
		return
	}

	// 检查数据访问权限
	// shared_viewer 和 admin 可以访问所有文件，普通用户只能访问自己创建的文件
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
		h.logger.Warn("用户无权访问文件",
			zap.Uint("user_id", userID),
			zap.Uint("file_id", id),
			zap.Uint("file_owner", file.CreatedBy))
		response.GinError(c, response.CodeForbidden, "无权访问此文件")
		return
	}

	if !file.Exists() {
		response.GinError(c, response.CodeNotFound, "物理文件不存在")
		return
	}

	// 打开文件
	fileHandle, err := os.Open(file.FilePath)
	if err != nil {
		h.logger.Error("无法打开文件", zap.Uint("file_id", id), zap.Error(err))
		response.GinError(c, response.CodeInternalError, "无法打开文件")
		return
	}
	defer fileHandle.Close()

	// 获取文件信息
	fileInfo, err := fileHandle.Stat()
	if err != nil {
		response.GinError(c, response.CodeInternalError, "无法获取文件信息")
		return
	}

	// 设置响应头
	setVideoHeaders(c, file, fileInfo.Size())

	// 发送文件
	http.ServeContent(c.Writer, c.Request, file.FileName, fileInfo.ModTime(), fileHandle)

	h.logger.Info("文件流传输完成", zap.Uint("file_id", id), zap.String("file_name", file.FileName))
}

// setVideoHeaders 设置视频流响应头
// 注意：http.ServeContent 会自动设置 Content-Length、Content-Range 和 Accept-Ranges
// 因此这里只需要设置 Content-Type、Content-Disposition 和 CORS 头
func setVideoHeaders(c *gin.Context, file *models.VideoFile, fileSize int64) {
	// 根据文件格式设置 Content-Type
	contentType := getContentType(file.Format)

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.FileName))

	// 设置视频流所需的 CORS 头
	// 注意：使用 * 允许所有源访问。如果部署在公网环境，建议改为 Origin 白名单
	// 内网环境或前后端同源部署时，此配置是安全的
	c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Range, Content-Type, Authorization")
	c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
}

// getContentType 根据格式获取 Content-Type
func getContentType(format string) string {
	switch format {
	case "mp4":
		return "video/mp4"
	case "mkv":
		return "video/x-matroska"
	default:
		return "application/octet-stream"
	}
}

// DeleteFile 删除文件
func (h *VideoFileHandler) DeleteFile(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
		return
	}

	oldFile, err := h.fileService.DeleteFile(c.Request.Context(), id)
	if err != nil {
		// Phase 19 D4: DeleteFile 返回 BusinessError，handler 用统一 HandleError 映射。
		response.HandleError(c, err)
		return
	}

	// 审计：OldData=oldFile (pre-delete), NewData=nil
	resourceID := oldFile.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionDelete,
		Module:     models.ModuleFile,
		Resource:   fmt.Sprintf("video_file:%d", oldFile.ID),
		ResourceID: &resourceID,
		OldData:    oldFile,
		NewData:    nil,
	}); err != nil {
		h.logger.Warn("Failed to record video file delete change", zap.Error(err), response.SentinelField(err), zap.Uint("file_id", id))
	}

	h.logger.Info("视频文件已删除", zap.Uint("file_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// BatchDeleteFilesRequest 批量删除请求
type BatchDeleteFilesRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchDeleteFiles 批量删除文件
func (h *VideoFileHandler) BatchDeleteFiles(c *gin.Context) {
	var req BatchDeleteFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "参数错误")
		return
	}

	oldFiles, result, err := h.fileService.BatchDeleteFiles(c.Request.Context(), req.IDs)
	if err != nil {
		h.logger.Error("批量删除文件失败", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	// 审计（option b）：单条 RecordChange，OldData=请求删除的所有旧文件（含 processing 跳过的），
	// Resource="video_file:<comma-joined-ids>"，Action="batch_delete"
	if len(oldFiles) > 0 {
		idStrs := make([]string, len(oldFiles))
		for i, f := range oldFiles {
			idStrs[i] = strconv.FormatUint(uint64(f.ID), 10)
		}
		resource := "video_file:" + strings.Join(idStrs, ",")
		resourceID := uint(0) // 批量操作没有单 ID

		if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
			Action:     "batch_delete",
			Module:     models.ModuleFile,
			Resource:   resource,
			ResourceID: &resourceID,
			OldData:    oldFiles,
			NewData:    nil,
		}); err != nil {
			h.logger.Warn("Failed to record batch video file delete change", zap.Error(err), response.SentinelField(err))
		}
	}

	h.logger.Info("批量删除文件成功",
		zap.Int("total", len(req.IDs)),
		zap.Int("success", result.Success),
		zap.Int("failed", result.Failed),
	)

	response.GinSuccess(c, result)
}

// GetFileStats 获取文件统计信息
func (h *VideoFileHandler) GetFileStats(c *gin.Context) {
	// 获取格式参数，默认只统计 mp4
	format := c.DefaultQuery("format", "mp4")

	stats, err := h.fileService.GetFileStats(c.Request.Context(), format)
	if err != nil {
		h.logger.Error("获取统计信息失败", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, stats)
}

// ScanFiles 扫描并导入未入库的视频文件
func (h *VideoFileHandler) ScanFiles(c *gin.Context) {
	result, err := h.fileService.ScanFiles(c.Request.Context())
	if err != nil {
		h.logger.Error("扫描文件失败", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, result)
}

// RenameFileRequest 重命名请求
type RenameFileRequest struct {
	NewName string `json:"new_name" binding:"required,min=1,max=200"`
}

// RenameFile 重命名视频文件
func (h *VideoFileHandler) RenameFile(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
		return
	}

	var req RenameFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// Trim whitespace and validate
	newName := trimString(req.NewName)
	if newName == "" {
		response.GinError(c, response.CodeInvalidRequest, "新文件名不能为空")
		return
	}

	// Reject path separators
	if containsPathSeparator(newName) {
		response.GinError(c, response.CodeInvalidRequest, "文件名不能包含路径分隔符")
		return
	}

	// Get user context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}

	// Load file to check ownership
	file, err := h.fileService.GetFileByID(c.Request.Context(),id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "文件不存在")
		return
	}

	// Check data access permission
	if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权重命名此文件")
		return
	}

	// Call service to rename
	hasSharedViewer := middleware.GetHasSharedViewer(c)
	if err := h.fileService.RenameVideoFile(c.Request.Context(), id, newName, userID, hasSharedViewer); err != nil {
		h.logger.Warn("重命名视频文件失败",
			zap.Uint("file_id", id),
			zap.String("new_name", newName),
			zap.Error(err),
			response.SentinelField(err))

		// STYLE-001 Phase 19 Wave 6：service 返回 BusinessError，handler 用统一 HandleError 映射。
		// 不再需要手写字符串匹配 switch。
		response.HandleError(c, err)
		return
	}

	// Get updated file info
	file, err = h.fileService.GetFileByID(c.Request.Context(),id)
	if err != nil {
		h.logger.Error("重命名成功但无法获取更新后的文件信息", zap.Error(err), response.SentinelField(err))
		response.GinSuccess(c, gin.H{
			"message": "重命名成功",
		})
		return
	}

	response.GinSuccess(c, gin.H{
		"message": "重命名成功",
		"data": gin.H{
			"id":        file.ID,
			"file_name": file.FileName,
			"file_path": file.FilePath,
		},
	})
}

// --- Phase 14 Batch Download Handler ---

// BatchDownloadFiles 批量下载文件（打包为ZIP）
func (h *VideoFileHandler) BatchDownloadFiles(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required,min=1,dive,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "参数错误")
		return
	}

	// 获取用户信息
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	isAdmin := middleware.GetIsAdmin(c)

	// 调用服务层
	resp, err := h.fileService.BatchDownloadFiles(c.Request.Context(), req.IDs, userID, isAdmin)
	if err != nil {
		h.logger.Error("批量下载文件失败",
			zap.Uint("user_id", userID),
			zap.Int("file_count", len(req.IDs)),
			zap.Error(err),
			response.SentinelField(err),
		)
		response.HandleError(c, err)
		return
	}
	defer resp.Reader.Close()

	// 设置响应头
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", resp.Filename))
	c.Header("Content-Type", resp.ContentType)
	c.Header("Accept-Ranges", "none") // ZIP 不支持范围请求
	c.Header("Access-Control-Expose-Headers", "Content-Disposition, Content-Type")

	// 流式响应
	c.DataFromReader(200, -1, resp.ContentType, resp.Reader, nil)

	h.logger.Info("批量下载文件成功",
		zap.Uint("user_id", userID),
		zap.Int("file_count", resp.FileCount),
		zap.String("filename", resp.Filename),
	)
}
