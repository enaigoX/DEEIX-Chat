package imagestudio

import "time"

const (
	TaskTypeGeneration = "image_generation"
	TaskTypeEdit       = "image_edit"

	TaskStatusQueued  = "queued"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusError   = "error"
	TaskStatusDeleted = "deleted"

	AssetRoleInput  = "input"
	AssetRoleMask   = "mask"
	AssetRoleOutput = "output"
)

// Task stores one cloud-backed Image Studio generation or edit run.
type Task struct {
	ID                  uint
	TaskID              string
	UserID              uint
	TaskType            string
	Prompt              string
	Status              string
	RequestedModelName  string
	PlatformModelName   string
	ProviderProtocol    string
	UpstreamID          uint
	UpstreamModelID     uint
	UpstreamName        string
	RoutedBindingCode   string
	ModelVendor         string
	ModelIcon           string
	UpstreamModelName   string
	OptionsJSON         string
	RouteSnapshotJSON   string
	UsageSnapshotJSON   string
	BillingSnapshotJSON string
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheWriteTokens    int64
	ReasoningTokens     int64
	LatencyMS           int64
	BilledCurrency      string
	BilledNanousd       int64
	IsFavorite          bool
	FavoritedAt         *time.Time
	ErrorCode           string
	ErrorMessage        string
	StartedAt           *time.Time
	CompletedAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Assets              []Asset
}

// Asset links Image Studio task history to DEEIX file objects.
type Asset struct {
	ID                uint
	TaskID            uint
	TaskPublicID      string
	UserID            uint
	FileID            string
	Role              string
	SortOrder         int
	FileName          string
	MimeType          string
	FileSize          int64
	SHA256            string
	StoragePath       string
	ThumbnailFileID   string
	ThumbnailFileName string
	ThumbnailMimeType string
	ThumbnailFileSize int64
	ThumbnailWidth    int
	ThumbnailHeight   int
	FileSnapshotJSON  string
	MetadataJSON      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// DeletePlan contains task rows hidden by a delete operation and file objects
// that no longer have an active Image Studio output reference.
type DeletePlan struct {
	DeletedTaskIDs []string
	CleanupFileIDs []string
}

// TaskListFilter describes task history filters.
type TaskListFilter struct {
	Status   string
	Favorite string
	Query    string
}

// TaskPatch contains partial task updates.
type TaskPatch struct {
	Status              *string
	RequestedModelName  *string
	PlatformModelName   *string
	ProviderProtocol    *string
	UpstreamID          *uint
	UpstreamModelID     *uint
	UpstreamName        *string
	RoutedBindingCode   *string
	ModelVendor         *string
	ModelIcon           *string
	UpstreamModelName   *string
	OptionsJSON         *string
	RouteSnapshotJSON   *string
	UsageSnapshotJSON   *string
	BillingSnapshotJSON *string
	InputTokens         *int64
	OutputTokens        *int64
	CacheReadTokens     *int64
	CacheWriteTokens    *int64
	ReasoningTokens     *int64
	LatencyMS           *int64
	BilledCurrency      *string
	BilledNanousd       *int64
	ErrorCode           *string
	ErrorMessage        *string
	StartedAt           **time.Time
	CompletedAt         **time.Time
}
