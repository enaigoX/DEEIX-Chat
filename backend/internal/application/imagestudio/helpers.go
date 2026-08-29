package imagestudio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/channel"
	appupload "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/application/upload"
	domainconversation "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/conversation"
	domainimagestudio "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/imagestudio"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/pkg/conv"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/ports/llm"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func isRepoNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}

func normalizeTaskID(raw string) string {
	return conv.NormalizePublicID(raw)
}

func newImageStudioTaskID() string {
	value := normalizeTaskID(uuid.NewString())
	if len(value) > 28 {
		value = value[:28]
	}
	return "img_" + value
}

func marshalJSON(value interface{}) string {
	raw, err := json.Marshal(value)
	if err != nil || len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func defaultJSON(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "{}"
	}
	return normalized
}

func taskRouteType(taskType string) (string, string) {
	if taskType == domainimagestudio.TaskTypeEdit {
		return channel.TaskTypeImageEdit, llm.EndpointImageEdits
	}
	return channel.TaskTypeImageGeneration, llm.EndpointImageGenerations
}

func validateImageTask(taskType string, prompt string, fileIDs []string) error {
	if taskType != domainimagestudio.TaskTypeGeneration && taskType != domainimagestudio.TaskTypeEdit {
		return ErrInvalidTask
	}
	if strings.TrimSpace(prompt) == "" {
		return ErrPromptRequired
	}
	if taskType == domainimagestudio.TaskTypeGeneration && len(fileIDs) > 0 {
		return ErrGenerationRejectsInputs
	}
	if taskType == domainimagestudio.TaskTypeEdit && len(fileIDs) == 0 {
		return ErrEditInputRequired
	}
	if taskType == domainimagestudio.TaskTypeEdit && len(fileIDs) > maxEditImages {
		return ErrEditTooManyInputs
	}
	return nil
}

func fileAssetFromFile(task domainimagestudio.Task, file domainconversation.FileObject, role string, order int, metadata map[string]interface{}) domainimagestudio.Asset {
	mimeType := strings.TrimSpace(file.DetectedMIME)
	if mimeType == "" {
		mimeType = strings.TrimSpace(file.MimeType)
	}
	fileSnapshot := map[string]interface{}{
		"file_id":       file.FileID,
		"file_name":     file.FileName,
		"mime_type":     mimeType,
		"size_bytes":    file.SizeBytes,
		"sha256":        file.SHA256,
		"storage_path":  file.StoragePath,
		"file_category": file.FileCategory,
	}
	return domainimagestudio.Asset{
		TaskID:           task.ID,
		TaskPublicID:     task.TaskID,
		UserID:           task.UserID,
		FileID:           file.FileID,
		Role:             role,
		SortOrder:        order,
		FileName:         file.FileName,
		MimeType:         mimeType,
		FileSize:         file.SizeBytes,
		SHA256:           file.SHA256,
		StoragePath:      file.StoragePath,
		FileSnapshotJSON: marshalJSON(fileSnapshot),
		MetadataJSON:     marshalJSON(metadata),
	}
}

func routeSnapshot(route *channel.ResolvedRoute) map[string]interface{} {
	if route == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"upstream_id":            route.UpstreamID,
		"upstream_model_id":      route.UpstreamModelID,
		"upstream_name":          route.UpstreamName,
		"platform_model_name":    route.PlatformModelName,
		"routed_binding_code":    route.BindingCode,
		"provider_protocol":      route.Protocol,
		"model_vendor":           route.ModelVendor,
		"model_icon":             route.ModelIcon,
		"upstream_model_name":    route.UpstreamModel,
		"connect_timeout_ms":     route.ConnectTimeoutMS,
		"read_timeout_ms":        route.ReadTimeoutMS,
		"stream_idle_timeout_ms": route.StreamIdleTimeoutMS,
	}
}

func usageSnapshot(usage llm.Usage) map[string]interface{} {
	return map[string]interface{}{
		"input_tokens":          usage.InputTokens,
		"output_tokens":         usage.OutputTokens,
		"cache_read_tokens":     usage.CacheReadTokens,
		"cache_write_tokens":    usage.CacheWriteTokens,
		"cache_write_5m_tokens": usage.CacheWrite5mTokens,
		"cache_write_1h_tokens": usage.CacheWrite1hTokens,
		"reasoning_tokens":      usage.ReasoningTokens,
		"speed":                 strings.TrimSpace(usage.Speed),
		"service_tier":          strings.TrimSpace(usage.ServiceTier),
	}
}

