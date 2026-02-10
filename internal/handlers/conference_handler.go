package handlers

import (
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ConferenceRecordHandler 会议记录处理器
type ConferenceRecordHandler struct {
	conferenceService *services.ConferenceRecordService
	logger            *zap.Logger
}

// NewConferenceRecordHandler 创建会议记录处理器
func NewConferenceRecordHandler(
	conferenceService *services.ConferenceRecordService,
	logger *zap.Logger,
) *ConferenceRecordHandler {
	return &ConferenceRecordHandler{
		conferenceService: conferenceService,
		logger:            logger,
	}
}

// ListConferences 获取会议列表
// @Summary 获取会议列表
// @Description 分页获取会议列表，支持筛选
// @Tags 会议管理
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param status query string false "会议状态"
// @Param conference_number query string false "会议号"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} response.Response{data=services.ListConferencesResponse}
// @Router /api/v1/conferences [get]
func (h *ConferenceRecordHandler) ListConferences(c *gin.Context) {
	var req services.ListConferencesRequest
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

	result, err := h.conferenceService.ListConferences(&req)
	if err != nil {
		h.logger.Error("Failed to list conferences", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取会议列表失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetConference 获取会议详情
// @Summary 获取会议详情
// @Description 根据ID获取会议详细信息
// @Tags 会议管理
// @Security Bearer
// @Param id path int true "会议ID"
// @Success 200 {object} response.Response{data=models.ConferenceRecord}
// @Router /api/v1/conferences/{id} [get]
func (h *ConferenceRecordHandler) GetConference(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的会议ID")
		return
	}

	conference, err := h.conferenceService.GetConferenceByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "会议不存在")
		return
	}

	response.GinSuccess(c, conference)
}

// CreateConference 创建会议
// @Summary 创建会议
// @Description 创建新的会议记录
// @Tags 会议管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateConferenceRequest true "创建会议请求"
// @Success 200 {object} response.Response{data=models.ConferenceRecord}
// @Router /api/v1/conferences [post]
func (h *ConferenceRecordHandler) CreateConference(c *gin.Context) {
	var req services.CreateConferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	conference, err := h.conferenceService.CreateConference(&req)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Conference created", zap.Uint("conference_id", conference.ID), zap.String("title", conference.Title))
	response.GinSuccess(c, conference)
}

// UpdateConference 更新会议
// @Summary 更新会议
// @Description 更新会议信息
// @Tags 会议管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "会议ID"
// @Param request body services.UpdateConferenceRequest true "更新会议请求"
// @Success 200 {object} response.Response{data=models.ConferenceRecord}
// @Router /api/v1/conferences/{id} [put]
func (h *ConferenceRecordHandler) UpdateConference(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的会议ID")
		return
	}

	var req services.UpdateConferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	conference, err := h.conferenceService.UpdateConference(id, &req)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Conference updated", zap.Uint("conference_id", id))
	response.GinSuccess(c, conference)
}

// DeleteConference 删除会议
// @Summary 删除会议
// @Description 删除指定的会议记录
// @Tags 会议管理
// @Security Bearer
// @Param id path int true "会议ID"
// @Success 200 {object} response.Response
// @Router /api/v1/conferences/{id} [delete]
func (h *ConferenceRecordHandler) DeleteConference(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的会议ID")
		return
	}

	if err := h.conferenceService.DeleteConference(id); err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Conference deleted", zap.Uint("conference_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// GetConferencesByStatus 根据状态获取会议列表
// @Summary 根据状态获取会议列表
// @Description 根据指定状态获取会议列表
// @Tags 会议管理
// @Security Bearer
// @Param status query string true "会议状态"
// @Success 200 {object} response.Response{data=[]models.ConferenceRecord}
// @Router /api/v1/conferences/by-status [get]
func (h *ConferenceRecordHandler) GetConferencesByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		response.GinError(c, response.CodeInvalidRequest, "缺少状态参数")
		return
	}

	conferences, err := h.conferenceService.GetConferencesByStatus(models.ConferenceStatus(status))
	if err != nil {
		h.logger.Error("Failed to get conferences by status", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取会议列表失败")
		return
	}

	response.GinSuccess(c, conferences)
}
