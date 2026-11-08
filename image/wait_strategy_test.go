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
	valid := resource.FixedWait(time.Millisecond)
	interval := time.Millisecond
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
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWaitStrategy(invalid))
			return err
		},
		"cloud wait zero": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWaitStrategy(resource.WaitStrategy{}))
			return err
		},
		"cloud wait both": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWaitOpts(ImageRecordCloudWaitOpts{PollInterval: &interval, WaitStrategy: &valid}))
			return err
		},
		"cloud delete invalid": func() error {
			_, err := deletes.DeleteCloudImageRecord(ctx, "img", WithImageRecordCloudDeleteWaitStrategy(invalid))
			return err
		},
		"cloud delete both": func() error {
			_, err := deletes.DeleteCloudImageRecord(ctx, "img", WithImageRecordCloudDeleteOpts(ImageRecordCloudDeleteOpts{Wait: true, PollInterval: &interval, WaitStrategy: &valid}))
			return err
		},
		"record wait invalid": func() error {
			_, err := service.WaitForImageRecordStatus(ctx, seed, "active", WithImageRecordWaitStrategy(invalid))
			return err
		},
		"record wait both": func() error {
			_, err := service.WaitForImageRecordStatus(ctx, seed, "active", WithImageRecordWaitOpts(ImageRecordWaitOpts{PollInterval: &interval, WaitStrategy: &valid}))
			return err
		},
		"record delete wait invalid": func() error {
			_, err := service.WaitForImageRecordDelete(ctx, seed, WithImageRecordWaitStrategy(invalid))
			return err
		},
		"create import invalid": func() error {
			_, err := prepareCreateImportWait(&CreateImportWaitOpts{WaitStrategy: &invalid})
			return err
		},
		"create import both": func() error {
			_, err := prepareCreateImportWait(&CreateImportWaitOpts{WaitStrategy: &valid, PollInterval: &interval})
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
	if options, err := prepareCreateImportWait(&CreateImportWaitOpts{WaitStrategy: &valid}); err != nil || len(options) != 2 {
		t.Fatal(len(options), err)
	}
}

func TestImageWaitStrategyLaterOptionWins(t *testing.T) {
	ctx := context.Background()
	check := func(context.Context) error { return nil }
	growing := resource.LinearWait(time.Millisecond, time.Millisecond, 0)
	wait, err := prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{WithImageRecordCloudWaitPollInterval(time.Second), WithImageRecordCloudWaitStrategy(growing)})
	if err != nil || wait.PollInterval != nil || wait.WaitStrategy == nil || *wait.WaitStrategy != growing {
		t.Fatal(wait, err)
	}
	wait, err = prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{WithImageRecordCloudWaitStrategy(growing), WithImageRecordCloudWaitPollInterval(time.Second)})
	if err != nil || wait.WaitStrategy != nil || wait.PollInterval == nil || *wait.PollInterval != time.Second {
		t.Fatal(wait, err)
	}
	deletion, cloudWait, err := prepareImageRecordCloudDelete(ctx, check, []ImageRecordCloudDeleteOption{WithImageRecordCloudDeletePollInterval(time.Second), WithImageRecordCloudDeleteWaitStrategy(growing)})
	if err != nil || deletion.PollInterval != nil || cloudWait.WaitStrategy == nil || *cloudWait.WaitStrategy != growing {
		t.Fatal(deletion, cloudWait, err)
	}
	record, err := prepareImageRecordWait(ctx, check, []ImageRecordWaitOption{WithImageRecordWaitStrategy(growing), WithImageRecordWaitPollInterval(time.Second)}, false)
	if err != nil || record.WaitStrategy != nil || record.PollInterval == nil {
		t.Fatal(record, err)
	}
	record, err = prepareImageRecordWait(ctx, check, []ImageRecordWaitOption{WithImageRecordWaitPollInterval(time.Second), WithImageRecordWaitStrategy(growing)}, false)
	if err != nil || record.PollInterval != nil || record.WaitStrategy == nil {
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
	result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWaitStrategy(resource.ExponentialWait(2*ms, 2, 0)))
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
	got, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"queued"`)), "active", WithImageRecordWaitStrategy(resource.LinearWait(2*ms, 3*ms, 0)))
	if err != nil || got == nil || len(seen) != 4 {
		t.Fatal(got, err, len(seen))
	}
	imageStrategyGaps(t, seen, 2*ms, 5*ms, 8*ms)
}
