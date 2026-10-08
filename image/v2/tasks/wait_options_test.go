package tasks

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestTaskWaitOptionsSnapshotCustomRetentionAndConcurrentReuse(t *testing.T) {
	timeout, interval := time.Second, time.Millisecond
	failures := []string{"ERROR"}
	headers := map[string]string{"x-proof": "original"}
	option := WithTaskWaitOpts(TaskWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: failures, Headers: headers})
	timeout, interval, failures[0], headers["x-proof"] = time.Hour, time.Hour, "wrong", "wrong"
	first, err := parseTaskWaitOpts([]TaskWaitOption{option})
	if err != nil {
		t.Fatal(err)
	}
	if *first.Timeout != time.Second || *first.PollInterval != time.Millisecond || first.FailureStates[0] != "ERROR" || first.Headers["X-Proof"] != "original" {
		t.Fatalf("creation snapshot lost: %#v", first)
	}
	*first.Timeout, first.FailureStates[0], first.Headers["X-Proof"] = 5*time.Hour, "mutated", "mutated"
	var retained *TaskWaitOpts
	custom := TaskWaitOption(func(value *TaskWaitOpts) error {
		retained = value
		value.FailureStates = []string{"failure"}
		value.Headers = map[string]string{"X-Proof": "custom"}
		return nil
	})
	parsed, err := parseTaskWaitOpts([]TaskWaitOption{custom})
	if err != nil {
		t.Fatal(err)
	}
	retained.FailureStates[0], retained.Headers["X-Proof"] = "late", "late"
	if parsed.FailureStates[0] != "failure" || parsed.Headers["X-Proof"] != "custom" {
		t.Fatalf("custom captured config aliases parsed policy: %#v", parsed)
	}
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := parseTaskWaitOpts([]TaskWaitOption{option})
			if err != nil || *value.Timeout != time.Second || value.FailureStates[0] != "ERROR" || value.Headers["X-Proof"] != "original" {
				t.Errorf("reusable option lost owned values: %#v, %v", value, err)
				return
			}
			value.FailureStates[0], value.Headers["X-Proof"] = "private", "private"
		}()
	}
	group.Wait()
}

func TestTaskWaitOptionsDefaultsOrderingEmptyFailuresAndInvalidValues(t *testing.T) {
	zero := time.Duration(0)
	for _, tc := range []struct {
		name     string
		options  []TaskWaitOption
		timeout  time.Duration
		failures int
	}{
		{"defaults", nil, 120 * time.Second, 1},
		{"unlimited-last", []TaskWaitOption{WithTaskWaitTimeout(time.Second), WithUnlimitedTaskWait()}, 0, 1},
		{"bounded-last", []TaskWaitOption{WithUnlimitedTaskWait(), WithTaskWaitTimeout(time.Second)}, time.Second, 1},
		{"bulk-zero", []TaskWaitOption{WithTaskWaitOpts(TaskWaitOpts{Timeout: &zero})}, 0, 1},
		{"empty-failures", []TaskWaitOption{WithTaskWaitFailureStates()}, 120 * time.Second, 0},
		{"bulk-reset", []TaskWaitOption{WithUnlimitedTaskWait(), WithTaskWaitFailureStates(), WithTaskWaitOpts(TaskWaitOpts{})}, 120 * time.Second, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := parseTaskWaitOpts(tc.options)
			if err != nil || *value.Timeout != tc.timeout || len(value.FailureStates) != tc.failures || *value.PollInterval != 2*time.Second {
				t.Fatalf("policy = %#v, %v", value, err)
			}
		})
	}
	negative := -time.Second
	for _, options := range [][]TaskWaitOption{{nil}, {WithTaskWaitTimeout(0)}, {WithTaskWaitTimeout(-1)}, {WithTaskWaitPollInterval(0)}, {WithTaskWaitOpts(TaskWaitOpts{Timeout: &negative})}, {WithTaskWaitOpts(TaskWaitOpts{PollInterval: &zero})}, {WithTaskWaitFailureStates(" ")}} {
		if _, err := parseTaskWaitOpts(options); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid option accepted: %v", err)
		}
	}
}

func TestTaskWaitHeadersCanonicalLastWinsAndProtectedOwnership(t *testing.T) {
	headers := map[string]string{"x-proof": "first"}
	option := WithTaskWaitHeaders(headers)
	headers["x-proof"] = "late"
	value, err := parseTaskWaitOpts([]TaskWaitOption{option, WithTaskWaitHeader("X-PROOF", "last")})
	if err != nil || len(value.Headers) != 1 || value.Headers["X-Proof"] != "last" {
		t.Fatalf("canonical last wins: %#v %v", value.Headers, err)
	}
	for _, options := range [][]TaskWaitOption{
		{WithTaskWaitHeaders(map[string]string{"X-Proof": "one", "x-proof": "two"})},
		{WithTaskWaitHeader("X-Proof", "line\r\nbreak")},
		{WithTaskWaitHeader("X Bad", "value")},
		{WithTaskWaitHeader("X-Proof", "\x7f")},
	} {
		if _, err := parseTaskWaitOpts(options); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid headers accepted: %v", err)
		}
	}
	for _, name := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Type", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "OpenStack-API-Version", "X-OpenStack-Glance-API-Version"} {
		if _, err := parseTaskWaitOpts([]TaskWaitOption{WithTaskWaitHeader(name, "value")}); !errors.Is(err, resource.ErrInvalidOption) {
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
