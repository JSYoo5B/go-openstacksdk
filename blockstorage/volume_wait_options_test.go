package blockstorage_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// preparedWait resolves prepared WaitOptions. Prepared policies replay every
// control, so the placeholder defaults here never leak into the result.
func preparedWait(t *testing.T, options []resource.WaitOption) resource.WaitSettings {
	t.Helper()
	settings, err := resource.ResolveWait(resource.WaitDefaults{Timeout: time.Nanosecond, Strategy: resource.FixedWait(time.Nanosecond), Supports: resource.WaitSupportFailureStates | resource.WaitSupportStatusAttribute | resource.WaitSupportProgress}, options...)
	if err != nil {
		t.Errorf("prepared wait options: %v", err)
	}
	return settings
}

// failureStates selects failure states; nil keeps the operation default.
func failureStates(states []string) []resource.WaitOption {
	if states == nil {
		return nil
	}
	return []resource.WaitOption{resource.WithFailureStates(states...)}
}

// volumeWaitPolicy builds the interval, timeout and callback options of the
// transport tests at call time.
func volumeWaitPolicy(interval, timeout time.Duration, unlimited bool, callback func(int) error) []resource.WaitOption {
	options := []resource.WaitOption{resource.WithPollInterval(interval), resource.WithTimeout(timeout)}
	if unlimited {
		options = append(options, resource.WithUnlimitedWait())
	}
	if callback != nil {
		options = append(options, resource.WithProgressCallback(callback))
	}
	return options
}

func TestVolumeWaitOptionsPreparedDefaultsAndLaterWins(t *testing.T) {
	ctx := context.Background()
	create, err := blockstorage.PrepareCreateVolumeOptions(ctx)
	if got := preparedWait(t, create.WaitOptions); err != nil || !got.Unlimited || got.Strategy.Delay(1) != 2*time.Second || got.FailureStates[0] != "error" || len(got.FailureStates) != 1 {
		t.Fatalf("create defaults=%+v error=%v", got, err)
	}
	detach, err := blockstorage.PrepareDetachVolumeOptions(ctx, blockstorage.WithDetachVolumeWaitOptions(resource.WithTimeout(time.Minute), resource.WithFailureStates("custom")), blockstorage.WithDetachVolumeWaitOptions(resource.WithPollInterval(time.Second), resource.WithFailureStates()))
	if got := preparedWait(t, detach.WaitOptions); err != nil || got.Timeout != time.Minute || got.Strategy.Delay(1) != time.Second || !got.FailureStatesSet || len(got.FailureStates) != 0 {
		t.Fatalf("detach later wins=%+v error=%v", got, err)
	}
	deletion, err := blockstorage.PrepareDeleteVolumeOptions(ctx, blockstorage.WithDeleteVolumeWaitOptions(resource.WithTimeout(time.Minute)), blockstorage.WithDeleteVolumeWaitOptions(resource.WithUnlimitedWait()))
	if got := preparedWait(t, deletion.WaitOptions); err != nil || !got.Unlimited || got.Strategy.Delay(1) != 2*time.Second || got.FailureStatesSet {
		t.Fatalf("delete later wins=%+v error=%v", got, err)
	}
}

func TestVolumeWaitOptionsStructSliceIsSnapshotted(t *testing.T) {
	options := []resource.WaitOption{resource.WithTimeout(time.Minute)}
	option := blockstorage.WithDeleteVolumeOptions(blockstorage.DeleteVolumeOpts{WaitOptions: options})
	options[0] = resource.WithTimeout(time.Hour)
	value, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), option)
	if got := preparedWait(t, value.WaitOptions); err != nil || got.Timeout != time.Minute {
		t.Fatalf("settings=%+v error=%v", got, err)
	}
	value.WaitOptions[0] = resource.WithTimeout(time.Hour)
	again, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), option)
	if got := preparedWait(t, again.WaitOptions); err != nil || got.Timeout != time.Minute {
		t.Fatalf("returned slice aliased the option: settings=%+v error=%v", got, err)
	}
}

func TestVolumeWaitOptionsRejectUnsupportedControlsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("unsupported wait control reached HTTP", r.URL)
		w.WriteHeader(500)
	})
	client := cloud.Client("volumev3", "/v3")
	_, err := blockstorage.DeleteVolume(context.Background(), client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume")}, blockstorage.WithDeleteVolumeWaitOptions(resource.WithFailureStates("error")))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("delete failure states error=%v", err)
	}
	for _, option := range []resource.WaitOption{resource.WithStatusAttribute("status"), resource.WithFailureStates("error")} {
		if _, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), blockstorage.WithDeleteVolumeWaitOptions(option)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("delete error=%v", err)
		}
	}
	for _, option := range []resource.WaitOption{resource.WithStatusAttribute("status")} {
		if _, err := blockstorage.PrepareDetachVolumeOptions(context.Background(), blockstorage.WithDetachVolumeWaitOptions(option)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("detach error=%v", err)
		}
		if _, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), blockstorage.WithCreateVolumeWaitOptions(option)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("create error=%v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("calls=%d", calls.Load())
	}
}
