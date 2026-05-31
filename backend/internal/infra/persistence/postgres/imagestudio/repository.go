package imagestudio

import (
	"context"
	"errors"
	"strings"
	"time"

	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
	models "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"gorm.io/gorm"
)

// Repo implements Image Studio persistence.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates an Image Studio repository.
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// CreateTask creates an Image Studio task.
func (r *Repo) CreateTask(ctx context.Context, task *domainimagestudio.Task) error {
	if task == nil {
		return repository.ErrInvalidInput
	}
	entity := toTaskModel(task)
	if err := r.db.WithContext(ctx).Create(&entity).Error; err != nil {
		return translateError(err)
	}
	*task = toTaskDomain(entity, nil)
	return nil
}

// GetTaskByPublicID loads one task and its assets.
func (r *Repo) GetTaskByPublicID(ctx context.Context, userID uint, taskID string) (*domainimagestudio.Task, error) {
	var item models.ImageStudioTask
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND task_id = ? AND status <> ?", userID, strings.TrimSpace(taskID), domainimagestudio.TaskStatusDeleted).
		First(&item).Error; err != nil {
		return nil, translateError(err)
	}
	assets, err := r.listAssetsByTaskIDs(ctx, []uint{item.ID})
	if err != nil {
		return nil, err
	}
	result := toTaskDomain(item, assets[item.ID])
	return &result, nil
}

// ListTasks lists current user's task history.
func (r *Repo) ListTasks(
	ctx context.Context,
	userID uint,
	filter domainimagestudio.TaskListFilter,
	offset int,
	limit int,
) ([]domainimagestudio.Task, int64, error) {
	items := make([]models.ImageStudioTask, 0)
	var total int64
	query := r.db.WithContext(ctx).
		Model(&models.ImageStudioTask{}).
		Where("user_id = ? AND status <> ?", userID, domainimagestudio.TaskStatusDeleted)

	switch strings.TrimSpace(filter.Status) {
	case "", "all":
	case domainimagestudio.TaskStatusQueued, domainimagestudio.TaskStatusRunning, domainimagestudio.TaskStatusSuccess, domainimagestudio.TaskStatusError:
		query = query.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	switch strings.TrimSpace(filter.Favorite) {
	case "favorite", "favorites", "true":
		query = query.Where("is_favorite = ?", true)
	case "unfavorite", "unfavorites", "false":
		query = query.Where("is_favorite = ?", false)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query = query.Where("(LOWER(prompt) LIKE ? OR LOWER(platform_model_name) LIKE ? OR LOWER(requested_model_name) LIKE ?)", like, like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, translateError(err)
	}
	if err := query.
		Order("updated_at DESC").
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, 0, translateError(err)
	}
	taskIDs := make([]uint, 0, len(items))
	for _, item := range items {
		taskIDs = append(taskIDs, item.ID)
	}
	assetMap, err := r.listAssetsByTaskIDs(ctx, taskIDs)
	if err != nil {
		return nil, 0, err
	}
	results := make([]domainimagestudio.Task, 0, len(items))
	for _, item := range items {
		results = append(results, toTaskDomain(item, assetMap[item.ID]))
	}
	return results, total, nil
}

// UpdateTask applies partial task updates.
func (r *Repo) UpdateTask(ctx context.Context, userID uint, taskID string, patch domainimagestudio.TaskPatch) (*domainimagestudio.Task, error) {
	updates := taskPatchUpdates(patch)
	if len(updates) == 0 {
		return r.GetTaskByPublicID(ctx, userID, taskID)
	}
	var item models.ImageStudioTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("user_id = ? AND task_id = ? AND status <> ?", userID, strings.TrimSpace(taskID), domainimagestudio.TaskStatusDeleted).
			First(&item).Error; err != nil {
			return translateError(err)
		}
		if err := tx.Model(&item).Updates(updates).Error; err != nil {
			return translateError(err)
		}
		return tx.First(&item, item.ID).Error
	})
	if err != nil {
		return nil, translateError(err)
	}
	assets, err := r.listAssetsByTaskIDs(ctx, []uint{item.ID})
	if err != nil {
		return nil, err
	}
	result := toTaskDomain(item, assets[item.ID])
	return &result, nil
}

