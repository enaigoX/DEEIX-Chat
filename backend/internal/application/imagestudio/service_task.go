package imagestudio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	appbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/billing"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/channel"
	domainbilling "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/billing"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
)

// StreamTaskInput is the application input for one streamed task.
type StreamTaskInput struct {
	UserID            uint
	RequestID         string
	TaskType          string
	Prompt            string
	PlatformModelName string
	Options           map[string]interface{}
	FileIDs           []string
	MaskFileID        string
	OnEvent           func(eventType string, payload map[string]interface{}) error
}

// StreamTask creates, runs, stores and bills one Image Studio task.
func (s *Service) StreamTask(ctx context.Context, input StreamTaskInput) (*domainimagestudio.Task, error) {
	taskType := strings.TrimSpace(input.TaskType)
	if taskType == "" {
		taskType = domainimagestudio.TaskTypeGeneration
	}
	if err := validateImageTask(taskType, input.Prompt, input.FileIDs); err != nil {
		return nil, err
	}
	if s.repo == nil || s.routeResolver == nil || s.llmClient == nil || s.uploadSvc == nil {
		return nil, ErrRouteNotConfigured
	}

	ctx = context.WithoutCancel(ctx)
	now := time.Now()
	task := &domainimagestudio.Task{
		TaskID:             newImageStudioTaskID(),
		UserID:             input.UserID,
		TaskType:           taskType,
		Prompt:             strings.TrimSpace(input.Prompt),
		Status:             domainimagestudio.TaskStatusQueued,
		RequestedModelName: strings.TrimSpace(input.PlatformModelName),
		OptionsJSON:        defaultJSON(marshalJSON(input.Options)),
		BilledCurrency:     "USD",
		StartedAt:          &now,
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	emitEvent(input.OnEvent, "media_status", map[string]interface{}{
		"status":  "queued",
		"message": "image task queued",
		"task":    task,
	})

	if err := s.attachInputAssets(ctx, task, input.FileIDs, input.MaskFileID); err != nil {
		return s.failTask(ctx, task, err)
	}

	if err := s.ensureBillingAccess(ctx, input.UserID, input.PlatformModelName); err != nil {
		return s.failTask(ctx, task, err)
	}
	reservation, err := s.reserveUsage(ctx, input.UserID, input.PlatformModelName, task.TaskID)
	if err != nil {
		return s.failTask(ctx, task, err)
	}

	result, runErr := s.runUpstreamTask(ctx, task, input)
	if runErr != nil {
		_ = s.releaseUsage(ctx, reservation, "图片生成失败退回预扣")
		return s.failTask(ctx, task, runErr)
	}

	completed, err := s.completeTask(ctx, task, result, reservation)
	if err != nil {
		if shouldReleaseReservationAfterBillingError(err) {
			_ = s.releaseUsage(ctx, reservation, "计费失败退回预扣")
		}
		return s.failTask(ctx, task, err)
	}
	emitEvent(input.OnEvent, "completed", map[string]interface{}{"data": completed})
	return completed, nil
}

type upstreamTaskResult struct {
	Route            *channel.ResolvedRoute
	EffectiveOptions map[string]interface{}
	Output           *llm.GenerateOutput
	Uploaded         []domainimagestudio.Asset
	LatencyMS        int64
}

func (s *Service) runUpstreamTask(ctx context.Context, task *domainimagestudio.Task, input StreamTaskInput) (*upstreamTaskResult, error) {
	taskRoute, endpoint := taskRouteType(task.TaskType)
	platformModelName := strings.TrimSpace(input.PlatformModelName)
	if platformModelName == "" {
		return nil, ErrRouteNotConfigured
	}
	route, err := s.routeResolver.ResolveRoute(ctx, channel.ResolveRouteInput{
		PlatformModelName: platformModelName,
		TaskType:          taskRoute,
		UserID:            input.UserID,
		ConversationID:    0,
		RequestID:         strings.TrimSpace(input.RequestID),
	})
	if err != nil {
		return nil, ErrRouteNotConfigured
	}
	if task.TaskType == domainimagestudio.TaskTypeGeneration && !llm.IsImageGenerationAdapter(route.Protocol) {
		return nil, ErrRouteProtocolMismatch
	}
	if task.TaskType == domainimagestudio.TaskTypeEdit && !llm.IsImageEditAdapter(route.Protocol) {
		return nil, ErrRouteProtocolMismatch
	}
	startedAt := time.Now()
	status := domainimagestudio.TaskStatusRunning
	routeJSON := marshalJSON(routeSnapshot(route))
	startedAtPtr := &startedAt
	taskPatch := domainimagestudio.TaskPatch{
		Status:             &status,
		PlatformModelName:  &route.PlatformModelName,
		ProviderProtocol:   &route.Protocol,
		UpstreamID:         &route.UpstreamID,
		UpstreamModelID:    &route.UpstreamModelID,
		UpstreamName:       &route.UpstreamName,
		RoutedBindingCode:  &route.BindingCode,
		ModelVendor:        &route.ModelVendor,
		ModelIcon:          &route.ModelIcon,
		UpstreamModelName:  &route.UpstreamModel,
		RouteSnapshotJSON:  &routeJSON,
		StartedAt:          &startedAtPtr,
		RequestedModelName: &platformModelName,
	}
	if updated, updateErr := s.repo.UpdateTask(ctx, input.UserID, task.TaskID, taskPatch); updateErr == nil {
		*task = *updated
	}
	emitEvent(input.OnEvent, "media_status", map[string]interface{}{
		"status":  "running",
		"message": runningMessage(task.TaskType),
		"task":    task,
	})

	cfg := s.cfg.Snapshot()
	routeConfig := llm.RouteConfig{
		Protocol:            route.Protocol,
		BaseURL:             route.BaseURL,
		APIKey:              route.APIKey,
		HeadersJSON:         route.HeadersJSON,
		ConnectTimeoutMS:    route.ConnectTimeoutMS,
		ReadTimeoutMS:       route.ReadTimeoutMS,
		StreamIdleTimeoutMS: route.StreamIdleTimeoutMS,
		Endpoint:            endpoint,
		UpstreamModel:       route.UpstreamModel,
		AttributionReferer:  cfg.PublicWebBaseURL,
		AttributionTitle:    cfg.AppName,
	}
	filteredOptions := filterModelOptions(input.Options, route.Protocol, modelOptionPolicyConfig{
		Mode:                       cfg.ModelOptionPolicyMode,
		AllowedPathsJSON:           cfg.ModelOptionAllowedPaths,
		DeniedPathsJSON:            cfg.ModelOptionDeniedPaths,
		NativeToolAllowedTypesJSON: cfg.NativeToolAllowedTypes,
	})
	generateInput, err := s.buildGenerateInput(ctx, task, input, filteredOptions)
	if err != nil {
		return nil, err
	}

	var output *llm.GenerateOutput
	if llm.SupportsImageGenerationStream(routeConfig.Protocol, routeConfig.UpstreamModel) {
		output, err = s.llmClient.GenerateStream(ctx, routeConfig, generateInput, func(event llm.GenerateStreamEvent) error {
			if event.Usage != (llm.Usage{}) {
				emitEvent(input.OnEvent, "usage", usageSnapshot(event.Usage))
			}
			if event.GeneratedImage != nil && event.GeneratedImagePartial {
				return emitImageDelta(input.OnEvent, event)
			}
			return nil
		})
	} else {
		output, err = s.llmClient.Generate(ctx, routeConfig, generateInput)
	}
	if err != nil {
		s.routeResolver.MarkRouteFailure(ctx, route, err)
		return nil, wrapUpstreamError(err)
	}
	s.routeResolver.MarkRouteSuccess(ctx, route)
	if output == nil || len(output.GeneratedImages) == 0 {
		return nil, ErrUpstreamEmptyResponse
	}

	emitEvent(input.OnEvent, "media_status", map[string]interface{}{
		"status":  "saving_artifact",
		"message": "saving image",
		"task":    task,
	})
	uploadedAssets := make([]domainimagestudio.Asset, 0, len(output.GeneratedImages))
	capturedAt := time.Now()
	for i, image := range output.GeneratedImages {
		file, uploadErr := s.uploadGeneratedImage(ctx, input.UserID, route.PlatformModelName, capturedAt, i, len(output.GeneratedImages), image)
		if uploadErr != nil {
			return nil, uploadErr
		}
		uploadedAssets = append(uploadedAssets, fileAssetFromFile(*task, *file, domainimagestudio.AssetRoleOutput, i, map[string]interface{}{
			"revised_prompt": strings.TrimSpace(image.RevisedPrompt),
			"source":         "provider",
		}))
	}
	if err = s.repo.ReplaceTaskAssetsByRole(ctx, task.ID, domainimagestudio.AssetRoleOutput, uploadedAssets); err != nil {
		return nil, err
	}
	return &upstreamTaskResult{
		Route:            route,
		EffectiveOptions: filteredOptions,
		Output:           output,
		Uploaded:         uploadedAssets,
		LatencyMS:        time.Since(startedAt).Milliseconds(),
	}, nil
}

func (s *Service) buildGenerateInput(ctx context.Context, task *domainimagestudio.Task, input StreamTaskInput, options map[string]interface{}) (llm.GenerateInput, error) {
	generateInput := llm.GenerateInput{
		RequestID:      strings.TrimSpace(input.RequestID),
		ConversationID: 0,
		Messages: []llm.Message{{
			Role:    "user",
			Content: task.Prompt,
		}},
		Options: options,
	}
	if task.TaskType != domainimagestudio.TaskTypeEdit {
		return generateInput, nil
	}
	parts := make([]llm.ContentPart, 0, 1+len(input.FileIDs))
	parts = append(parts, llm.ContentPart{
		Kind: llm.ContentPartText,
		Text: task.Prompt,
	})
	inputParts, err := s.resolveEditImageParts(ctx, input.UserID, input.FileIDs)
	if err != nil {
		return generateInput, err
	}
	parts = append(parts, inputParts...)
	generateInput.Messages = []llm.Message{{
		Role:  "user",
		Parts: parts,
	}}
	if strings.TrimSpace(input.MaskFileID) != "" {
		maskPart, maskErr := s.readImageFilePart(ctx, input.UserID, input.MaskFileID)
		if maskErr != nil {
			return generateInput, maskErr
		}
		generateInput.ImageEditMask = &maskPart
	}
	return generateInput, nil
}

func (s *Service) attachInputAssets(ctx context.Context, task *domainimagestudio.Task, fileIDs []string, maskFileID string) error {
	assets := make([]domainimagestudio.Asset, 0, len(fileIDs)+1)
	if len(fileIDs) > 0 {
		files, err := s.getActiveFileObjectsByIDs(ctx, task.UserID, fileIDs)
		if err != nil {
			return mapUploadError(err)
		}
		if len(files) != len(fileIDs) {
			return ErrFileNotFound
		}
		fileByID := make(map[string]domainconversation.FileObject, len(files))
		for _, file := range files {
			if !isImageFile(file) {
				return ErrEditInputInvalid
			}
			fileByID[file.FileID] = file
		}
		for i, fileID := range fileIDs {
			file, ok := fileByID[strings.TrimSpace(fileID)]
			if !ok {
				return ErrFileNotFound
			}
			assets = append(assets, fileAssetFromFile(*task, file, domainimagestudio.AssetRoleInput, i, nil))
		}
	}
	if normalizedMask := strings.TrimSpace(maskFileID); normalizedMask != "" {
		file, err := s.getActiveFileObjectByID(ctx, task.UserID, normalizedMask)
		if err != nil {
			return mapUploadError(err)
		}
		if !isImageFile(*file) {
			return ErrEditInputInvalid
		}
		assets = append(assets, fileAssetFromFile(*task, *file, domainimagestudio.AssetRoleMask, 0, nil))
	}
	if len(assets) == 0 {
		return nil
	}
	return s.repo.AddTaskAssets(ctx, task.ID, assets)
}

func (s *Service) resolveEditImageParts(ctx context.Context, userID uint, fileIDs []string) ([]llm.ContentPart, error) {
	if len(fileIDs) == 0 {
		return nil, ErrEditInputRequired
	}
	if len(fileIDs) > maxEditImages {
		return nil, ErrEditTooManyInputs
	}
	parts := make([]llm.ContentPart, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		part, err := s.readImageFilePart(ctx, userID, fileID)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func (s *Service) readImageFilePart(ctx context.Context, userID uint, fileID string) (llm.ContentPart, error) {
	content, err := s.uploadSvc.OpenFileContent(ctx, userID, strings.TrimSpace(fileID))
	if err != nil {
		return llm.ContentPart{}, mapUploadError(err)
	}
	defer content.Reader.Close() //nolint:errcheck
	limit := s.cfg.Snapshot().MaxUploadFileBytes
	if limit <= 0 {
		limit = 20 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(content.Reader, limit+1))
	if err != nil {
		return llm.ContentPart{}, err
	}
	if int64(len(data)) > limit {
		return llm.ContentPart{}, ErrFileTooLarge
	}
	mimeType := strings.TrimSpace(content.ContentType)
	if mimeType == "" {
		mimeType = strings.TrimSpace(content.File.DetectedMIME)
	}
	data, mimeType, err = validateImageBytes(data, mimeType)
	if err != nil {
		return llm.ContentPart{}, ErrEditInputInvalid
	}
	return llm.ContentPart{
		Kind:     llm.ContentPartImage,
		MimeType: mimeType,
		Data:     data,
		FileName: strings.TrimSpace(content.File.FileName),
	}, nil
}

func (s *Service) completeTask(ctx context.Context, task *domainimagestudio.Task, result *upstreamTaskResult, reservation *domainbilling.UsageBalanceReservation) (*domainimagestudio.Task, error) {
	usage := result.Output.Usage
	usageJSON := marshalJSON(usageSnapshot(usage))
	completedAt := time.Now()
	completedAtPtr := &completedAt
	status := domainimagestudio.TaskStatusSuccess
	patch := domainimagestudio.TaskPatch{
		Status:            &status,
		UsageSnapshotJSON: &usageJSON,
		InputTokens:       &usage.InputTokens,
		OutputTokens:      &usage.OutputTokens,
		CacheReadTokens:   &usage.CacheReadTokens,
		CacheWriteTokens:  &usage.CacheWriteTokens,
		ReasoningTokens:   &usage.ReasoningTokens,
		LatencyMS:         &result.LatencyMS,
		CompletedAt:       &completedAtPtr,
	}
	usageLedger, err := s.recordUsage(ctx, task, result, reservation)
	if err != nil {
		return nil, err
	}
	if usageLedger != nil {
		patch.BillingSnapshotJSON = &usageLedger.PricingSnapshotJSON
		patch.BilledCurrency = &usageLedger.BilledCurrency
		patch.BilledNanousd = &usageLedger.BilledNanousd
	}
	return s.repo.UpdateTask(ctx, task.UserID, task.TaskID, patch)
}

func (s *Service) failTask(ctx context.Context, task *domainimagestudio.Task, cause error) (*domainimagestudio.Task, error) {
	if task == nil {
		return nil, cause
	}
	completedAt := time.Now()
	completedAtPtr := &completedAt
	status := domainimagestudio.TaskStatusError
	code := classifyError(cause)
	message := errorSummary(cause)
	_, _ = s.repo.UpdateTask(ctx, task.UserID, task.TaskID, domainimagestudio.TaskPatch{
		Status:       &status,
		ErrorCode:    &code,
		ErrorMessage: &message,
		CompletedAt:  &completedAtPtr,
	})
	return nil, cause
}

func (s *Service) ensureBillingAccess(ctx context.Context, userID uint, modelName string) error {
	if s.billingSvc == nil {
		return nil
	}
	return s.billingSvc.EnsureModelUsable(ctx, userID, strings.TrimSpace(modelName), time.Now())
}

func (s *Service) reserveUsage(ctx context.Context, userID uint, modelName string, taskID string) (*domainbilling.UsageBalanceReservation, error) {
	if s.billingSvc == nil {
		return nil, nil
	}
	return s.billingSvc.ReserveUsageBalance(ctx, userID, strings.TrimSpace(modelName), strings.TrimSpace(taskID))
}

func (s *Service) releaseUsage(ctx context.Context, reservation *domainbilling.UsageBalanceReservation, description string) error {
	if s.billingSvc == nil || reservation == nil {
		return nil
	}
	return s.billingSvc.ReleaseUsageBalanceReservation(ctx, reservation, description)
}

func (s *Service) recordUsage(ctx context.Context, task *domainimagestudio.Task, result *upstreamTaskResult, reservation *domainbilling.UsageBalanceReservation) (*domainbilling.UsageLedger, error) {
	if s.billingSvc == nil || result == nil || result.Output == nil || result.Route == nil {
		return nil, nil
	}
	usage := result.Output.Usage
	usageLedger, err := s.billingSvc.BuildUsageLedger(ctx, appbilling.UsagePricingInput{
		UserID:             task.UserID,
		ConversationID:     0,
		PlatformModelName:  strings.TrimSpace(result.Route.PlatformModelName),
		RoutedBindingCode:  strings.TrimSpace(result.Route.BindingCode),
		ProviderProtocol:   strings.TrimSpace(result.Route.Protocol),
		UpstreamName:       strings.TrimSpace(result.Route.UpstreamName),
		UpstreamModelName:  strings.TrimSpace(result.Route.UpstreamModel),
		RequestSpeed:       stringOption(result.EffectiveOptions, "speed"),
		UsageSpeed:         strings.TrimSpace(usage.Speed),
		RequestServiceTier: stringOption(result.EffectiveOptions, "service_tier"),
		UsageServiceTier:   strings.TrimSpace(usage.ServiceTier),
		InputTokens:        usage.InputTokens,
		CacheReadTokens:    usage.CacheReadTokens,
		CacheWriteTokens:   usage.CacheWriteTokens,
		CacheWrite5mTokens: usage.CacheWrite5mTokens,
		CacheWrite1hTokens: usage.CacheWrite1hTokens,
		OutputTokens:       usage.OutputTokens,
		ReasoningTokens:    usage.ReasoningTokens,
		CallCount:          1,
		LatencyMS:          result.LatencyMS,
	})
	if err != nil || usageLedger == nil {
		return usageLedger, err
	}
	if err = s.billingSvc.RecordUsageWithReservation(ctx, usageLedger, reservation); err != nil {
		return nil, err
	}
	return usageLedger, nil
}

func shouldReleaseReservationAfterBillingError(err error) bool {
	return err != nil
}

func emitEvent(onEvent func(string, map[string]interface{}) error, eventType string, payload map[string]interface{}) {
	if onEvent == nil {
		return
	}
	_ = onEvent(eventType, payload)
}

func emitImageDelta(onEvent func(string, map[string]interface{}) error, event llm.GenerateStreamEvent) error {
	if onEvent == nil || event.GeneratedImage == nil {
		return nil
	}
	image := event.GeneratedImage
	if strings.TrimSpace(image.B64JSON) == "" {
		return nil
	}
	return onEvent("media_image_delta", map[string]interface{}{
		"index":          event.GeneratedImageIndex,
		"b64_json":       image.B64JSON,
		"mime_type":      strings.TrimSpace(image.MIMEType),
		"revised_prompt": strings.TrimSpace(image.RevisedPrompt),
	})
}

func runningMessage(taskType string) string {
	if taskType == domainimagestudio.TaskTypeEdit {
		return "editing image"
	}
	return "generating image"
}

func isImageFile(file domainconversation.FileObject) bool {
	mimeType := strings.ToLower(strings.TrimSpace(file.DetectedMIME))
	if mimeType == "" {
		mimeType = strings.ToLower(strings.TrimSpace(file.MimeType))
	}
	return strings.HasPrefix(mimeType, "image/")
}

func mapUploadError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrFileNotFound):
		return ErrFileNotFound
	case errors.Is(err, ErrInvalidFileReference):
		return ErrInvalidFileReference
	case strings.Contains(strings.ToLower(err.Error()), "file not found"):
		return ErrFileNotFound
	default:
		return err
	}
}

func stringOption(options map[string]interface{}, key string) string {
	if len(options) == 0 {
		return ""
	}
	raw, ok := options[key]
	if !ok || raw == nil {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
