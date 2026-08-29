package promptpreset

import "errors"

var (
	// ErrPromptPresetNotFound 表示预制提示词不存在或当前用户无权访问。
	ErrPromptPresetNotFound = errors.New("prompt preset not found")
	// ErrInvalidPromptPreset 表示预制提示词参数不合法。
	ErrInvalidPromptPreset = errors.New("invalid prompt preset")
	// ErrPromptPresetConflict 表示触发词在当前作用域内已存在。
	ErrPromptPresetConflict = errors.New("prompt preset trigger already exists")
)