// AddTaskAssets appends assets to a task.
func (r *Repo) AddTaskAssets(ctx context.Context, taskID uint, assets []domainimagestudio.Asset) error {
	if len(assets) == 0 {
		return nil
	}
	entities := make([]models.ImageStudioTaskAsset, 0, len(assets))
	for _, asset := range assets {
		asset.TaskID = taskID
		entities = append(entities, toAssetModel(asset))
	}
	return translateError(r.db.WithContext(ctx).Create(&entities).Error)
}

// ReplaceTaskAssetsByRole replaces assets of one role.
func (r *Repo) ReplaceTaskAssetsByRole(ctx context.Context, taskID uint, role string, assets []domainimagestudio.Asset) error {
	normalizedRole := strings.TrimSpace(role)
	if taskID == 0 || normalizedRole == "" {
		return repository.ErrInvalidInput
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("task_id = ? AND role = ?", taskID, normalizedRole).
			Delete(&models.ImageStudioTaskAsset{}).Error; err != nil {
			return translateError(err)
		}
		if len(assets) == 0 {
			return nil
		}
		entities := make([]models.ImageStudioTaskAsset, 0, len(assets))
		for _, asset := range assets {
			asset.TaskID = taskID
			asset.Role = normalizedRole
			entities = append(entities, toAssetModel(asset))
		}
		return translateError(tx.Create(&entities).Error)
	})
}

// SetTaskFavorite toggles favorite state.
func (r *Repo) SetTaskFavorite(ctx context.Context, userID uint, taskID string, favorite bool) (*domainimagestudio.Task, error) {
	var favoritedAt *time.Time
	if favorite {
		now := time.Now()
		favoritedAt = &now
	}
	result := r.db.WithContext(ctx).
		Model(&models.ImageStudioTask{}).
		Where("user_id = ? AND task_id = ? AND status <> ?", userID, strings.TrimSpace(taskID), domainimagestudio.TaskStatusDeleted).
		Updates(map[string]interface{}{
			"is_favorite":  favorite,
			"favorited_at": favoritedAt,
		})
	if result.Error != nil {
		return nil, translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return r.GetTaskByPublicID(ctx, userID, taskID)
}

// DeleteTask marks a task deleted without touching file objects.
func (r *Repo) DeleteTask(ctx context.Context, userID uint, taskID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&models.ImageStudioTask{}).
		Where("user_id = ? AND task_id = ? AND status <> ?", userID, strings.TrimSpace(taskID), domainimagestudio.TaskStatusDeleted).
		Updates(map[string]interface{}{
			"status":       domainimagestudio.TaskStatusDeleted,
			"completed_at": &now,
		})
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *Repo) listAssetsByTaskIDs(ctx context.Context, taskIDs []uint) (map[uint][]domainimagestudio.Asset, error) {
	results := make(map[uint][]domainimagestudio.Asset, len(taskIDs))
	if len(taskIDs) == 0 {
		return results, nil
	}
	items := make([]models.ImageStudioTaskAsset, 0)
	if err := r.db.WithContext(ctx).
		Where("task_id IN ?", taskIDs).
		Order("task_id ASC").
		Order("sort_order ASC").
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, translateError(err)
	}
	for _, item := range items {
		results[item.TaskID] = append(results[item.TaskID], toAssetDomain(item))
	}
	return results, nil
}

