package imagestudio

import (
	"context"
	"errors"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/ports/llm"
)

type imageStudioMediaDownloaderStub struct {
	downloadImage func(context.Context, string, string, int64) ([]byte, string, error)
}

func (s imageStudioMediaDownloaderStub) DownloadImage(ctx context.Context, sourceURL string, trustedProviderEndpoint string, maxBytes int64) ([]byte, string, error) {
	return s.downloadImage(ctx, sourceURL, trustedProviderEndpoint, maxBytes)
}

type imageStudioMediaTooLargeError struct{}

func (imageStudioMediaTooLargeError) Error() string { return "media too large" }

func (imageStudioMediaTooLargeError) MediaArtifactResponseTooLarge() {}

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

func TestReadGeneratedImageDelegatesDownloadWithProviderEndpoint(t *testing.T) {
	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	service := &Service{
		cfg: config.NewRuntime(config.Config{MaxUploadFileBytes: 1024}),
		mediaDownloader: imageStudioMediaDownloaderStub{
			downloadImage: func(_ context.Context, sourceURL string, trustedProviderEndpoint string, maxBytes int64) ([]byte, string, error) {
				if sourceURL != "https://cdn.example.test/image" || trustedProviderEndpoint != "http://model.internal:8080/v1" || maxBytes != 1024 {
					t.Fatalf("unexpected download input: URL=%q trustedEndpoint=%q maxBytes=%d", sourceURL, trustedProviderEndpoint, maxBytes)
				}
				return pngHeader, "image/png", nil
			},
		},
	}

	data, mimeType, err := service.readGeneratedImage(t.Context(), llm.GeneratedImage{
		URL:      "https://cdn.example.test/image",
		MIMEType: "application/octet-stream",
	}, "http://model.internal:8080/v1")
	if err != nil {
		t.Fatalf("read generated image: %v", err)
	}
	if string(data) != string(pngHeader) || mimeType != "image/png" {
		t.Fatalf("unexpected generated image: data=%q MIME=%q", data, mimeType)
	}
}

func TestReadGeneratedImageMapsDownloaderSizeLimit(t *testing.T) {
	service := &Service{
		cfg: config.NewRuntime(config.Config{MaxUploadFileBytes: 1024}),
		mediaDownloader: imageStudioMediaDownloaderStub{
			downloadImage: func(context.Context, string, string, int64) ([]byte, string, error) {
				return nil, "", imageStudioMediaTooLargeError{}
			},
		},
	}

	_, _, err := service.readGeneratedImage(t.Context(), llm.GeneratedImage{URL: "https://cdn.example.test/image"}, "https://model.example.test/v1")
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("read generated image error = %v, want ErrFileTooLarge", err)
	}
}
