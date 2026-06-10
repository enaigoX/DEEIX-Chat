package imagestudio

import (
	"encoding/json"
	"strings"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/llm"
)

const (
	modelOptionPolicyAllowlist = "allowlist"
	modelOptionPolicyDenylist  = "denylist"
	modelOptionPolicyDisabled  = "disabled"
)

var hardDeniedModelOptionPaths = [][]string{
	{"model"},
	{"messages"},
	{"input"},
	{"instructions"},
	{"prompt"},
	{"system"},
	{"systemInstruction"},
	{"headers"},
	{"api_key"},
	{"apiKey"},
	{"base_url"},
	{"baseURL"},
	{"stream"},
	{"previous_response_id"},
}

type modelOptionPolicyConfig struct {
	Mode                       string
	AllowedPathsJSON           string
	DeniedPathsJSON            string
	NativeToolAllowedTypesJSON string
}

func filterModelOptions(options map[string]interface{}, protocol string, cfg modelOptionPolicyConfig) map[string]interface{} {
	if len(options) == 0 {
		return nil
	}
	mode := strings.TrimSpace(cfg.Mode)
	if mode == "" {
		mode = modelOptionPolicyAllowlist
	}
	if mode == modelOptionPolicyDisabled {
		return nil
	}

	protocolKey := modelOptionPolicyProtocolKey(protocol)
	denied := append([][]string{}, hardDeniedModelOptionPaths...)

	var filtered map[string]interface{}
	switch mode {
	case modelOptionPolicyDenylist:
		denied = append(denied, modelOptionPathsForProtocol(cfg.DeniedPathsJSON, protocolKey)...)
		filtered = cloneModelOptionMap(options)
	default:
		filtered = make(map[string]interface{})
		for _, path := range allowedImageOptionPathsForProtocol(cfg.AllowedPathsJSON, protocolKey) {
			copyModelOptionPath(filtered, options, path)
		}
	}
	for _, path := range denied {
		deleteModelOptionPath(filtered, path)
	}
	sanitizeImageStudioModelOptionValues(filtered, protocolKey)
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func allowedImageOptionPathsForProtocol(raw string, protocol string) [][]string {
	paths := modelOptionPathsForProtocol(raw, protocol)
	if len(paths) > 0 {
		return paths
	}
	return defaultAllowedImageOptionPaths(protocol)
}

func defaultAllowedImageOptionPaths(protocol string) [][]string {
	values := []string{}
	switch protocol {
	case "openai_image_edits":
		values = []string{"background", "input_fidelity", "n", "output_compression", "output_format", "partial_images", "quality", "response_format", "size", "user"}
	case "google_image_generation":
		values = []string{
			"aspect_ratio",
			"aspectRatio",
			"image_size",
			"imageSize",
			"imageConfig.aspectRatio",
			"imageConfig.imageSize",
			"responseFormat.image.aspectRatio",
			"responseFormat.image.imageSize",
			"generationConfig.imageConfig.aspectRatio",
			"generationConfig.imageConfig.imageSize",
			"generationConfig.responseFormat.image.aspectRatio",
			"generationConfig.responseFormat.image.imageSize",
		}
	case "xai_image", "xai_image_edits":
		values = []string{"aspect_ratio", "n", "resolution", "response_format"}
	case "openai_image_generations":
		fallthrough
	default:
		values = []string{"background", "moderation", "n", "output_compression", "output_format", "partial_images", "quality", "response_format", "size", "style", "user"}
	}
	paths := make([][]string, 0, len(values))
	for _, value := range values {
		if path := splitModelOptionPath(value); len(path) > 0 {
			paths = append(paths, path)
		}
	}
	return paths
}

func sanitizeImageStudioModelOptionValues(options map[string]interface{}, protocolKey string) {
	if len(options) == 0 {
		return
	}
	switch protocolKey {
	case "openai_image_generations", "openai_image_edits":
		value, ok := modelParamIntFromOption(options["partial_images"])
		if !ok {
			delete(options, "partial_images")
			return
		}
		if value < 0 || value > 3 {
			delete(options, "partial_images")
		}
	}
}

func modelParamIntFromOption(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func modelOptionPolicyProtocolKey(protocol string) string {
	switch llm.NormalizeAdapter(protocol) {
	case llm.AdapterGoogleImageGeneration:
		return "google_image_generation"
	case llm.AdapterOpenAIImageGenerations:
		return "openai_image_generations"
	case llm.AdapterOpenAIImageEdits:
		return "openai_image_edits"
	case llm.AdapterXAIImage:
		return "xai_image"
	case llm.AdapterXAIImageEdits:
		return "xai_image_edits"
	default:
		return "openai_image_generations"
	}
}

func modelOptionPathsForProtocol(raw string, protocol string) [][]string {
	var config map[string][]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &config); err != nil {
		return nil
	}
	paths := make([][]string, 0, len(config["default"])+len(config[protocol]))
	for _, value := range append(config["default"], config[protocol]...) {
		if path := splitModelOptionPath(value); len(path) > 0 {
			paths = append(paths, path)
		}
	}
	return paths
}

func splitModelOptionPath(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return nil
	}
	parts := strings.Split(value, ".")
	for _, part := range parts {
		if part == "" {
			return nil
		}
	}
	return parts
}

func copyModelOptionPath(dst map[string]interface{}, src map[string]interface{}, path []string) {
	value, ok := readModelOptionPath(src, path)
	if !ok {
		return
	}
	writeModelOptionPath(dst, path, cloneModelOptionValue(value))
}

func readModelOptionPath(src map[string]interface{}, path []string) (interface{}, bool) {
	if len(path) == 0 {
		return nil, false
	}
	current := src
	for index, segment := range path {
		value, ok := current[segment]
		if !ok {
			return nil, false
		}
		if index == len(path)-1 {
			return value, true
		}
		next, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}
	return nil, false
}

func writeModelOptionPath(dst map[string]interface{}, path []string, value interface{}) {
	current := dst
	for index, segment := range path {
		if index == len(path)-1 {
			current[segment] = value
			return
		}
		next, ok := current[segment].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[segment] = next
		}
		current = next
	}
}

func deleteModelOptionPath(dst map[string]interface{}, path []string) {
	if len(path) == 0 || len(dst) == 0 {
		return
	}
	current := dst
	for index, segment := range path {
		if index == len(path)-1 {
			delete(current, segment)
			return
		}
		next, ok := current[segment].(map[string]interface{})
		if !ok {
			return
		}
		current = next
	}
}

func cloneModelOptionMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for key, value := range src {
		dst[key] = cloneModelOptionValue(value)
	}
	return dst
}

func cloneModelOptionValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return cloneModelOptionMap(typed)
	case []interface{}:
		items := make([]interface{}, len(typed))
		for index, item := range typed {
			items[index] = cloneModelOptionValue(item)
		}
		return items
	default:
		return typed
	}
}
