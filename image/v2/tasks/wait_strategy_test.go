package tasks

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestTaskWaitStrategyRejectedBeforeHTTPAndLaterWins(t *testing.T) {
	calls := 0
	api := taskWaitTestAPI(func(req *http.Request) (*http.Response, error) {
		calls++
		return taskWaitTestHTTP(500, `{}`), nil
	})
	valid := resource.FixedWait(time.Millisecond)
	interval := time.Millisecond
	for name, option := range map[string]TaskWaitOption{
		"invalid": WithTaskWaitStrategy(resource.LinearWait(time.Millisecond, -1, 0)),
		"zero":    WithTaskWaitStrategy(resource.WaitStrategy{}),
		"both":    WithTaskWaitOpts(TaskWaitOpts{PollInterval: &interval, WaitStrategy: &valid}),
	} {
		if _, err := api.WaitForTask(context.Background(), resource.ID("t"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if calls != 0 {
		t.Fatal("HTTP before validation", calls)
	}
	growing := resource.LinearWait(time.Millisecond, time.Millisecond, 0)
	value, err := parseTaskWaitOpts([]TaskWaitOption{WithTaskWaitPollInterval(time.Second), WithTaskWaitStrategy(growing)})
	if err != nil || value.PollInterval != nil || *value.WaitStrategy != growing || taskWaitStrategy(value).Delay(3) != 3*time.Millisecond {
		t.Fatal(value, err)
	}
	value, err = parseTaskWaitOpts([]TaskWaitOption{WithTaskWaitStrategy(growing), WithTaskWaitPollInterval(time.Second)})
	if err != nil || value.WaitStrategy != nil || *value.PollInterval != time.Second || taskWaitStrategy(value).Delay(3) != time.Second {
		t.Fatal(value, err)
	}
}

func TestTaskWaitStrategyPacesPolls(t *testing.T) {
	var seen []time.Time
	api := taskWaitTestAPI(func(req *http.Request) (*http.Response, error) {
		seen = append(seen, time.Now())
		if len(seen) == 4 {
			return taskWaitTestHTTP(200, `{"id":"t","status":"success"}`), nil
		}
		return taskWaitTestHTTP(200, `{"id":"t","status":"processing"}`), nil
	})
	ms := time.Millisecond
	result, err := api.WaitForTask(context.Background(), resource.ID("t"), WithTaskWaitStrategy(resource.ExponentialWait(2*ms, 2, 0)))
	if err != nil || result == nil || len(seen) != 4 {
		t.Fatal(result, err, len(seen))
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := seen[i+1].Sub(seen[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