func taskPatchUpdates(patch domainimagestudio.TaskPatch) map[string]interface{} {
	updates := make(map[string]interface{})
	setString := func(column string, value *string) {
		if value != nil {
			updates[column] = strings.TrimSpace(*value)
		}
	}
	setUint := func(column string, value *uint) {
		if value != nil {
			updates[column] = *value
		}
	}
	setInt64 := func(column string, value *int64) {
		if value != nil {
			updates[column] = *value
		}
	}
	setString("status", patch.Status)
	setString("requested_model_name", patch.RequestedModelName)
	setString("platform_model_name", patch.PlatformModelName)
	setString("provider_protocol", patch.ProviderProtocol)
	setUint("upstream_id", patch.UpstreamID)
	setUint("upstream_model_id", patch.UpstreamModelID)
	setString("upstream_name", patch.UpstreamName)
	setString("routed_binding_code", patch.RoutedBindingCode)
	setString("model_vendor", patch.ModelVendor)
	setString("model_icon", patch.ModelIcon)
	setString("upstream_model_name", patch.UpstreamModelName)
	setString("options_json", patch.OptionsJSON)
	setString("route_snapshot_json", patch.RouteSnapshotJSON)
	setString("usage_snapshot_json", patch.UsageSnapshotJSON)
	setString("billing_snapshot_json", patch.BillingSnapshotJSON)
	setInt64("input_tokens", patch.InputTokens)
	setInt64("output_tokens", patch.OutputTokens)
	setInt64("cache_read_tokens", patch.CacheReadTokens)
	setInt64("cache_write_tokens", patch.CacheWriteTokens)
	setInt64("reasoning_tokens", patch.ReasoningTokens)
	setInt64("latency_ms", patch.LatencyMS)
	setString("billed_currency", patch.BilledCurrency)
	setInt64("billed_nanousd", patch.BilledNanousd)
	setString("error_code", patch.ErrorCode)
	setString("error_message", patch.ErrorMessage)
	if patch.StartedAt != nil {
		updates["started_at"] = *patch.StartedAt
	}
	if patch.CompletedAt != nil {
		updates["completed_at"] = *patch.CompletedAt
	}
	return updates
}

func toTaskModel(item *domainimagestudio.Task) models.ImageStudioTask {
	return models.ImageStudioTask{
		TaskID:              strings.TrimSpace(item.TaskID),
		UserID:              item.UserID,
		TaskType:            strings.TrimSpace(item.TaskType),
		Prompt:              strings.TrimSpace(item.Prompt),
		Status:              strings.TrimSpace(item.Status),
		RequestedModelName:  strings.TrimSpace(item.RequestedModelName),
		PlatformModelName:   strings.TrimSpace(item.PlatformModelName),
		ProviderProtocol:    strings.TrimSpace(item.ProviderProtocol),
		UpstreamID:          item.UpstreamID,
		UpstreamModelID:     item.UpstreamModelID,
		UpstreamName:        strings.TrimSpace(item.UpstreamName),
		RoutedBindingCode:   strings.TrimSpace(item.RoutedBindingCode),
		ModelVendor:         strings.TrimSpace(item.ModelVendor),
		ModelIcon:           strings.TrimSpace(item.ModelIcon),
		UpstreamModelName:   strings.TrimSpace(item.UpstreamModelName),
		OptionsJSON:         defaultJSON(item.OptionsJSON),
		RouteSnapshotJSON:   defaultJSON(item.RouteSnapshotJSON),
		UsageSnapshotJSON:   defaultJSON(item.UsageSnapshotJSON),
		BillingSnapshotJSON: strings.TrimSpace(item.BillingSnapshotJSON),
		InputTokens:         item.InputTokens,
		OutputTokens:        item.OutputTokens,
		CacheReadTokens:     item.CacheReadTokens,
		CacheWriteTokens:    item.CacheWriteTokens,
		ReasoningTokens:     item.ReasoningTokens,
		LatencyMS:           item.LatencyMS,
		BilledCurrency:      defaultString(item.BilledCurrency, "USD"),
		BilledNanousd:       item.BilledNanousd,
		IsFavorite:          item.IsFavorite,
		FavoritedAt:         item.FavoritedAt,
		ErrorCode:           strings.TrimSpace(item.ErrorCode),
		ErrorMessage:        truncateString(item.ErrorMessage, 255),
		StartedAt:           item.StartedAt,
		CompletedAt:         item.CompletedAt,
	}
}

