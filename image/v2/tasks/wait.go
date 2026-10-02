package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const taskWaitImportError396 = "Image cannot be imported. Error code: '396'"

// WaitForTask waits for success, recreating only a selected failed task whose
// fetched message exactly identifies Glance import error396. It always starts
// with a fresh GET of an explicit ID, rather than a cached Task resource.
func (a *API) WaitForTask(ctx context.Context, ref resource.Ref, options ...TaskWaitOption) (*TaskWaitResult, error) {
	return a.WaitForTaskState(ctx, ref, "success", options...)
}

// WaitForTaskState uses one timeout budget for every GET, recreation and pause.
// The target takes precedence over failure detection. A newly created task is
// fetched after the pause even when its POST response already reports success.
func (a *API) WaitForTaskState(ctx context.Context, ref resource.Ref, target string, options ...TaskWaitOption) (*TaskWaitResult, error) {
	var budget context.Context
	wrap := func(err error) error {
		if err != nil && budget != nil && budget.Err() != nil && !errors.Is(err, budget.Err()) {
			err = errors.Join(err, budget.Err())
		}
		return request.Wrap("WaitForTask", "tasks", err)
	}
	var source *gophercloud.ServiceClient
	if a != nil {
		source = a.client
	}
	if err := validateTaskWaitSource(ctx, source); err != nil {
		return nil, wrap(err)
	}
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	if ref.IsName() {
		return nil, wrap(fmt.Errorf("%w: task waits require an explicit ID", resource.ErrUnsupported))
	}
	if err := taskWaitID(ref.String()); err != nil {
		return nil, wrap(err)
	}
	if strings.TrimSpace(target) == "" {
		return nil, wrap(taskWaitInvalid("target state must not be empty"))
	}
	policy, err := parseTaskWaitOpts(append([]TaskWaitOption(nil), options...))
	if err != nil {
		return nil, wrap(err)
	}
	if err := validateTaskWaitSource(ctx, source); err != nil {
		return nil, wrap(err)
	}
	canonical, err := taskWaitHeaders(source.MoreHeaders, true, source.Microversion)
	if err != nil {
		return nil, wrap(err)
	}
	for key, value := range policy.Headers {
		canonical[key] = value
	}
	client := *source
	client.MoreHeaders = canonical
	collection := source.ServiceURL("tasks")
	if err := rest.ValidateTarget(source, collection); err != nil {
		return nil, wrap(err)
	}
	var cancel context.CancelFunc
	if *policy.Timeout == 0 {
		budget, cancel = context.WithCancel(ctx)
	} else {
		budget, cancel = context.WithTimeout(ctx, *policy.Timeout)
	}
	defer cancel()
	current := ref.String()
	var result *TaskWaitResult
	perform := func(method, endpoint string, body any, code int) (*rest.Response, error) {
		if err := validateTaskWaitSource(budget, source); err != nil {
			return nil, err
		}
		if source.ProviderClient != client.ProviderClient {
			return nil, taskWaitInvalid("task wait provider changed")
		}
		if err := rest.ValidateTarget(source, endpoint); err != nil {
			return nil, err
		}
		response, err := rest.DoJSON(budget, &client, method, endpoint, body, nil, code)
		if err != nil && budget.Err() != nil {
			err = errors.Join(err, budget.Err())
		}
		return response, err
	}
	for {
		response, err := perform(http.MethodGet, collection+"/"+url.PathEscape(current), nil, http.StatusOK)
		if err != nil {
			return result, wrap(err)
		}
		task, fields, err := decodeTaskWait(response)
		if err != nil {
			return result, wrap(err)
		}
		status, err := taskWaitString(fields, "status")
		if err != nil {
			return result, wrap(response.Fail(err))
		}
		if result == nil {
			result = &TaskWaitResult{OriginalID: ref.String(), CurrentID: current}
		}
		result.observed(response, task)
		if err := validateTaskWaitSource(budget, source); err != nil {
			return result, wrap(err)
		}
		if strings.EqualFold(status, target) {
			return result, nil
		}
		if taskWaitFailed(status, policy.FailureStates) {
			if task.Message != taskWaitImportError396 {
				return result, wrap(&resource.FailedStateError{Resource: "tasks", ID: current, Status: status})
			}
			kind, err := taskWaitString(fields, "type")
			if err != nil || kind == "" {
				if err == nil {
					err = fmt.Errorf("task recreation requires nonempty type")
				}
				return result, wrap(response.Fail(err))
			}
			input, exists := fields["input"]
			if !exists {
				return result, wrap(response.Fail(fmt.Errorf("task recreation requires fetched input")))
			}
			body := map[string]any{"type": kind, "input": append(json.RawMessage(nil), input...)}
			created, err := perform(http.MethodPost, collection, body, http.StatusCreated)
			if created != nil {
				result.Created = taskWaitResponse(created, nil)
			}
			if err != nil {
				return result, wrap(err)
			}
			newTask, newFields, err := decodeTaskWait(created)
			if err != nil {
				return result, wrap(err)
			}
			result.Created.Task = copyTaskWaitTask(newTask)
			id, err := taskWaitString(newFields, "id")
			if err == nil {
				err = taskWaitID(id)
			}
			if err != nil {
				return result, wrap(created.Fail(err))
			}
			current = id
			result.CurrentID = current
			result.Recreated++
			result.observed(created, newTask)
		}
		if err := taskWaitPause(budget, *policy.PollInterval); err != nil {
			return result, wrap(err)
		}
	}
}

func taskWaitFailed(status string, failures []string) bool {
	for _, failure := range failures {
		if strings.EqualFold(status, failure) {
			return true
		}
	}
	return false
}

func taskWaitPause(ctx context.Context, interval time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

func validateTaskWaitSource(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return taskWaitInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return taskWaitInvalid("image service client is required")
	}
	if client.Type != "image" {
		return fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if err := rest.ValidateTarget(client, base); err != nil {
		return err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return taskWaitInvalid("image service base must be query-free and end in a slash")
	}
	_, err = taskWaitHeaders(maps.Clone(client.MoreHeaders), true, client.Microversion)
	return err
}

func taskWaitID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return taskWaitInvalid("task ID must be valid UTF-8")
	}
	for _, r := range id {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return taskWaitInvalid("task ID contains whitespace or control characters")
		}
	}
	return nil
}
