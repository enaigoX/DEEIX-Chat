package imagestudio

import (
	"slices"
	"testing"

	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
	models "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/persistence/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteTasksOnlyReturnsUnsharedOutputsAndThumbnails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:image-studio-delete?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err = db.AutoMigrate(&models.ImageStudioTask{}, &models.ImageStudioTaskAsset{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewRepo(db)
	first := createDeleteTestTask(t, repo, "task-a")
	second := createDeleteTestTask(t, repo, "task-b")
	inputConsumer := createDeleteTestTask(t, repo, "task-input-consumer")
	if err = repo.AddTaskAssets(t.Context(), first.ID, []domainimagestudio.Asset{
		{TaskPublicID: first.TaskID, UserID: 7, Role: domainimagestudio.AssetRoleOutput, FileID: "shared-original", ThumbnailFileID: "thumb-a"},
		{TaskPublicID: first.TaskID, UserID: 7, Role: domainimagestudio.AssetRoleInput, FileID: "input-file"},
	}); err != nil {
		t.Fatalf("add first assets: %v", err)
	}
	if err = repo.AddTaskAssets(t.Context(), second.ID, []domainimagestudio.Asset{
		{TaskPublicID: second.TaskID, UserID: 7, Role: domainimagestudio.AssetRoleOutput, FileID: "shared-original", ThumbnailFileID: "thumb-b"},
	}); err != nil {
		t.Fatalf("add second assets: %v", err)
	}
	if err = repo.AddTaskAssets(t.Context(), inputConsumer.ID, []domainimagestudio.Asset{
		{TaskPublicID: inputConsumer.TaskID, UserID: 7, Role: domainimagestudio.AssetRoleInput, FileID: "shared-original"},
	}); err != nil {
		t.Fatalf("add input consumer asset: %v", err)
	}

	firstPlan, err := repo.DeleteTasks(t.Context(), 7, []string{"task-a", "missing"})
	if err != nil {
		t.Fatalf("delete first task: %v", err)
	}
	if !slices.Equal(firstPlan.DeletedTaskIDs, []string{"task-a"}) {
		t.Fatalf("deleted task IDs = %#v, want task-a", firstPlan.DeletedTaskIDs)
	}
	if !slices.Equal(firstPlan.CleanupFileIDs, []string{"thumb-a"}) {
		t.Fatalf("first cleanup IDs = %#v, want only thumb-a", firstPlan.CleanupFileIDs)
	}

	secondPlan, err := repo.DeleteTasks(t.Context(), 7, []string{"task-b"})
	if err != nil {
		t.Fatalf("delete second task: %v", err)
	}
	if !slices.Equal(secondPlan.CleanupFileIDs, []string{"thumb-b"}) {
		t.Fatalf("second cleanup IDs = %#v, want only thumb-b while original is used as input", secondPlan.CleanupFileIDs)
	}

	consumerPlan, err := repo.DeleteTasks(t.Context(), 7, []string{"task-input-consumer"})
	if err != nil {
		t.Fatalf("delete input consumer task: %v", err)
	}
	if !slices.Equal(consumerPlan.CleanupFileIDs, []string{"shared-original"}) {
		t.Fatalf("last generated-image input reference cleanup = %#v, want shared-original", consumerPlan.CleanupFileIDs)
	}
}

func createDeleteTestTask(t *testing.T, repo *Repo, taskID string) *domainimagestudio.Task {
	t.Helper()
	task := &domainimagestudio.Task{
		TaskID:   taskID,
		UserID:   7,
		TaskType: domainimagestudio.TaskTypeGeneration,
		Prompt:   "test",
		Status:   domainimagestudio.TaskStatusSuccess,
	}
	if err := repo.CreateTask(t.Context(), task); err != nil {
		t.Fatalf("create task %s: %v", taskID, err)
	}
	return task
}
