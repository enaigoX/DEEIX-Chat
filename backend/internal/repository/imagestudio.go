package repository

import (
	"context"

	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
)

// ImageStudioRepository defines persistence for the independent Image Studio module.
type ImageStudioRepository interface {
	CreateTask(ctx context.Context, task *domainimagestudio.Task) error
	GetTaskByPublicID(ctx context.Context, userID uint, taskID string) (*domainimagestudio.Task, error)
	ListTasks(ctx context.Context, userID uint, filter domainimagestudio.TaskListFilter, offset int, limit int) ([]domainimagestudio.Task, int64, error)
	UpdateTask(ctx context.Context, userID uint, taskID string, patch domainimagestudio.TaskPatch) (*domainimagestudio.Task, error)
	AddTaskAssets(ctx context.Context, taskID uint, assets []domainimagestudio.Asset) error
	ReplaceTaskAssetsByRole(ctx context.Context, taskID uint, role string, assets []domainimagestudio.Asset) error
	SetTaskFavorite(ctx context.Context, userID uint, taskID string, favorite bool) (*domainimagestudio.Task, error)
	DeleteTasks(ctx context.Context, userID uint, taskIDs []string) (*domainimagestudio.DeletePlan, error)
}