func toTaskDomain(item models.ImageStudioTask, assets []domainimagestudio.Asset) domainimagestudio.Task {
	return domainimagestudio.Task{
		ID:                  item.ID,
		TaskID:              item.TaskID,
		UserID:              item.UserID,
		TaskType:            item.TaskType,
		Prompt:              item.Prompt,
		Status:              item.Status,
		RequestedModelName:  item.RequestedModelName,
		PlatformModelName:   item.PlatformModelName,
		ProviderProtocol:    item.ProviderProtocol,
		UpstreamID:          item.UpstreamID,
		UpstreamModelID:     item.UpstreamModelID,
		UpstreamName:        item.UpstreamName,
		RoutedBindingCode:   item.RoutedBindingCode,
		ModelVendor:         item.ModelVendor,
		ModelIcon:           item.ModelIcon,
		UpstreamModelName:   item.UpstreamModelName,
		OptionsJSON:         defaultJSON(item.OptionsJSON),
		RouteSnapshotJSON:   defaultJSON(item.RouteSnapshotJSON),
		UsageSnapshotJSON:   defaultJSON(item.UsageSnapshotJSON),
		BillingSnapshotJSON: item.BillingSnapshotJSON,
		InputTokens:         item.InputTokens,
		OutputTokens:        item.OutputTokens,
		CacheReadTokens:     item.CacheReadTokens,
		CacheWriteTokens:    item.CacheWriteTokens,
		ReasoningTokens:     item.ReasoningTokens,
		LatencyMS:           item.LatencyMS,
		BilledCurrency:      defaultString(item.BilledCurrency, "USD"),
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

func toAssetModel(item domainimagestudio.Asset) models.ImageStudioTaskAsset {
	return models.ImageStudioTaskAsset{
		TaskID:           item.TaskID,
		TaskPublicID:     strings.TrimSpace(item.TaskPublicID),
		UserID:           item.UserID,
		FileID:           strings.TrimSpace(item.FileID),
		Role:             strings.TrimSpace(item.Role),
		SortOrder:        item.SortOrder,
		FileName:         strings.TrimSpace(item.FileName),
		MimeType:         strings.TrimSpace(item.MimeType),
		FileSize:         item.FileSize,
		SHA256:           strings.TrimSpace(item.SHA256),
		StoragePath:      strings.TrimSpace(item.StoragePath),
		FileSnapshotJSON: defaultJSON(item.FileSnapshotJSON),
		MetadataJSON:     defaultJSON(item.MetadataJSON),
	}
}

func toAssetDomain(item models.ImageStudioTaskAsset) domainimagestudio.Asset {
	return domainimagestudio.Asset{
		ID:               item.ID,
		TaskID:           item.TaskID,
		TaskPublicID:     item.TaskPublicID,
		UserID:           item.UserID,
		FileID:           item.FileID,
		Role:             item.Role,
		SortOrder:        item.SortOrder,
		FileName:         item.FileName,
		MimeType:         item.MimeType,
		FileSize:         item.FileSize,
		SHA256:           item.SHA256,
		StoragePath:      item.StoragePath,
		FileSnapshotJSON: defaultJSON(item.FileSnapshotJSON),
		MetadataJSON:     defaultJSON(item.MetadataJSON),
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) || isUniqueConstraintError(err) {
		return repository.ErrDuplicate
	}
	return err
}

type sqlStateError interface {
	SQLState() string
}

func isUniqueConstraintError(err error) bool {
	var stateErr sqlStateError
	if errors.As(err, &stateErr) && stateErr.SQLState() == "23505" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}

func defaultJSON(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "{}"
	}
	return normalized
}

func defaultString(value string, fallback string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fallback
	}
	return normalized
}

func truncateString(value string, maxChars int) string {
	runes := []rune(strings.TrimSpace(value))
	if maxChars <= 0 || len(runes) <= maxChars {
		return string(runes)
	}
	return string(runes[:maxChars])
}
