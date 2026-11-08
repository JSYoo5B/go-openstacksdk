package tasks

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestTaskWaitOptionsSnapshotCustomRetentionAndConcurrentReuse(t *testing.T) {
	wait := []resource.WaitOption{resource.WithTimeout(time.Second), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("ERROR")}
	headers := map[string]string{"x-proof": "original"}
	option := WithTaskWaitOpts(TaskWaitOpts{Wait: wait, Headers: headers})
	wait[0], wait[2], headers["x-proof"] = resource.WithTimeout(time.Hour), resource.WithFailureStates("wrong"), "wrong"
	first, err := parseTaskWaitOpts([]TaskWaitOption{option})
	if err != nil {
		t.Fatal(err)
	}
	if first.Timeout != time.Second || first.Strategy.Delay(1) != time.Millisecond || first.FailureStates[0] != "ERROR" || first.Headers["X-Proof"] != "original" {
		t.Fatalf("creation snapshot lost: %#v", first)
	}
	first.FailureStates[0], first.Headers["X-Proof"] = "mutated", "mutated"
	var retained *TaskWaitOpts
	custom := TaskWaitOption(request.Custom(func(value *TaskWaitOpts) error {
		retained = value
		value.Wait = []resource.WaitOption{resource.WithFailureStates("failure")}
		value.Headers = map[string]string{"X-Proof": "custom"}
		return nil
	}))
	parsed, err := parseTaskWaitOpts([]TaskWaitOption{custom})
	if err != nil {
		t.Fatal(err)
	}
	retained.Wait[0], retained.Headers["X-Proof"] = resource.WithFailureStates("late"), "late"
	if parsed.FailureStates[0] != "failure" || parsed.Headers["X-Proof"] != "custom" {
		t.Fatalf("custom captured config aliases parsed policy: %#v", parsed)
	}
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := parseTaskWaitOpts([]TaskWaitOption{option})
			if err != nil || value.Timeout != time.Second || value.FailureStates[0] != "ERROR" || value.Headers["X-Proof"] != "original" {
				t.Errorf("reusable option lost owned values: %#v, %v", value, err)
				return
			}
			value.FailureStates[0], value.Headers["X-Proof"] = "private", "private"
		}()
	}
	group.Wait()
}

func TestTaskWaitOptionsDefaultsOrderingEmptyFailuresAndInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name     string
		options  []TaskWaitOption
		timeout  time.Duration
		failures int
	}{
		{"defaults", nil, 120 * time.Second, 1},
		{"unlimited-last", []TaskWaitOption{WithTaskWait(resource.WithTimeout(time.Second)), WithTaskWait(resource.WithUnlimitedWait())}, 0, 1},
		{"bounded-last", []TaskWaitOption{WithTaskWait(resource.WithUnlimitedWait()), WithTaskWait(resource.WithTimeout(time.Second))}, time.Second, 1},
		{"bulk-unlimited", []TaskWaitOption{WithTaskWaitOpts(TaskWaitOpts{Wait: []resource.WaitOption{resource.WithUnlimitedWait()}})}, 0, 1},
		{"empty-failures", []TaskWaitOption{WithTaskWait(resource.WithFailureStates())}, 120 * time.Second, 0},
		{"bulk-reset", []TaskWaitOption{WithTaskWait(resource.WithUnlimitedWait(), resource.WithFailureStates()), WithTaskWaitOpts(TaskWaitOpts{})}, 120 * time.Second, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := parseTaskWaitOpts(tc.options)
			if err != nil || value.Timeout != tc.timeout || len(value.FailureStates) != tc.failures || value.Strategy.Delay(1) != 2*time.Second {
				t.Fatalf("policy = %#v, %v", value, err)
			}
		})
	}
	for _, options := range [][]TaskWaitOption{{nil}, {WithTaskWait(nil)}, {WithTaskWait(resource.WithTimeout(0))}, {WithTaskWait(resource.WithTimeout(-1))}, {WithTaskWait(resource.WithPollInterval(0))}, {WithTaskWait(resource.WithFailureStates(" "))}} {
		if _, err := parseTaskWaitOpts(options); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid option accepted: %v", err)
		}
	}
}

func TestTaskWaitOptionsAppendLaterWinsAndRejectUnsupported(t *testing.T) {
	value, err := parseTaskWaitOpts([]TaskWaitOption{
		WithTaskWait(resource.WithTimeout(time.Second), resource.WithFailureStates("A")),
		WithTaskWait(resource.WithFailureStates("B")),
	})
	if err != nil || value.Timeout != time.Second || len(value.FailureStates) != 1 || value.FailureStates[0] != "B" {
		t.Fatalf("policy = %#v, %v", value, err)
	}
	for name, option := range map[string]resource.WaitOption{
		"status attribute":  resource.WithStatusAttribute("status"),
		"progress callback": resource.WithProgressCallback(func(int) error { return nil }),
		"progress channel":  resource.WithProgress(make(chan int)),
	} {
		if _, err := parseTaskWaitOpts([]TaskWaitOption{WithTaskWait(option)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s accepted: %v", name, err)
		}
	}
}

func TestTaskWaitHeadersCanonicalLastWinsAndProtectedOwnership(t *testing.T) {
	headers := map[string]string{"x-proof": "first"}
	option := WithTaskWaitHeaders(headers)
	headers["x-proof"] = "late"
	value, err := parseTaskWaitOpts([]TaskWaitOption{option, request.Header("X-PROOF", "last")})
	if err != nil || len(value.Headers) != 1 || value.Headers["X-Proof"] != "last" {
		t.Fatalf("canonical last wins: %#v %v", value.Headers, err)
	}
	for _, options := range [][]TaskWaitOption{
		{WithTaskWaitHeaders(map[string]string{"X-Proof": "one", "x-proof": "two"})},
		{request.Header("X-Proof", "line\r\nbreak")},
		{request.Header("X Bad", "value")},
		{request.Header("X-Proof", "\x7f")},
	} {
		if _, err := parseTaskWaitOpts(options); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid headers accepted: %v", err)
		}
	}
	for _, name := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Type", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "OpenStack-API-Version", "X-OpenStack-Glance-API-Version"} {
		if _, err := parseTaskWaitOpts([]TaskWaitOption{request.Header(name, "value")}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("protected header %s accepted: %v", name, err)
		}
	}
	if _, err := taskWaitHeaders(map[string]string{"openstack-api-version": "image 2.9"}, true, "2.9"); err != nil {
		t.Fatal(err)
	}
	if _, err := taskWaitHeaders(map[string]string{"OpenStack-API-Version": "image 2.8"}, true, "2.9"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("source version conflict accepted: %v", err)
	}
}
