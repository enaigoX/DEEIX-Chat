package imagestudio

import (
	"reflect"
	"testing"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
)

func TestFilterModelOptionsAllowsOpenAIImageParams(t *testing.T) {
	filtered := filterModelOptions(map[string]interface{}{
		"size":               "1024x1024",
		"quality":            "high",
		"output_format":      "webp",
		"output_compression": 80,
		"partial_images":     2,
		"prompt":             "override",
		"headers":            map[string]interface{}{"Authorization": "Bearer bad"},
	}, "openai_image_generations", modelOptionPolicyConfig{
		Mode:             modelOptionPolicyAllowlist,
		AllowedPathsJSON: config.DefaultModelOptionAllowedPathsJSON(),
	})

	want := map[string]interface{}{
		"size":               "1024x1024",
		"quality":            "high",
		"output_format":      "webp",
		"output_compression": 80,
		"partial_images":     2,
	}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("unexpected filtered options:\nwant %#v\ngot  %#v", want, filtered)
	}
}

func TestFilterModelOptionsAllowsGeminiImagePaths(t *testing.T) {
	filtered := filterModelOptions(map[string]interface{}{
		"generationConfig": map[string]interface{}{
			"responseModalities": []interface{}{"IMAGE"},
			"imageConfig": map[string]interface{}{
				"aspectRatio": "1:1",
				"imageSize":   "4K",
				"unsafe":      "drop",
			},
		},
		"quality": "high",
		"prompt":  "override",
	}, "google_image_generation", modelOptionPolicyConfig{
		Mode:             modelOptionPolicyAllowlist,
		AllowedPathsJSON: config.DefaultModelOptionAllowedPathsJSON(),
	})

	want := map[string]interface{}{
		"generationConfig": map[string]interface{}{
			"responseModalities": []interface{}{"IMAGE"},
			"imageConfig": map[string]interface{}{
				"aspectRatio": "1:1",
				"imageSize":   "4K",
			},
		},
	}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("unexpected filtered options:\nwant %#v\ngot  %#v", want, filtered)
	}
}

func TestFilterModelOptionsAllowsXAIImageParams(t *testing.T) {
	filtered := filterModelOptions(map[string]interface{}{
		"aspect_ratio":    "16:9",
		"resolution":      "2k",
		"n":               2,
		"response_format": "b64_json",
		"quality":         "high",
		"stream":          true,
	}, "xai_image", modelOptionPolicyConfig{
		Mode:             modelOptionPolicyAllowlist,
		AllowedPathsJSON: config.DefaultModelOptionAllowedPathsJSON(),
	})

	want := map[string]interface{}{
		"aspect_ratio":    "16:9",
		"resolution":      "2k",
		"n":               2,
		"response_format": "b64_json",
	}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("unexpected filtered options:\nwant %#v\ngot  %#v", want, filtered)
	}
}

func TestFilterModelOptionsDenylistStillRemovesHardDeniedPaths(t *testing.T) {
	filtered := filterModelOptions(map[string]interface{}{
		"size":    "1024x1024",
		"quality": "high",
		"model":   "other",
		"headers": map[string]interface{}{"Authorization": "Bearer bad"},
	}, "openai_image_generations", modelOptionPolicyConfig{
		Mode: modelOptionPolicyDenylist,
	})

	want := map[string]interface{}{
		"size":    "1024x1024",
		"quality": "high",
	}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("unexpected filtered options:\nwant %#v\ngot  %#v", want, filtered)
	}
}

func TestFilterModelOptionsDropsInvalidPartialImages(t *testing.T) {
	filtered := filterModelOptions(map[string]interface{}{
		"partial_images": 8,
		"size":           "1024x1024",
	}, "openai_image_generations", modelOptionPolicyConfig{
		Mode:             modelOptionPolicyAllowlist,
		AllowedPathsJSON: config.DefaultModelOptionAllowedPathsJSON(),
	})

	want := map[string]interface{}{"size": "1024x1024"}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("unexpected filtered options:\nwant %#v\ngot  %#v", want, filtered)
	}
}
