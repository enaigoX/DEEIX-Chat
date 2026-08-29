package db

import "gorm.io/gorm"

func applyImageStudioBaselineIndexes(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE "image_studio_tasks"
			ALTER COLUMN "task_id" TYPE varchar(64)`,
		`ALTER TABLE "image_studio_task_assets"
			ALTER COLUMN "task_public_id" TYPE varchar(64)`,
		`CREATE INDEX IF NOT EXISTS idx_image_studio_tasks_user_updated
			ON "image_studio_tasks" ("user_id", "updated_at" DESC, "id" DESC)
			WHERE status <> 'deleted'`,
		`CREATE INDEX IF NOT EXISTS idx_image_studio_tasks_user_favorite
			ON "image_studio_tasks" ("user_id", "is_favorite", "updated_at" DESC)
			WHERE status <> 'deleted'`,
		`CREATE INDEX IF NOT EXISTS idx_image_studio_task_assets_task_role
			ON "image_studio_task_assets" ("task_id", "role", "sort_order")`,
		`CREATE INDEX IF NOT EXISTS idx_image_studio_task_assets_user_file
			ON "image_studio_task_assets" ("user_id", "file_id")`,
		`CREATE INDEX IF NOT EXISTS idx_image_studio_task_assets_user_thumbnail_file
			ON "image_studio_task_assets" ("user_id", "thumbnail_file_id")`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
