package imagestudio

import "strings"

const (
	modelOptionPolicyAllowlist = "allowlist"
	modelOptionPolicyDenylist  = "denylist"
	modelOptionPolicyDisabled  = "disabled"
)

var hardDeniedModelOptionKeys = map[string]struct{}{
	"model":                {},
	"messages":             {},
	"input":                {},
	"instructions":         {},
	"prompt":               {},
	"system":               {},
	"systemInstruction":    {},
	"headers":              {},
	"api_key":              {},
	"apiKey":               {},
	"base_url":             {},
	"baseURL":              {},
	"stream":               {},
	"previous_response_id": {},
}

type modelOptionPolicyConfig struct {
	Mode                       string
	AllowedPathsJSON           string
	DeniedPathsJSON            string
	NativeToolAllowedTypesJSON string
}

func filterModelOptions(options map[string]interface{}, _ string, cfg modelOptionPolicyConfig) map[string]interface{} {
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

	filtered := make(map[string]interface{}, len(options))
	switch mode {
	case modelOptionPolicyDenylist:
		for key, value := range options {
			if _, denied := hardDeniedModelOptionKeys[key]; denied {
				continue
			}
			filtered[key] = value
		}
	default:
		for _, key := range defaultAllowedImageOptionKeys() {
			if value, ok := options[key]; ok {
				filtered[key] = value
			}
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func defaultAllowedImageOptionKeys() []string {
	return []string{
		"n",
		"size",
		"quality",
		"style",
		"output_format",
		"output_compression",
		"background",
		"moderation",
		"partial_images",
		"response_format",
		"user",
	}
}
