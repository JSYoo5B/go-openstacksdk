package tasks

import (
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"gophercloudsdk/resource"
)

// TaskWaitOpts controls the SDK-owned task workflow. Nil Timeout selects 120
// seconds; a zero duration removes the SDK deadline. Nil PollInterval selects
// two seconds. Nil FailureStates selects "failure"; a nonnil empty slice
// disables failure detection. The parent context always controls cancellation.
type TaskWaitOpts struct {
	Timeout       *time.Duration
	PollInterval  *time.Duration
	FailureStates []string
	Headers       map[string]string
}

// TaskWaitOption configures the task waiter. Later options replace earlier values.
type TaskWaitOption func(*TaskWaitOpts) error

// WithTaskWaitOpts snapshots and replaces the complete task policy.
func WithTaskWaitOpts(value TaskWaitOpts) TaskWaitOption {
	snapshot := copyTaskWaitOpts(value)
	return func(config *TaskWaitOpts) error { *config = copyTaskWaitOpts(snapshot); return nil }
}

func WithTaskWaitTimeout(value time.Duration) TaskWaitOption {
	return func(config *TaskWaitOpts) error {
		if value <= 0 {
			return taskWaitInvalid("timeout must be positive")
		}
		owned := value
		config.Timeout = &owned
		return nil
	}
}

// WithUnlimitedTaskWait removes the SDK timeout while retaining the parent context.
func WithUnlimitedTaskWait() TaskWaitOption {
	return func(config *TaskWaitOpts) error { owned := time.Duration(0); config.Timeout = &owned; return nil }
}

func WithTaskWaitPollInterval(value time.Duration) TaskWaitOption {
	return func(config *TaskWaitOpts) error {
		if value <= 0 {
			return taskWaitInvalid("poll interval must be positive")
		}
		owned := value
		config.PollInterval = &owned
		return nil
	}
}

// WithTaskWaitFailureStates replaces failure detection with exact,
// case-insensitive states. No arguments disable failure detection.
func WithTaskWaitFailureStates(states ...string) TaskWaitOption {
	snapshot := append(make([]string, 0, len(states)), states...)
	return func(config *TaskWaitOpts) error {
		config.FailureStates = append(make([]string, 0, len(snapshot)), snapshot...)
		return nil
	}
}

// WithTaskWaitHeader adds a request header. Case-insensitive later options win.
// Authentication, transport and version headers remain owned by the SDK.
func WithTaskWaitHeader(key, value string) TaskWaitOption {
	return func(config *TaskWaitOpts) error {
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
}

// WithTaskWaitHeaders snapshots and merges headers, rejecting conflicting
// case aliases within the supplied map.
func WithTaskWaitHeaders(headers map[string]string) TaskWaitOption {
	snapshot := maps.Clone(headers)
	return func(config *TaskWaitOpts) error {
		canonical, err := taskWaitHeaders(snapshot, false, "")
		if err != nil {
			return err
		}
		for key, value := range canonical {
			if err := WithTaskWaitHeader(key, value)(config); err != nil {
				return err
			}
		}
		return nil
	}
}

func copyTaskWaitOpts(value TaskWaitOpts) TaskWaitOpts {
	if value.Timeout != nil {
		owned := *value.Timeout
		value.Timeout = &owned
	}
	if value.PollInterval != nil {
		owned := *value.PollInterval
		value.PollInterval = &owned
	}
	if value.FailureStates != nil {
		value.FailureStates = append(make([]string, 0, len(value.FailureStates)), value.FailureStates...)
	}
	value.Headers = maps.Clone(value.Headers)
	return value
}

func parseTaskWaitOpts(options []TaskWaitOption) (TaskWaitOpts, error) {
	var value TaskWaitOpts
	for _, apply := range options {
		if apply == nil {
			return value, taskWaitInvalid("nil task wait option")
		}
		if err := apply(&value); err != nil {
			return value, err
		}
	}
	// Keep the option-owned config separate: custom options may retain its pointer.
	snapshot := copyTaskWaitOpts(value)
	if snapshot.Timeout == nil {
		owned := 120 * time.Second
		snapshot.Timeout = &owned
	}
	if *snapshot.Timeout < 0 {
		return snapshot, taskWaitInvalid("timeout must not be negative")
	}
	if snapshot.PollInterval == nil {
		owned := 2 * time.Second
		snapshot.PollInterval = &owned
	}
	if *snapshot.PollInterval <= 0 {
		return snapshot, taskWaitInvalid("poll interval must be positive")
	}
	if snapshot.FailureStates == nil {
		snapshot.FailureStates = []string{"failure"}
	}
	for _, state := range snapshot.FailureStates {
		if strings.TrimSpace(state) == "" {
			return snapshot, taskWaitInvalid("failure state must not be empty")
		}
	}
	headers, err := taskWaitHeaders(snapshot.Headers, false, "")
	snapshot.Headers = headers
	return snapshot, err
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
