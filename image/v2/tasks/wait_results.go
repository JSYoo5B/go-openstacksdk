package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// TaskWaitResponse retains an actual task HTTP response. Task is nil when an
// accepted creation response could not be decoded. Body and Header are owned.
type TaskWaitResponse struct {
	Task       *Task
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// TaskWaitResult retains the latest successfully decoded task, its actual
// response, and the fixed request IDs. Created is the latest accepted POST201
// evidence, including accepted read/decode failures. Recreated counts only
// successfully decoded creations with a safe canonical ID.
type TaskWaitResult struct {
	Task       *Task
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
	OriginalID string
	CurrentID  string
	Recreated  int
	Created    *TaskWaitResponse
}

func taskWaitResponse(response *rest.Response, task *Task) *TaskWaitResponse {
	if response == nil {
		return nil
	}
	return &TaskWaitResponse{Task: copyTaskWaitTask(task), Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

func copyTaskWaitTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	owned := *task
	if task.Input != nil {
		owned.Input = copyTaskWaitJSON(task.Input).(map[string]any)
	}
	if task.Result != nil {
		owned.Result = copyTaskWaitJSON(task.Result).(map[string]any)
	}
	return &owned
}

func copyTaskWaitJSON(value any) any {
	switch value := value.(type) {
	case map[string]any:
		owned := make(map[string]any, len(value))
		for key, nested := range value {
			owned[key] = copyTaskWaitJSON(nested)
		}
		return owned
	case []any:
		owned := make([]any, len(value))
		for index, nested := range value {
			owned[index] = copyTaskWaitJSON(nested)
		}
		return owned
	default:
		return value
	}
}

func (result *TaskWaitResult) observed(response *rest.Response, task *Task) {
	result.Task = task
	result.Body = append(json.RawMessage(nil), response.Body...)
	result.Header = response.Header.Clone()
	result.StatusCode = response.StatusCode
}

func decodeTaskWait(response *rest.Response) (*Task, map[string]json.RawMessage, error) {
	if response != nil && !utf8.Valid(response.Body) {
		return nil, nil, response.Fail(fmt.Errorf("task response is not valid UTF-8"))
	}
	body, err := response.Object("")
	if err != nil {
		return nil, nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, nil, response.Fail(err)
	}
	// Known fields use exact canonical keys. Case aliases remain extensions in
	// Body and cannot override identity/status/type or mutation input.
	canonical := make(map[string]json.RawMessage, 12)
	for _, key := range []string{"id", "type", "status", "input", "result", "owner", "message", "expires_at", "created_at", "updated_at", "self", "schema"} {
		if raw, exists := fields[key]; exists {
			canonical[key] = raw
		}
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, nil, response.Fail(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var task Task
	if err := decoder.Decode(&task); err != nil {
		return nil, nil, response.Fail(err)
	}
	return &task, fields, nil
}

func taskWaitString(fields map[string]json.RawMessage, key string) (string, error) {
	raw, exists := fields[key]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", fmt.Errorf("task response requires %q string", key)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}
