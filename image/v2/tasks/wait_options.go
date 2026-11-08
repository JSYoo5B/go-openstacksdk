package tasks

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TaskWaitOpts controls the SDK-owned task workflow. Wait holds shared
// resource wait options applied in order over the defaults of a 120-second
// timeout, a two-second pause and the "failure" state: WithTimeout,
// WithUnlimitedWait, WithPollInterval, WithWaitStrategy and WithFailureStates
// (no states disable failure detection). Other shared controls fail with
// resource.ErrInvalidOption before HTTP. The parent context always controls
// cancellation.
type TaskWaitOpts struct {
	Wait    []resource.WaitOption
	Headers map[string]string
}

// TaskWaitOption configures the task waiter: WithTaskWaitOpts, WithTaskWait,
// WithTaskWaitHeaders (validates and merges the map), request.Header (one
// request header; case-insensitive later options win) or request.Custom over
// *TaskWaitOpts. Later options replace earlier values. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type TaskWaitOption = request.GetCall

var taskWaitCalls = calls.Spec[TaskWaitOpts]{Header: setTaskWaitHeader}

// taskWaitSupport lists the shared controls the task wait honors.
const taskWaitSupport = resource.WaitSupportFailureStates

// WithTaskWaitOpts snapshots and replaces the complete task policy.
func WithTaskWaitOpts(value TaskWaitOpts) TaskWaitOption {
	snapshot := copyTaskWaitOpts(value)
	return request.Custom(func(config *TaskWaitOpts) error { *config = copyTaskWaitOpts(snapshot); return nil })
}

// WithTaskWait appends shared resource wait options to Wait. Repeated calls
// append; for each control the later option wins.
func WithTaskWait(options ...resource.WaitOption) TaskWaitOption {
	snapshot := slices.Clone(options)
	return request.Custom(func(config *TaskWaitOpts) error {
		config.Wait = append(slices.Clone(config.Wait), snapshot...)
		return nil
	})
}

// setTaskWaitHeader adds a request header. Case-insensitive later options win.
// Authentication, transport and version headers remain owned by the SDK.
func setTaskWaitHeader(config *TaskWaitOpts, key, value string) error {
	headers, err := taskWaitHeaders(map[string]string{key: value}, false, "")
	if err != nil {
		return err
	}
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	for existing := range config.Headers {
		if strings.EqualFold(existing, key) {
			delete(config.Headers, existing)
		}
	}
	for name, value := range headers {
		config.Headers[name] = value
	}
	return nil
}

// WithTaskWaitHeaders snapshots and merges headers, rejecting conflicting
// case aliases within the supplied map.
func WithTaskWaitHeaders(headers map[string]string) TaskWaitOption {
	snapshot := maps.Clone(headers)
	return request.Custom(func(config *TaskWaitOpts) error {
		canonical, err := taskWaitHeaders(snapshot, false, "")
		if err != nil {
			return err
		}
		for key, value := range canonical {
			if err := setTaskWaitHeader(config, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func copyTaskWaitOpts(value TaskWaitOpts) TaskWaitOpts {
	value.Wait = slices.Clone(value.Wait)
	value.Headers = maps.Clone(value.Headers)
	return value
}

// taskWaitPolicy is a parsed task wait policy. Timeout zero is unlimited.
type taskWaitPolicy struct {
	Headers       map[string]string
	Timeout       time.Duration
	Strategy      resource.WaitStrategy
	FailureStates []string
}

func parseTaskWaitOpts(options []TaskWaitOption) (taskWaitPolicy, error) {
	var value TaskWaitOpts
	for _, apply := range calls.Translate(options, taskWaitCalls) {
		if apply == nil {
			return taskWaitPolicy{}, taskWaitInvalid("nil task wait option")
		}
		if err := apply(&value); err != nil {
			return taskWaitPolicy{}, err
		}
	}
	// Keep the option-owned config separate: custom options may retain its pointer.
	snapshot := copyTaskWaitOpts(value)
	settings, err := resource.ResolveWait(resource.WaitDefaults{Timeout: 120 * time.Second, Strategy: resource.FixedWait(2 * time.Second), Supports: taskWaitSupport}, snapshot.Wait...)
	if err != nil {
		return taskWaitPolicy{}, err
	}
	policy := taskWaitPolicy{Timeout: settings.Timeout, Strategy: settings.Strategy, FailureStates: settings.FailureStates}
	if !settings.FailureStatesSet {
		policy.FailureStates = []string{"failure"}
	}
	policy.Headers, err = taskWaitHeaders(snapshot.Headers, false, "")
	return policy, err
}

func taskWaitHeaders(headers map[string]string, source bool, version string) (map[string]string, error) {
	result := make(map[string]string, len(headers))
	for key, value := range headers {
		if key == "" {
			return nil, taskWaitInvalid("invalid task wait header")
		}
		for _, c := range []byte(key) {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
				continue
			}
			return nil, taskWaitInvalid("invalid task wait header %q", key)
		}
		for _, c := range []byte(value) {
			if (c < 32 && c != '\t') || c == 127 {
				return nil, taskWaitInvalid("invalid task wait header value")
			}
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := result[name]; exists && old != value {
			return nil, taskWaitInvalid("conflicting task wait header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version":
			return nil, taskWaitInvalid("header %q is owned by the SDK", key)
		case "openstack-api-version":
			if !source || version == "" || value != "image "+version {
				return nil, taskWaitInvalid("image version header conflicts with selected microversion")
			}
		}
		result[name] = value
	}
	return result, nil
}

func taskWaitInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
