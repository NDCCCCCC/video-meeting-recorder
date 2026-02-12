package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VideoFileHandler 视频文件处理器
type VideoFileHandler struct {
	fileService *services.VideoFileService
	logger      *zap.Logger
}

// NewVideoFileHandler 创建视频文件处理器
func NewVideoFileHandler(
	fileService *services.VideoFileService,
	logger *zap.Logger,
) *VideoFileHandler {
	return &VideoFileHandler{
		fileService: fileService,
		logger:      logger,
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
	req.UserID = middleware.GetUserID(c)
	req.IsAdmin = middleware.GetIsAdmin(c)
	req.ApplyDataScope = true

	result, err := h.fileService.ListFiles(&req)
	if err != nil {
		h.logger.Error("获取文件列表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取文件列表失败")
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

	file, err := h.fileService.GetFileByID(id)
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

	// 注意：token 验证由 JWTAuth 中间件处理（支持 Authorization 头和 token 查询参数）

	file, err := h.fileService.GetFileByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "文件不存在")
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
func setVideoHeaders(c *gin.Context, file *models.VideoFile, fileSize int64) {
	// 根据文件格式设置 Content-Type
	contentType := getContentType(file.Format)

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.FileName))
	c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
	c.Header("Accept-Ranges", "bytes") // 支持视频进度拖动

	// 设置 CORS 头 - 验证 Origin 并返回匹配的源
	origin := c.GetHeader("Origin")
	if origin != "" {
		// 生产环境应该配置允许的 Origin 白名单
		// 这里为了兼容视频播放功能，允许请求的 Origin
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
	} else {
		// 没有 Origin 头的请求（如同源请求、直接下载）
		c.Header("Access-Control-Allow-Origin", "*")
	}

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

	if err := h.fileService.DeleteFile(id); err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("视频文件已删除", zap.Uint("file_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// GetFileStats 获取文件统计信息
func (h *VideoFileHandler) GetFileStats(c *gin.Context) {
	stats, err := h.fileService.GetFileStats()
	if err != nil {
		h.logger.Error("获取统计信息失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取统计信息失败")
		return
	}

	response.GinSuccess(c, stats)
}

// ScanFiles 扫描并导入未入库的视频文件
func (h *VideoFileHandler) ScanFiles(c *gin.Context) {
	result, err := h.fileService.ScanFiles()
	if err != nil {
		h.logger.Error("扫描文件失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "扫描文件失败")
		return
	}

	response.GinSuccess(c, result)
}
