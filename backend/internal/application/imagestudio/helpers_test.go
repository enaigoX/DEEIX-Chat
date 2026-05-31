package imagestudio

import "testing"

func TestNewImageStudioTaskIDFitsStorageLimit(t *testing.T) {
	for range 100 {
		taskID := newImageStudioTaskID()
		if len(taskID) > 32 {
			t.Fatalf("task ID length = %d, want <= 32: %q", len(taskID), taskID)
		}
		if len(taskID) == 0 || taskID[:4] != "img_" {
			t.Fatalf("task ID should keep img_ prefix: %q", taskID)
		}
	}
}
