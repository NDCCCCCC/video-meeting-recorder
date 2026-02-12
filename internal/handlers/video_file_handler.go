package handlers

import (
	"fmt"
	"net/http"
	"os"

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
// @Summary 获取视频文件列表
// @Description 分页获取视频文件列表，支持筛选
// @Tags 文件管理
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param task_id query int false "任务ID"
// @Param status query string false "文件状态"
// @Param format query string false "文件格式"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} response.Response{data=services.ListFilesResponse}
// @Router /api/v1/files [get]
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

	result, err := h.fileService.ListFiles(&req)
	if err != nil {
		h.logger.Error("Failed to list video files", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取文件列表失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetFile 获取文件详情
// @Summary 获取视频文件详情
// @Description 根据ID获取视频文件详细信息
// @Tags 文件管理
// @Security Bearer
// @Param id path int true "文件ID"
// @Success 200 {object} response.Response{data=models.VideoFile}
// @Router /api/v1/files/{id} [get]
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

// DownloadFile 下载文件
// @Summary 下载视频文件
// @Description 下载指定的视频文件
// @Tags 文件管理
// @Security Bearer
// @Param id path int true "文件ID"
// @Success 200 {file} binary
// @Router /api/v1/files/{id}/download [get]
func (h *VideoFileHandler) DownloadFile(c *gin.Context) {
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

	// 检查文件是否存在
	if !file.Exists() {
		response.GinError(c, response.CodeNotFound, "物理文件不存在")
		return
	}

	// 打开文件
	fileHandle, err := os.Open(file.FilePath)
	if err != nil {
		h.logger.Error("Failed to open file", zap.Uint("file_id", id), zap.Error(err))
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
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\""+file.FileName+"\"")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 发送文件
	http.ServeContent(c.Writer, c.Request, file.FileName, fileInfo.ModTime(), fileHandle)

	h.logger.Info("File downloaded", zap.Uint("file_id", id), zap.String("file_name", file.FileName))
}

// DeleteFile 删除文件
// @Summary 删除视频文件
// @Description 删除指定的视频文件（包括物理文件）
// @Tags 文件管理
// @Security Bearer
// @Param id path int true "文件ID"
// @Success 200 {object} response.Response
// @Router /api/v1/files/{id} [delete]
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

	h.logger.Info("Video file deleted", zap.Uint("file_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// GetFileStats 获取文件统计信息
// @Summary 获取文件统计信息
// @Description 获取视频文件的统计信息
// @Tags 文件管理
// @Security Bearer
// @Success 200 {object} response.Response
// @Router /api/v1/files/stats [get]
func (h *VideoFileHandler) GetFileStats(c *gin.Context) {
	stats, err := h.fileService.GetFileStats()
	if err != nil {
		h.logger.Error("Failed to get file stats", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取统计信息失败")
		return
	}

	response.GinSuccess(c, stats)
}

// ScanFiles 扫描并导入未入库的视频文件
// @Summary 扫描录制目录
// @Description 扫描 data/recordings/ 目录，自动导入未入库的视频文件
// @Tags 文件管理
// @Security Bearer
// @Success 200 {object} response.Response{data=services.ScanResult}
// @Router /api/v1/files/scan [post]
func (h *VideoFileHandler) ScanFiles(c *gin.Context) {
	result, err := h.fileService.ScanFiles()
	if err != nil {
		h.logger.Error("Failed to scan files", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "扫描文件失败")
		return
	}

	response.GinSuccess(c, result)
}
