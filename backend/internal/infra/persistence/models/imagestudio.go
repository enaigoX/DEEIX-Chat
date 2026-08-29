package model

import "time"

// ImageStudioTask stores cloud history for the standalone Image Studio module.
type ImageStudioTask struct {
	BaseModel
	TaskID              string     `gorm:"size:64;not null;default:'';uniqueIndex:idx_image_studio_tasks_task_id;comment:公开任务ID"`
	UserID              uint       `gorm:"not null;default:0;index:idx_image_studio_tasks_user_id;index:idx_image_studio_tasks_user_status,priority:1;comment:用户ID"`
	TaskType            string     `gorm:"size:32;not null;default:'';index:idx_image_studio_tasks_task_type;comment:任务类型"`
	Prompt              string     `gorm:"type:text;not null;default:'';comment:提示词"`
	Status              string     `gorm:"size:32;not null;default:'queued';index:idx_image_studio_tasks_status;index:idx_image_studio_tasks_user_status,priority:2;comment:任务状态"`
	RequestedModelName  string     `gorm:"size:128;not null;default:'';index:idx_image_studio_tasks_requested_model_name;comment:请求平台模型名"`
	PlatformModelName   string     `gorm:"size:128;not null;default:'';index:idx_image_studio_tasks_platform_model_name;comment:实际平台模型名"`
	ProviderProtocol    string     `gorm:"size:64;not null;default:'';index:idx_image_studio_tasks_provider_protocol;comment:协议适配器快照"`
	UpstreamID          uint       `gorm:"not null;default:0;index:idx_image_studio_tasks_upstream_id;comment:上游ID"`
	UpstreamModelID     uint       `gorm:"not null;default:0;index:idx_image_studio_tasks_upstream_model_id;comment:上游模型ID"`
	UpstreamName        string     `gorm:"size:128;not null;default:'';comment:上游名称快照"`
	RoutedBindingCode   string     `gorm:"size:64;not null;default:'';index:idx_image_studio_tasks_routed_binding_code;comment:路由绑定编码"`
	ModelVendor         string     `gorm:"size:64;not null;default:'';comment:模型厂商快照"`
	ModelIcon           string     `gorm:"size:64;not null;default:'';comment:模型图标快照"`
	UpstreamModelName   string     `gorm:"size:256;not null;default:'';comment:上游真实模型名"`
	OptionsJSON         string     `gorm:"type:text;not null;default:'{}';comment:请求参数快照JSON"`
	RouteSnapshotJSON   string     `gorm:"type:text;not null;default:'{}';comment:路由快照JSON"`
	UsageSnapshotJSON   string     `gorm:"type:text;not null;default:'{}';comment:用量快照JSON"`
	BillingSnapshotJSON string     `gorm:"type:text;not null;default:'';comment:计费快照JSON"`
	InputTokens         int64      `gorm:"not null;default:0;comment:输入Token"`
	OutputTokens        int64      `gorm:"not null;default:0;comment:输出Token"`
	CacheReadTokens     int64      `gorm:"not null;default:0;comment:缓存读取Token"`
	CacheWriteTokens    int64      `gorm:"not null;default:0;comment:缓存写入Token"`
	ReasoningTokens     int64      `gorm:"not null;default:0;comment:推理Token"`
	LatencyMS           int64      `gorm:"not null;default:0;comment:任务耗时毫秒"`
	BilledCurrency      string     `gorm:"size:16;not null;default:'USD';comment:计费币种"`
	BilledNanousd       int64      `gorm:"not null;default:0;comment:账单金额(纳美元)"`
	IsFavorite          bool       `gorm:"not null;default:false;index:idx_image_studio_tasks_is_favorite;comment:是否收藏"`
	FavoritedAt         *time.Time `gorm:"index:idx_image_studio_tasks_favorited_at;comment:收藏时间"`
	ErrorCode           string     `gorm:"size:64;not null;default:'';comment:错误码"`
	ErrorMessage        string     `gorm:"size:255;not null;default:'';comment:错误摘要"`
	StartedAt           *time.Time `gorm:"index:idx_image_studio_tasks_started_at;comment:开始时间"`
	CompletedAt         *time.Time `gorm:"index:idx_image_studio_tasks_completed_at;comment:完成时间"`
}

// TableName specifies the database table.
func (ImageStudioTask) TableName() string {
	return "image_studio_tasks"
}

// ImageStudioTaskAsset links task history to file objects.
type ImageStudioTaskAsset struct {
	BaseModel
	TaskID            uint   `gorm:"not null;default:0;index:idx_image_studio_task_assets_task_id;comment:任务内部ID"`
	TaskPublicID      string `gorm:"size:64;not null;default:'';index:idx_image_studio_task_assets_task_public_id;comment:公开任务ID"`
	UserID            uint   `gorm:"not null;default:0;index:idx_image_studio_task_assets_user_id;comment:用户ID"`
	FileID            string `gorm:"size:64;not null;default:'';index:idx_image_studio_task_assets_file_id;comment:文件对象ID"`
	Role              string `gorm:"size:32;not null;default:'';index:idx_image_studio_task_assets_role;comment:资产角色"`
	SortOrder         int    `gorm:"not null;default:0;comment:展示顺序"`
	FileName          string `gorm:"size:255;not null;default:'';comment:文件名快照"`
	MimeType          string `gorm:"size:128;not null;default:'';comment:媒体类型快照"`
	FileSize          int64  `gorm:"not null;default:0;comment:文件大小快照"`
	SHA256            string `gorm:"size:64;not null;default:'';index:idx_image_studio_task_assets_sha256;comment:文件SHA256快照"`
	StoragePath       string `gorm:"size:512;not null;default:'';comment:存储路径快照"`
	ThumbnailFileID   string `gorm:"size:64;not null;default:'';index:idx_image_studio_task_assets_thumbnail_file_id;comment:缩略图文件对象ID"`
	ThumbnailFileName string `gorm:"size:255;not null;default:'';comment:缩略图文件名快照"`
	ThumbnailMimeType string `gorm:"size:128;not null;default:'';comment:缩略图媒体类型快照"`
	ThumbnailFileSize int64  `gorm:"not null;default:0;comment:缩略图大小快照"`
	ThumbnailWidth    int    `gorm:"not null;default:0;comment:缩略图宽度"`
	ThumbnailHeight   int    `gorm:"not null;default:0;comment:缩略图高度"`
	FileSnapshotJSON  string `gorm:"type:text;not null;default:'{}';comment:文件快照JSON"`
	MetadataJSON      string `gorm:"type:text;not null;default:'{}';comment:上游或UI元数据JSON"`
}

// TableName specifies the database table.
func (ImageStudioTaskAsset) TableName() string {
	return "image_studio_task_assets"
}
