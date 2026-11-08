package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func imageStrategyGaps(t *testing.T, seen []time.Time, want ...time.Duration) {
	t.Helper()
	if len(seen) < len(want)+1 {
		t.Fatalf("%d polls, want %d", len(seen), len(want)+1)
	}
	for i, d := range want {
		if gap := seen[i+1].Sub(seen[i]); gap < d {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, d)
		}
	}
}

func TestImageWaitStrategyOptionsRejectedBeforeHTTP(t *testing.T) {
	invalid := resource.ExponentialWait(time.Millisecond, 2, time.Microsecond)
	ctx := context.Background()
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		t.Error("unexpected request", req.URL)
		return taskCoreJSON(req, 500, `{}`)
	})
	var routes []string
	deletes := cloudDeleteService(t, "true", &routes, func(route string, req *http.Request) *http.Response {
		t.Error("unexpected request", route)
		return taskCoreJSON(req, 500, `{}`)
	})
	seed := imageRecordWaitSeed(json.RawMessage(`"queued"`))
	for name, call := range map[string]func() error{
		"cloud wait invalid": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWait(resource.WithWaitStrategy(invalid)))
			return err
		},
		"cloud wait zero": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWait(resource.WithWaitStrategy(resource.WaitStrategy{})))
			return err
		},
		"cloud wait unsupported failure states": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWait(resource.WithFailureStates("error")))
			return err
		},
		"cloud delete invalid": func() error {
			_, err := deletes.DeleteCloudImageRecord(ctx, "img", WithImageRecordCloudDeleteWaitOptions(resource.WithWaitStrategy(invalid)))
			return err
		},
		"cloud delete unsupported progress": func() error {
			_, err := deletes.DeleteCloudImageRecord(ctx, "img", WithImageRecordCloudDeleteWait(true), WithImageRecordCloudDeleteWaitOptions(resource.WithProgress(make(chan int))))
			return err
		},
		"record wait invalid": func() error {
			_, err := service.WaitForImageRecordStatus(ctx, seed, "active", WithImageRecordWait(resource.WithWaitStrategy(invalid)))
			return err
		},
		"record delete wait invalid": func() error {
			_, err := service.WaitForImageRecordDelete(ctx, seed, WithImageRecordWait(resource.WithWaitStrategy(invalid)))
			return err
		},
		"create import invalid": func() error {
			_, err := prepareCreateImportWait(CreateImportOpts{Wait: true, WaitOptions: []resource.WaitOption{resource.WithWaitStrategy(invalid)}})
			return err
		},
		"create import unsupported status attribute": func() error {
			_, err := prepareCreateImportWait(CreateImportOpts{Wait: true, WaitOptions: []resource.WaitOption{resource.WithStatusAttribute("status")}})
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if calls != 0 || len(routes) != 0 {
		t.Fatal(calls, routes)
	}
	if options, err := prepareCreateImportWait(CreateImportOpts{Wait: true, WaitOptions: []resource.WaitOption{resource.WithWaitStrategy(resource.FixedWait(time.Millisecond))}}); err != nil || len(options) != 1 {
		t.Fatal(len(options), err)
	}
}

func TestImageWaitStrategyLaterOptionWins(t *testing.T) {
	ctx := context.Background()
	check := func(context.Context) error { return nil }
	growing := resource.LinearWait(time.Millisecond, time.Millisecond, 0)
	wait, err := prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{WithImageRecordCloudWait(resource.WithPollInterval(time.Second)), WithImageRecordCloudWait(resource.WithWaitStrategy(growing))})
	if err != nil || wait.Strategy != growing {
		t.Fatal(wait, err)
	}
	wait, err = prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{WithImageRecordCloudWait(resource.WithWaitStrategy(growing)), WithImageRecordCloudWait(resource.WithPollInterval(time.Second))})
	if err != nil || wait.Strategy.Delay(3) != time.Second {
		t.Fatal(wait, err)
	}
	_, cloudWait, err := prepareImageRecordCloudDelete(ctx, check, []ImageRecordCloudDeleteOption{WithImageRecordCloudDeleteWaitOptions(resource.WithPollInterval(time.Second)), WithImageRecordCloudDeleteWaitOptions(resource.WithWaitStrategy(growing))})
	if err != nil || cloudWait.Strategy != growing {
		t.Fatal(cloudWait, err)
	}
	record, err := prepareImageRecordWait(ctx, check, []ImageRecordWaitOption{WithImageRecordWait(resource.WithWaitStrategy(growing)), WithImageRecordWait(resource.WithPollInterval(time.Second))}, false)
	if err != nil || !record.settings.StrategySet || record.settings.Strategy.Delay(3) != time.Second {
		t.Fatal(record, err)
	}
	record, err = prepareImageRecordWait(ctx, check, []ImageRecordWaitOption{WithImageRecordWait(resource.WithPollInterval(time.Second)), WithImageRecordWait(resource.WithWaitStrategy(growing))}, false)
	if err != nil || record.settings.Strategy != growing {
		t.Fatal(record, err)
	}
}

func TestImageCloudWaitStrategyPacesLookups(t *testing.T) {
	var seen []time.Time
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		seen = append(seen, time.Now())
		if calls == 4 {
			return taskCoreJSON(req, 200, `{"id":"img","status":"active"}`)
		}
		return taskCoreJSON(req, 200, `{"id":"img","status":"queued"}`)
	})
	ms := time.Millisecond
	result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWait(resource.WithWaitStrategy(resource.ExponentialWait(2*ms, 2, 0))))
	if err != nil || result.Image == nil || result.Lookups != 4 {
		t.Fatal(result, err, calls)
	}
	imageStrategyGaps(t, seen, 2*ms, 4*ms, 8*ms)
}

func TestImageRecordWaitStrategyPacesPolls(t *testing.T) {
	var seen []time.Time
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		seen = append(seen, time.Now())
		if len(seen) == 4 {
			return taskCoreJSON(req, 200, `{"status":"active"}`), nil
		}
		return taskCoreJSON(req, 200, `{"status":"queued"}`), nil
	})
	ms := time.Millisecond
	got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"queued"`)), "active", WithImageRecordWait(resource.WithWaitStrategy(resource.LinearWait(2*ms, 3*ms, 0))))
	if err != nil || got == nil || len(seen) != 4 {
		t.Fatal(got, err, len(seen))
	}
	imageStrategyGaps(t, seen, 2*ms, 5*ms, 8*ms)
}
