package imagestudio

import (
	"strings"
	"time"

	appimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/imagestudio"
	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
)

// StreamTaskRequest starts an Image Studio generation or edit task.
type StreamTaskRequest struct {
	TaskType   string                 `json:"taskType" binding:"omitempty,oneof=image_generation image_edit"`
	Prompt     string                 `json:"prompt" binding:"required,max=20000"`
	Model      string                 `json:"model" binding:"required,max=128"`
	Options    map[string]interface{} `json:"options"`
	FileIDs    []string               `json:"fileIDs" binding:"max=20"`
	MaskFileID string                 `json:"maskFileID" binding:"omitempty,max=128"`
}

// SetFavoriteRequest toggles a task favorite state.
type SetFavoriteRequest struct {
	Favorite bool `json:"favorite"`
}

// BatchDeleteTasksRequest deletes a bounded set of Image Studio tasks.
type BatchDeleteTasksRequest struct {
	TaskIDs []string `json:"taskIDs" binding:"required,min=1,max=100,dive,required,max=64"`
}

// BatchDeleteTasksResponse reports database deletion and storage cleanup status.
type BatchDeleteTasksResponse struct {
	DeletedTaskIDs       []string `json:"deletedTaskIDs"`
	MissingTaskIDs       []string `json:"missingTaskIDs"`
	CleanupFailedFileIDs []string `json:"cleanupFailedFileIDs"`
}

