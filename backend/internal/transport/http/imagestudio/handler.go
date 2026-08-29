package imagestudio

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	appbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	appimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/imagestudio"
	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/response"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// Handler handles Image Studio HTTP requests.
type Handler struct {
	service *appimagestudio.Service
}

// NewHandler creates an Image Studio handler.
func NewHandler(service *appimagestudio.Service) *Handler {
	return &Handler{service: service}
}

// StreamTask creates and streams one Image Studio task.
func (h *Handler) StreamTask(c *gin.Context) {
	userID := middleware.MustUserID(c)
	var req StreamTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	taskInput := toServiceInput(userID, middleware.MustRequestID(c), req)

	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	var clientDisconnected atomic.Bool
	flushStreamEvent := func(payload map[string]interface{}) error {
		if clientDisconnected.Load() {
			return nil
		}
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return marshalErr
		}
		if _, writeErr := c.Writer.Write(append(encoded, '\n')); writeErr != nil {
			clientDisconnected.Store(true)
			return writeErr
		}
		c.Writer.Flush()
		return nil
	}

	taskInput.OnEvent = func(eventType string, payload map[string]interface{}) error {
		_ = flushStreamEvent(normalizeStreamEventPayload(eventType, payload))
		return nil
	}
	_, err := h.service.StreamTask(c.Request.Context(), taskInput)
	if err != nil {
		_ = flushStreamEvent(streamErrorPayload(err))
		return
	}
}

// ListTasks returns task history.
func (h *Handler) ListTasks(c *gin.Context) {
	userID := middleware.MustUserID(c)
	page, pageSize := pageParams(c)
	filter := domainimagestudio.TaskListFilter{
		Status:   strings.TrimSpace(c.Query("status")),
		Favorite: strings.TrimSpace(c.Query("favorite")),
		Query:    strings.TrimSpace(c.Query("q")),
	}
	items, total, err := h.service.ListTasks(c.Request.Context(), userID, page, pageSize, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "list image studio tasks failed")
		return
	}
	results := make([]TaskResponse, 0, len(items))
	for i := range items {
		results = append(results, toTaskResponse(&items[i]))
	}
	response.SuccessPage(c, total, results)
}

// GetTask returns one task.
func (h *Handler) GetTask(c *gin.Context) {
	userID := middleware.MustUserID(c)
	task, err := h.service.GetTask(c.Request.Context(), userID, c.Param("task_id"))
	if err != nil {
		if errors.Is(err, appimagestudio.ErrTaskNotFound) {
			response.Error(c, http.StatusNotFound, "image studio task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "get image studio task failed")
		return
	}
	response.Success(c, toTaskResponse(task))
}

// SetFavorite toggles favorite.
func (h *Handler) SetFavorite(c *gin.Context) {
	userID := middleware.MustUserID(c)
	var req SetFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	task, err := h.service.SetFavorite(c.Request.Context(), userID, c.Param("task_id"), req.Favorite)
	if err != nil {
		if errors.Is(err, appimagestudio.ErrTaskNotFound) {
			response.Error(c, http.StatusNotFound, "image studio task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "set favorite failed")
		return
	}
	response.Success(c, toTaskResponse(task))
}

// DeleteTask deletes task history.
func (h *Handler) DeleteTask(c *gin.Context) {
	userID := middleware.MustUserID(c)
	result, err := h.service.DeleteTask(c.Request.Context(), userID, c.Param("task_id"))
	if err != nil {
		if errors.Is(err, appimagestudio.ErrTaskNotFound) {
			response.Error(c, http.StatusNotFound, "image studio task not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "delete image studio task failed")
		return
	}
	response.Success(c, gin.H{
		"deleted":              true,
		"cleanupFailedFileIDs": result.CleanupFailedFileIDs,
	})
}

// BatchDeleteTasks deletes multiple task histories and their generated files.
func (h *Handler) BatchDeleteTasks(c *gin.Context) {
	userID := middleware.MustUserID(c)
	var req BatchDeleteTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.DeleteTasks(c.Request.Context(), userID, req.TaskIDs)
	if err != nil {
		if errors.Is(err, appimagestudio.ErrInvalidTask) {
			response.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "batch delete image studio tasks failed")
		return
	}
	response.Success(c, BatchDeleteTasksResponse{
		DeletedTaskIDs:       result.DeletedTaskIDs,
		MissingTaskIDs:       result.MissingTaskIDs,
		CleanupFailedFileIDs: result.CleanupFailedFileIDs,
	})
}

func pageParams(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			pageSize = parsed
		}
	}
	return page, pageSize
}

func normalizeStreamEventPayload(eventType string, payload map[string]interface{}) map[string]interface{} {
	normalized := map[string]interface{}{"type": eventType}
	for key, value := range payload {
		switch typed := value.(type) {
		case *domainimagestudio.Task:
			normalized[key] = toTaskResponse(typed)
		case domainimagestudio.Task:
			task := typed
			normalized[key] = toTaskResponse(&task)
		default:
			normalized[key] = value
		}
	}
	if eventType == "completed" {
		if data, ok := payload["data"].(*domainimagestudio.Task); ok {
			normalized["data"] = toTaskResponse(data)
		}
	}
	return normalized
}

func streamErrorPayload(err error) map[string]interface{} {
	status := http.StatusInternalServerError
	message := "image task failed"
	errorCode := "internal_error"
	switch {
	case errors.Is(err, appimagestudio.ErrPromptRequired),
		errors.Is(err, appimagestudio.ErrGenerationRejectsInputs),
		errors.Is(err, appimagestudio.ErrEditInputRequired),
		errors.Is(err, appimagestudio.ErrEditTooManyInputs),
		errors.Is(err, appimagestudio.ErrEditInputInvalid),
		errors.Is(err, appimagestudio.ErrInvalidTask),
		errors.Is(err, appimagestudio.ErrInvalidFileReference):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, appimagestudio.ErrFileNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, appimagestudio.ErrStorageQuotaExceeded):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, appimagestudio.ErrFileTooLarge):
		status = http.StatusRequestEntityTooLarge
		message = err.Error()
	case errors.Is(err, appbilling.ErrPeriodCreditExceeded),
		errors.Is(err, appbilling.ErrModelPricingRequired),
		errors.Is(err, appbilling.ErrUsageBalanceInsufficient):
		status = http.StatusPaymentRequired
		message = err.Error()
	case errors.Is(err, appimagestudio.ErrRouteNotConfigured),
		errors.Is(err, appimagestudio.ErrRouteProtocolMismatch):
		status = http.StatusBadRequest
		message = err.Error()
	default:
		if trimmed := strings.TrimSpace(err.Error()); trimmed != "" {
			message = trimmed
		}
	}
	if status == http.StatusPaymentRequired {
		errorCode = "payment_required"
	}
	return map[string]interface{}{
		"type":      "error",
		"status":    status,
		"message":   message,
		"errorCode": errorCode,
	}
}
