package image

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageRecordCreateTaskWaitOptionsValidateAndLaterWins(t *testing.T) {
	ctx := context.Background()
	check := func(context.Context) error { return nil }
	growing := resource.LinearWait(time.Millisecond, time.Millisecond, 0)
	defaults, err := prepareImageRecordCreateOptions(ctx, check, nil)
	if err != nil || imageRecordCreateTaskStrategy(defaults).Delay(3) != 2*time.Second {
		t.Fatal(defaults, err)
	}
	config, err := prepareImageRecordCreateOptions(ctx, check, []ImageRecordCreateOption{WithImageRecordCreateTaskWait(resource.WithPollInterval(time.Second)), WithImageRecordCreateTaskWait(resource.WithWaitStrategy(growing))})
	if err != nil || imageRecordCreateTaskStrategy(config).Delay(3) != 3*time.Millisecond {
		t.Fatal(config, err)
	}
	config, err = prepareImageRecordCreateOptions(ctx, check, []ImageRecordCreateOption{WithImageRecordCreateTaskWait(resource.WithWaitStrategy(growing)), WithImageRecordCreateTaskWait(resource.WithPollInterval(time.Second))})
	if err != nil || imageRecordCreateTaskStrategy(config).Delay(3) != time.Second {
		t.Fatal(config, err)
	}
	for name, option := range map[string]ImageRecordCreateOption{
		"zero interval":          WithImageRecordCreateTaskWait(resource.WithPollInterval(0)),
		"invalid strategy":       WithImageRecordCreateTaskWait(resource.WithWaitStrategy(resource.ExponentialWait(time.Millisecond, 0.5, 0))),
		"zero strategy":          WithImageRecordCreateTaskWait(resource.WithWaitStrategy(resource.WaitStrategy{})),
		"timeout owned by kwarg": WithImageRecordCreateTaskWait(resource.WithTimeout(time.Second)),
		"unlimited":              WithImageRecordCreateTaskWait(resource.WithUnlimitedWait()),
		"failure states":         WithImageRecordCreateTaskWait(resource.WithFailureStates()),
		"struct field":           WithImageRecordCreateOpts(ImageRecordCreateOpts{TaskWait: []resource.WaitOption{resource.WithProgress(make(chan int))}}),
	} {
		routes := 0
		service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
			routes++
			return taskCoreJSON(req, 500, `{}`), nil
		})
		_, err := service.CreateImageRecord(ctx, ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("bytes")}, imageCreateTaskOptions(WithImageRecordCreateWait(true), option)...)
		if !errors.Is(err, resource.ErrInvalidOption) || routes != 0 {
			t.Errorf("%s: %v routes=%d", name, err, routes)
		}
	}
}

func TestImageRecordCreateTaskWaitStrategyPacesTaskPolls(t *testing.T) {
	var mu sync.Mutex
	var polls []time.Time
	service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/tasks"):
			return taskCoreJSON(req, 201, `{"id":"t1","status":"pending"}`), nil
		case req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/tasks/t1"):
			mu.Lock()
			polls = append(polls, time.Now())
			n := len(polls)
			mu.Unlock()
			if n < 4 {
				return taskCoreJSON(req, 200, `{"id":"t1","status":"processing"}`), nil
			}
			return taskCoreJSON(req, 200, `{"id":"t1","status":"success","result":{"image_id":"imported"}}`), nil
		case req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/images/imported"):
			return taskCoreJSON(req, 200, `{"id":"imported","name":"input","status":"active"}`), nil
		case req.Method == "PATCH":
			return taskCoreJSON(req, 200, `{}`), nil
		case req.Method == "PUT":
			return taskCoreJSON(req, 201, ""), nil
		default:
			return taskCoreJSON(req, 204, ""), nil
		}
	})
	ms := time.Millisecond
	got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("bytes")},
		imageCreateTaskOptions(WithImageRecordCreateWait(true), WithImageRecordCreateTaskWait(resource.WithWaitStrategy(resource.ExponentialWait(2*ms, 2, 0))))...)
	if err != nil || got == nil || got.TaskWait == nil || got.TaskWait.Fetches != 4 {
		t.Fatal(got, err)
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := polls[i+1].Sub(polls[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
