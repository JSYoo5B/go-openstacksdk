package image

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// WithCreateTaskNullInput makes CreateTask send the top-level "input": null
// that Python's create_task(type=..., input=None) sends. It clears an earlier
// input object, and a later WithCreateTaskInput or WithCreateTaskOpts
// replaces it.
func WithCreateTaskNullInput() CreateTaskOption {
	return request.Custom(func(config *CreateTaskOpts) error {
		config.Input, config.NullInput = nil, true
		return nil
	})
}
