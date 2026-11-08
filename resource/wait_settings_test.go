package resource_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestResolveWaitKeepsDefaults(t *testing.T) {
	got, err := resource.ResolveWait(resource.WaitDefaults{Timeout: 90 * time.Second, Strategy: resource.FixedWait(3 * time.Second)})
	if err != nil || got.Timeout != 90*time.Second || got.Unlimited || got.Strategy.Delay(1) != 3*time.Second || got.FailureStatesSet || got.FailureStates != nil || got.StatusAttribute != "" || got.ProgressCallback != nil || got.Progress != nil {
		t.Fatalf("settings=%+v err=%v", got, err)
	}
	got, err = resource.ResolveWait(resource.WaitDefaults{Timeout: time.Minute, Unlimited: true, Strategy: resource.FixedWait(time.Second)})
	if err != nil || got.Timeout != 0 || !got.Unlimited {
		t.Fatalf("unlimited settings=%+v err=%v", got, err)
	}
	for name, defaults := range map[string]resource.WaitDefaults{
		"zero strategy": {Timeout: time.Minute},
		"zero timeout":  {Strategy: resource.FixedWait(time.Second)},
	} {
		if _, err := resource.ResolveWait(defaults); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: err=%v", name, err)
		}
	}
}

func TestResolveWaitLaterOptionWinsPerControl(t *testing.T) {
	defaults := resource.WaitDefaults{Timeout: time.Minute, Strategy: resource.FixedWait(time.Second), Supports: resource.WaitSupportFailureStates | resource.WaitSupportStatusAttribute | resource.WaitSupportProgress}
	progress := make(chan int)
	got, err := resource.ResolveWait(defaults,
		resource.WithTimeout(time.Hour), resource.WithUnlimitedWait(),
		resource.WithWaitStrategy(resource.LinearWait(time.Millisecond, time.Millisecond, 0)), resource.WithPollInterval(5*time.Second),
		resource.WithFailureStates("A"), resource.WithFailureStates(),
		resource.WithStatusAttribute("one"), resource.WithStatusAttribute("two"),
		resource.WithProgress(progress))
	if err != nil || !got.Unlimited || got.Timeout != 0 || got.Strategy.Delay(3) != 5*time.Second || !got.FailureStatesSet || len(got.FailureStates) != 0 || got.StatusAttribute != "two" || got.Progress != progress {
		t.Fatalf("settings=%+v err=%v", got, err)
	}
	got, err = resource.ResolveWait(defaults, resource.WithUnlimitedWait(), resource.WithTimeout(time.Second), resource.WithPollInterval(time.Hour), resource.WithWaitStrategy(resource.FixedWait(time.Millisecond)))
	if err != nil || got.Unlimited || got.Timeout != time.Second || got.Strategy.Delay(1) != time.Millisecond {
		t.Fatalf("settings=%+v err=%v", got, err)
	}
}

func TestResolveWaitRejectsUnsupportedAndInvalidControls(t *testing.T) {
	defaults := resource.WaitDefaults{Timeout: time.Minute, Strategy: resource.FixedWait(time.Second)}
	for name, option := range map[string]resource.WaitOption{
		"failure states":    resource.WithFailureStates("ERROR"),
		"empty failures":    resource.WithFailureStates(),
		"status attribute":  resource.WithStatusAttribute("state"),
		"progress callback": resource.WithProgressCallback(func(int) error { return nil }),
		"progress channel":  resource.WithProgress(make(chan int)),
		"nil option":        nil,
		"zero timeout":      resource.WithTimeout(0),
		"zero interval":     resource.WithPollInterval(0),
		"zero strategy":     resource.WithWaitStrategy(resource.WaitStrategy{}),
	} {
		if _, err := resource.ResolveWait(defaults, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: err=%v", name, err)
		}
	}
}

func TestWithWaitSettingsReplaysResolvedPolicyAndOwnsFailures(t *testing.T) {
	states := []string{"BROKEN"}
	settings, err := resource.ResolveWait(resource.WaitDefaults{Timeout: time.Minute, Strategy: resource.FixedWait(time.Millisecond), Supports: resource.WaitSupportFailureStates}, resource.WithFailureStates(states...))
	if err != nil {
		t.Fatal(err)
	}
	option := resource.WithWaitSettings(settings)
	settings.FailureStates[0] = "ACTIVE"
	collection, gets := progressChannelCollection(func(_ context.Context, n int32) (*progressChannelEntry, error) {
		if n == 2 {
			return &progressChannelEntry{ID: "fixed", Status: "BROKEN"}, nil
		}
		return &progressChannelEntry{ID: "fixed", Status: "BUILD"}, nil
	})
	_, err = collection.Wait(context.Background(), resource.ID("fixed"), "ACTIVE", option)
	var failed *resource.FailedStateError
	if !errors.As(err, &failed) || failed.Status != "BROKEN" || gets.Load() != 2 {
		t.Fatalf("err=%v gets=%d", err, gets.Load())
	}
}

func TestWaitProgressCallbackErrorStopsWaitWithoutSend(t *testing.T) {
	stop := errors.New("stop")
	for _, deletion := range []bool{false, true} {
		collection, gets := progressChannelCollection(func(_ context.Context, n int32) (*progressChannelEntry, error) {
			return &progressChannelEntry{ID: "fixed", Status: "BUILD", Progress: int(n)}, nil
		})
		sent := make(chan int, 4)
		var seen []int
		options := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithProgress(sent), resource.WithProgressCallback(func(value int) error {
			seen = append(seen, value)
			if value == 2 {
				return stop
			}
			return nil
		})}
		var err error
		if deletion {
			err = collection.WaitDeleted(context.Background(), resource.ID("fixed"), options...)
		} else {
			_, err = collection.Wait(context.Background(), resource.ID("fixed"), "ACTIVE", options...)
		}
		if !errors.Is(err, stop) || gets.Load() != 2 || !reflect.DeepEqual(seen, []int{1, 2}) || len(sent) != 1 || <-sent != 1 {
			t.Fatalf("deletion=%t err=%v gets=%d seen=%v sent=%d", deletion, err, gets.Load(), seen, len(sent))
		}
	}
}