func wrapUpstreamError(cause error) error {
	if cause == nil {
		return ErrUpstreamRequestFailed
	}
	return fmt.Errorf("%w: %w", ErrUpstreamRequestFailed, cause)
}

func classifyError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidFileReference):
		return "invalid_file_reference"
	case errors.Is(err, ErrFileNotFound):
		return "file_not_found"
	case errors.Is(err, ErrStorageQuotaExceeded):
		return "storage_quota_exceeded"
	case errors.Is(err, ErrFileTooLarge):
		return "file_too_large"
	case errors.Is(err, ErrRouteNotConfigured):
		return "model_route_not_configured"
	case errors.Is(err, ErrUpstreamEmptyResponse):
		return "upstream_empty_response"
	case errors.Is(err, ErrPromptRequired):
		return "media_image_prompt_required"
	case errors.Is(err, ErrGenerationRejectsInputs):
		return "media_image_generation_rejects_inputs"
	case errors.Is(err, ErrEditInputRequired):
		return "media_image_edit_input_required"
	case errors.Is(err, ErrEditTooManyInputs):
		return "media_image_edit_too_many_inputs"
	case errors.Is(err, ErrEditInputInvalid):
		return "media_image_edit_input_invalid"
	case errors.Is(err, ErrRouteProtocolMismatch):
		return "media_route_protocol_mismatch"
	case errors.Is(err, ErrUpstreamRequestFailed):
		return "upstream_request_failed"
	default:
		return "internal_error"
	}
}

func errorSummary(err error) string {
	if err == nil {
		return ""
	}
	var upstreamErr *llm.UpstreamError
	if errors.As(err, &upstreamErr) {
		value := strings.TrimSpace(upstreamErr.Message)
		if value != "" {
			return value
		}
	}
	value := strings.TrimSpace(err.Error())
	prefix := ErrUpstreamRequestFailed.Error() + ":"
	for strings.HasPrefix(value, prefix) {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	return truncateText(value, 255)
}

func truncateText(value string, maxChars int) string {
	runes := []rune(strings.TrimSpace(value))
	if maxChars <= 0 || len(runes) <= maxChars {
		return string(runes)
	}
	return string(runes[:maxChars])
}

func imageFileExtension(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func generatedImageFileName(modelName string, capturedAt time.Time, index int, total int, mimeType string) string {
	base := sanitizeFileBase(modelName)
	timestamp := fmt.Sprintf("%s-%03d", capturedAt.Format("20060102-150405"), capturedAt.Nanosecond()/int(time.Millisecond))
	if total > 1 {
		return fmt.Sprintf("%s-%s-%02d%s", base, timestamp, index+1, imageFileExtension(mimeType))
	}
	return fmt.Sprintf("%s-%s%s", base, timestamp, imageFileExtension(mimeType))
}

func sanitizeFileBase(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "image"
	}
	var builder strings.Builder
	builder.Grow(len(normalized))
	lastDash := false
	for _, item := range normalized {
		allowed := (item >= 'a' && item <= 'z') ||
			(item >= 'A' && item <= 'Z') ||
			(item >= '0' && item <= '9') ||
			item == '.' ||
			item == '_' ||
			item == '-'
		if allowed {
			builder.WriteRune(item)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), ".-_")
	if result == "" {
		return "image"
	}
	if len(result) > 80 {
		result = strings.Trim(result[:80], ".-_")
	}
	if result == "" {
		return "image"
	}
	return result
}

func validateImageBytes(data []byte, declaredMIME string) ([]byte, string, error) {
	detected := detectImageMIME(data)
	if detected == "" {
		return nil, strings.TrimSpace(declaredMIME), ErrEditInputInvalid
	}
	return data, detected, nil
}

func detectImageMIME(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png"
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg"
	}
	if len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	if len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))) {
		return "image/gif"
	}
	return ""
}

func stripBase64DataURLPrefix(value string) string {
	normalized := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(normalized), "data:") {
		return normalized
	}
	if index := strings.Index(normalized, ","); index >= 0 {
		return strings.TrimSpace(normalized[index+1:])
	}
	return normalized
}

