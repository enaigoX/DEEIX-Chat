package imagestudio

import "errors"

var (
	// ErrTaskNotFound indicates the task does not exist or belongs to another user.
	ErrTaskNotFound = errors.New("image studio task not found")
	// ErrInvalidTask indicates an invalid task type or payload.
	ErrInvalidTask = errors.New("invalid image studio task")
	// ErrPromptRequired indicates an empty image prompt.
	ErrPromptRequired = errors.New("image prompt is required")
	// ErrGenerationRejectsInputs indicates a generation task received edit inputs.
	ErrGenerationRejectsInputs = errors.New("image generation does not accept input images")
	// ErrEditInputRequired indicates an edit task lacks input images.
	ErrEditInputRequired = errors.New("image edit requires at least one input image")
	// ErrEditTooManyInputs indicates too many edit input images.
	ErrEditTooManyInputs = errors.New("too many image edit input images")
	// ErrEditInputInvalid indicates a referenced input image is invalid.
	ErrEditInputInvalid = errors.New("image edit input image is invalid")
	// ErrRouteNotConfigured indicates the model route cannot be resolved.
	ErrRouteNotConfigured = errors.New("model route not configured")
	// ErrRouteProtocolMismatch indicates the resolved route does not support this image task.
	ErrRouteProtocolMismatch = errors.New("media route protocol does not match task")
	// ErrUpstreamRequestFailed wraps an upstream provider error.
	ErrUpstreamRequestFailed = errors.New("upstream request failed")
	// ErrUpstreamEmptyResponse indicates the provider returned no final images.
	ErrUpstreamEmptyResponse = errors.New("upstream returned empty response")
	// ErrInvalidFileReference indicates an invalid file id or stream.
	ErrInvalidFileReference = errors.New("invalid file reference")
	// ErrInvalidFileName indicates an invalid filename.
	ErrInvalidFileName = errors.New("invalid file name")
	// ErrFileNotFound indicates a file does not exist or belongs to another user.
	ErrFileNotFound = errors.New("file not found")
	// ErrStorageQuotaExceeded indicates user storage quota is exceeded.
	ErrStorageQuotaExceeded = errors.New("storage quota exceeded")
	// ErrFileTooLarge indicates a file exceeds configured limits.
	ErrFileTooLarge = errors.New("file too large")
	// ErrMIMEBlocked indicates the MIME type is not allowed.
	ErrMIMEBlocked = errors.New("mime blocked")
	// ErrDangerousMIMEType indicates a dangerous MIME type.
	ErrDangerousMIMEType = errors.New("dangerous file type not allowed")
	// ErrEmbeddingUnavailable is kept for upload service compatibility.
	ErrEmbeddingUnavailable = errors.New("embedding unavailable")
)
