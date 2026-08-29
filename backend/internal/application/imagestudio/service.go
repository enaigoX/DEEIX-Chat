package imagestudio

import (
	"context"
	"time"

	appbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/channel"
	appstorage "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/objectstorage"
	appupload "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/upload"
	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/ports/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"go.uber.org/zap"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	maxEditImages   = 16
)

type routeResolver interface {
	ResolveRoute(ctx context.Context, input channel.ResolveRouteInput) (*channel.ResolvedRoute, error)
	MarkRouteFailure(ctx context.Context, route *channel.ResolvedRoute, cause error)
	MarkRouteSuccess(ctx context.Context, route *channel.ResolvedRoute)
}

type llmGateway interface {
	Generate(ctx context.Context, route llm.RouteConfig, input llm.GenerateInput) (*llm.GenerateOutput, error)
	GenerateStream(ctx context.Context, route llm.RouteConfig, input llm.GenerateInput, onEvent func(llm.GenerateStreamEvent) error) (*llm.GenerateOutput, error)
}

type generatedMediaDownloader interface {
	DownloadImage(ctx context.Context, sourceURL string, trustedProviderEndpoint string, maxBytes int64) ([]byte, string, error)
}

type mediaArtifactResponseTooLarge interface {
	MediaArtifactResponseTooLarge()
}

// Service owns Image Studio task execution and cloud history.
type Service struct {
	cfg             *config.Runtime
	repo            repository.ImageStudioRepository
	uploadSvc       *appupload.Service
	routeResolver   routeResolver
	llmClient       llmGateway
	mediaDownloader generatedMediaDownloader
	billingSvc      *appbilling.Service
	storeProvider   appstorage.Provider
	logger          *zap.Logger
}

// NewServiceWithRuntime creates an Image Studio service.
func NewServiceWithRuntime(
	cfg *config.Runtime,
	repo repository.ImageStudioRepository,
	uploadSvc *appupload.Service,
	routeResolver routeResolver,
	llmClient llmGateway,
	mediaDownloader generatedMediaDownloader,
	billingSvc *appbilling.Service,
	logger *zap.Logger,
) *Service {
	return &Service{
		cfg:             cfg,
		repo:            repo,
		uploadSvc:       uploadSvc,
		routeResolver:   routeResolver,
		llmClient:       llmClient,
		mediaDownloader: mediaDownloader,
		billingSvc:      billingSvc,
		storeProvider:   appstorage.NewRuntimeProvider(cfg, nil),
		logger:          logger,
	}
}

// SetObjectStoreProvider shares the app object storage provider with uploads.
func (s *Service) SetObjectStoreProvider(provider appstorage.Provider) {
	if provider != nil {
		s.storeProvider = provider
		if s.uploadSvc != nil {
			s.uploadSvc.SetObjectStoreProvider(provider)
		}
	}
}

// ListTasks returns paginated task history.
func (s *Service) ListTasks(ctx context.Context, userID uint, page int, pageSize int, filter domainimagestudio.TaskListFilter) ([]domainimagestudio.Task, int64, error) {
	offset, limit := normalizePage(page, pageSize)
	return s.repo.ListTasks(ctx, userID, filter, offset, limit)
}

// GetTask returns one task with assets.
func (s *Service) GetTask(ctx context.Context, userID uint, taskID string) (*domainimagestudio.Task, error) {
	item, err := s.repo.GetTaskByPublicID(ctx, userID, taskID)
	if err != nil {
		if isRepoNotFound(err) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return item, nil
}

// SetFavorite toggles task favorite state.
func (s *Service) SetFavorite(ctx context.Context, userID uint, taskID string, favorite bool) (*domainimagestudio.Task, error) {
	item, err := s.repo.SetTaskFavorite(ctx, userID, taskID, favorite)
	if err != nil {
		if isRepoNotFound(err) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return item, nil
}

// DeleteTask hides task history without deleting file objects.
func (s *Service) DeleteTask(ctx context.Context, userID uint, taskID string) error {
	if err := s.repo.DeleteTask(ctx, userID, taskID); err != nil {
		if isRepoNotFound(err) {
			return ErrTaskNotFound
		}
		return err
	}
	return nil
}

func normalizePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return (page - 1) * pageSize, pageSize
}

func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