func (s *Service) readGeneratedImage(ctx context.Context, image llm.GeneratedImage, trustedProviderEndpoint string) ([]byte, string, error) {
	mimeType := strings.TrimSpace(image.MIMEType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	if b64 := strings.TrimSpace(image.B64JSON); b64 != "" {
		data, err := base64.StdEncoding.DecodeString(stripBase64DataURLPrefix(b64))
		if err != nil {
			return nil, mimeType, err
		}
		return validateImageBytes(data, mimeType)
	}
	url := strings.TrimSpace(image.URL)
	if url == "" {
		return nil, mimeType, ErrUpstreamEmptyResponse
	}
	if s.mediaDownloader == nil {
		return nil, mimeType, fmt.Errorf("generated media downloader is not configured")
	}
	cfg := s.cfg.Snapshot()
	limit := cfg.MaxUploadFileBytes
	if limit <= 0 {
		limit = 20 * 1024 * 1024
	}
	data, downloadedMIME, err := s.mediaDownloader.DownloadImage(ctx, url, trustedProviderEndpoint, limit)
	if err != nil {
		var tooLarge mediaArtifactResponseTooLarge
		if errors.As(err, &tooLarge) {
			return nil, mimeType, ErrFileTooLarge
		}
		return nil, mimeType, err
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(downloadedMIME)), "image/") {
		mimeType = strings.TrimSpace(downloadedMIME)
	}
	return validateImageBytes(data, mimeType)
}

type generatedImageUpload struct {
	Original        domainconversation.FileObject
	Thumbnail       domainconversation.FileObject
	ThumbnailWidth  int
	ThumbnailHeight int
}

func (s *Service) uploadGeneratedImage(ctx context.Context, userID uint, modelName string, trustedProviderEndpoint string, capturedAt time.Time, index int, total int, generated llm.GeneratedImage) (*generatedImageUpload, error) {
	data, mimeType, err := s.readGeneratedImage(ctx, generated, trustedProviderEndpoint)
	if err != nil {
		return nil, err
	}
	thumbnailData, thumbnailWidth, thumbnailHeight, err := makeImageThumbnail(data, 480)
	if err != nil {
		return nil, err
	}
	fileName := generatedImageFileName(modelName, capturedAt, index, total, mimeType)
	result, err := s.uploadSvc.UploadFile(ctx, appupload.UploadFileInput{
		UserID:       userID,
		Purpose:      "generated_image",
		FileName:     fileName,
		MimeType:     mimeType,
		DeclaredSize: int64(len(data)),
		Reader:       bytes.NewReader(data),
	})
	if err != nil {
		return nil, err
	}
	thumbnailName := strings.TrimSuffix(fileName, filepath.Ext(fileName)) + "-thumb.jpg"
	thumbnailResult, err := s.uploadSvc.UploadFile(ctx, appupload.UploadFileInput{
		UserID:       userID,
		Purpose:      "image_studio_thumbnail",
		FileName:     thumbnailName,
		MimeType:     "image/jpeg",
		DeclaredSize: int64(len(thumbnailData)),
		Reader:       bytes.NewReader(thumbnailData),
	})
	if err != nil {
		return nil, err
	}
	return &generatedImageUpload{
		Original:        result.File,
		Thumbnail:       thumbnailResult.File,
		ThumbnailWidth:  thumbnailWidth,
		ThumbnailHeight: thumbnailHeight,
	}, nil
}

func makeImageThumbnail(data []byte, maxDimension int) ([]byte, int, int, error) {
	if maxDimension <= 0 {
		return nil, 0, 0, fmt.Errorf("thumbnail dimension must be positive")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}
	width, height := thumbnailDimensions(source.Bounds().Dx(), source.Bounds().Dy(), maxDimension)
	if width <= 0 || height <= 0 {
		return nil, 0, 0, fmt.Errorf("invalid image dimensions")
	}
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	stddraw.Draw(target, target.Bounds(), &image.Uniform{C: color.White}, image.Point{}, stddraw.Src)
	draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, target, &jpeg.Options{Quality: 82}); err != nil {
		return nil, 0, 0, err
	}
	return encoded.Bytes(), width, height, nil
}

func thumbnailDimensions(width int, height int, maxDimension int) (int, int) {
	if width <= 0 || height <= 0 || maxDimension <= 0 {
		return 0, 0
	}
	if width <= maxDimension && height <= maxDimension {
		return width, height
	}
	if width >= height {
		return maxDimension, max(1, height*maxDimension/width)
	}
	return max(1, width*maxDimension/height), maxDimension
}