// TaskResponse is the public Image Studio task DTO.
type TaskResponse struct {
	TaskID              string          `json:"taskID"`
	UserID              uint            `json:"userID"`
	TaskType            string          `json:"taskType"`
	Prompt              string          `json:"prompt"`
	Status              string          `json:"status"`
	RequestedModelName  string          `json:"requestedModelName"`
	PlatformModelName   string          `json:"platformModelName"`
	ProviderProtocol    string          `json:"providerProtocol"`
	UpstreamName        string          `json:"upstreamName"`
	RoutedBindingCode   string          `json:"routedBindingCode"`
	ModelVendor         string          `json:"modelVendor"`
	ModelIcon           string          `json:"modelIcon"`
	UpstreamModelName   string          `json:"upstreamModelName"`
	OptionsJSON         string          `json:"optionsJSON"`
	RouteSnapshotJSON   string          `json:"routeSnapshotJSON"`
	UsageSnapshotJSON   string          `json:"usageSnapshotJSON"`
	BillingSnapshotJSON string          `json:"billingSnapshotJSON"`
	InputTokens         int64           `json:"inputTokens"`
	OutputTokens        int64           `json:"outputTokens"`
	CacheReadTokens     int64           `json:"cacheReadTokens"`
	CacheWriteTokens    int64           `json:"cacheWriteTokens"`
	ReasoningTokens     int64           `json:"reasoningTokens"`
	LatencyMS           int64           `json:"latencyMS"`
	BilledCurrency      string          `json:"billedCurrency"`
	BilledNanousd       int64           `json:"billedNanousd"`
	IsFavorite          bool            `json:"isFavorite"`
	FavoritedAt         *time.Time      `json:"favoritedAt"`
	ErrorCode           string          `json:"errorCode"`
	ErrorMessage        string          `json:"errorMessage"`
	StartedAt           *time.Time      `json:"startedAt"`
	CompletedAt         *time.Time      `json:"completedAt"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
	Assets              []AssetResponse `json:"assets"`
}

// AssetResponse is the public Image Studio asset DTO.
type AssetResponse struct {
	FileID            string    `json:"fileID"`
	Role              string    `json:"role"`
	SortOrder         int       `json:"sortOrder"`
	FileName          string    `json:"fileName"`
	MimeType          string    `json:"mimeType"`
	FileSize          int64     `json:"fileSize"`
	SHA256            string    `json:"sha256"`
	ThumbnailFileID   string    `json:"thumbnailFileID"`
	ThumbnailFileName string    `json:"thumbnailFileName"`
	ThumbnailMimeType string    `json:"thumbnailMimeType"`
	ThumbnailFileSize int64     `json:"thumbnailFileSize"`
	ThumbnailWidth    int       `json:"thumbnailWidth"`
	ThumbnailHeight   int       `json:"thumbnailHeight"`
	FileSnapshotJSON  string    `json:"fileSnapshotJSON"`
	MetadataJSON      string    `json:"metadataJSON"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func toServiceInput(userID uint, requestID string, req StreamTaskRequest) appimagestudio.StreamTaskInput {
	taskType := strings.TrimSpace(req.TaskType)
	if taskType == "" {
		taskType = domainimagestudio.TaskTypeGeneration
	}
	return appimagestudio.StreamTaskInput{
		UserID:            userID,
		RequestID:         strings.TrimSpace(requestID),
		TaskType:          taskType,
		Prompt:            req.Prompt,
		PlatformModelName: req.Model,
		Options:           req.Options,
		FileIDs:           req.FileIDs,
		MaskFileID:        req.MaskFileID,
	}
}

func toTaskResponse(item *domainimagestudio.Task) TaskResponse {
	if item == nil {
		return TaskResponse{}
	}
	assets := make([]AssetResponse, 0, len(item.Assets))
	for i := range item.Assets {
		assets = append(assets, toAssetResponse(item.Assets[i]))
	}
	return TaskResponse{
		TaskID:              item.TaskID,
		UserID:              item.UserID,
		TaskType:            item.TaskType,
		Prompt:              item.Prompt,
		Status:              item.Status,
		RequestedModelName:  item.RequestedModelName,
		PlatformModelName:   item.PlatformModelName,
		ProviderProtocol:    item.ProviderProtocol,
		UpstreamName:        item.UpstreamName,
		RoutedBindingCode:   item.RoutedBindingCode,
		ModelVendor:         item.ModelVendor,
		ModelIcon:           item.ModelIcon,
		UpstreamModelName:   item.UpstreamModelName,
		OptionsJSON:         item.OptionsJSON,
		RouteSnapshotJSON:   item.RouteSnapshotJSON,
		UsageSnapshotJSON:   item.UsageSnapshotJSON,
		BillingSnapshotJSON: item.BillingSnapshotJSON,
		InputTokens:         item.InputTokens,
		OutputTokens:        item.OutputTokens,
		CacheReadTokens:     item.CacheReadTokens,
		CacheWriteTokens:    item.CacheWriteTokens,
		ReasoningTokens:     item.ReasoningTokens,
		LatencyMS:           item.LatencyMS,
		BilledCurrency:      item.BilledCurrency,
		BilledNanousd:       item.BilledNanousd,
		IsFavorite:          item.IsFavorite,
		FavoritedAt:         item.FavoritedAt,
		ErrorCode:           item.ErrorCode,
		ErrorMessage:        item.ErrorMessage,
		StartedAt:           item.StartedAt,
		CompletedAt:         item.CompletedAt,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
		Assets:              assets,
	}
}

func toAssetResponse(item domainimagestudio.Asset) AssetResponse {
	return AssetResponse{
		FileID:            item.FileID,
		Role:              item.Role,
		SortOrder:         item.SortOrder,
		FileName:          item.FileName,
		MimeType:          item.MimeType,
		FileSize:          item.FileSize,
		SHA256:            item.SHA256,
		ThumbnailFileID:   item.ThumbnailFileID,
		ThumbnailFileName: item.ThumbnailFileName,
		ThumbnailMimeType: item.ThumbnailMimeType,
		ThumbnailFileSize: item.ThumbnailFileSize,
		ThumbnailWidth:    item.ThumbnailWidth,
		ThumbnailHeight:   item.ThumbnailHeight,
		FileSnapshotJSON:  item.FileSnapshotJSON,
		MetadataJSON:      item.MetadataJSON,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}
