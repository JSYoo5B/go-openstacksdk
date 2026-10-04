package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/resource"
)

func TestCreateVolumeOptionsEffectiveDefaultsAndBootableForcesWaitAfterReplacement(t *testing.T) {
	defaults, err := blockstorage.PrepareCreateVolumeOptions(context.Background())
	if err != nil || defaults.Wait == nil || !*defaults.Wait || defaults.Bootable != nil || defaults.WaitPolicy.Timeout == nil || *defaults.WaitPolicy.Timeout != 0 || defaults.WaitPolicy.PollInterval == nil || *defaults.WaitPolicy.PollInterval != 2*time.Second || !reflect.DeepEqual(defaults.WaitPolicy.FailureStates, []string{"error"}) {
		t.Fatalf("defaults=%+v error=%v", defaults, err)
	}
	for _, bootable := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			options := []blockstorage.CreateVolumeOption{blockstorage.WithCreateVolumeBootable(bootable), blockstorage.WithCreateVolumeWait(false)}
			if reverse {
				options[0], options[1] = options[1], options[0]
			}
			value, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), options...)
			if err != nil || value.Wait == nil || !*value.Wait || value.Bootable == nil || *value.Bootable != bootable {
				t.Fatalf("bootable=%v reverse=%v policy=%+v error=%v", bootable, reverse, value, err)
			}
		}
	}
	value, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), blockstorage.WithCreateVolumeBootable(true), blockstorage.WithCreateVolumeOptions(blockstorage.CreateVolumeOpts{}), blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{FailureStates: []string{}}))
	if err != nil || *value.Wait || value.Bootable != nil || value.WaitPolicy.FailureStates == nil || len(value.WaitPolicy.FailureStates) != 0 {
		t.Fatalf("complete replacement=%+v error=%v", value, err)
	}
}

func TestCreateVolumeOptionsFactoriesOwnPointersMapsRawValuesAndReusableSnapshots(t *testing.T) {
	name, description, snapshot, volumeType := "worker", "description", "snapshot/literal", "fast"
	disabled, timeout, interval := false, time.Duration(0), time.Millisecond
	metadata := map[string]string{"owner": "original"}
	fields := map[string]json.RawMessage{"volume_image_metadata": json.RawMessage(`{"number":9007199254740993,"enabled":false}`), "metadata": json.RawMessage(`{"raw":"original"}`)}
	hints := map[string]json.RawMessage{"same_host": json.RawMessage(`["server-original"]`)}
	option := blockstorage.WithCreateVolumeOptions(blockstorage.CreateVolumeOpts{Wait: &disabled, Attributes: blockstorage.CreateVolumeAttributes{Name: &name, Description: &description, SnapshotID: &snapshot, VolumeType: &volumeType, Metadata: metadata, Fields: fields, SchedulerHints: hints}, WaitPolicy: blockstorage.CreateVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: []string{}}})
	name, description, snapshot, volumeType, disabled, timeout, interval = "changed", "changed", "changed", "changed", true, time.Hour, time.Hour
	metadata["owner"] = "changed"
	fields["volume_image_metadata"][0] = '!'
	hints["same_host"][0] = '!'
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), option)
			if err != nil || *value.Wait || *value.Attributes.Name != "worker" || *value.Attributes.Description != "description" || *value.Attributes.SnapshotID != "snapshot/literal" || *value.Attributes.VolumeType != "fast" || value.Attributes.Metadata["owner"] != "original" || string(value.Attributes.Fields["volume_image_metadata"]) != `{"number":9007199254740993,"enabled":false}` || string(value.Attributes.SchedulerHints["same_host"]) != `["server-original"]` || *value.WaitPolicy.Timeout != 0 || *value.WaitPolicy.PollInterval != time.Millisecond || value.WaitPolicy.FailureStates == nil {
				t.Errorf("owned policy=%+v error=%v", value, err)
				return
			}
			*value.Attributes.Name = "caller"
			value.Attributes.Metadata["owner"] = "caller"
			value.Attributes.Fields["volume_image_metadata"][0] = '!'
			value.Attributes.SchedulerHints["same_host"][0] = '!'
		}()
	}
	wait.Wait()
	value, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), option, blockstorage.WithCreateVolumeAttributes(blockstorage.CreateVolumeAttributes{}), blockstorage.WithCreateVolumeName("later"), blockstorage.WithCreateVolumeDescription("later description"), blockstorage.WithCreateVolumeSnapshot("later snapshot"), blockstorage.WithCreateVolumeType("later type"), blockstorage.WithCreateVolumeMetadata(map[string]string{}), blockstorage.WithCreateVolumeSchedulerHints(map[string]json.RawMessage{}), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{}))
	if err != nil || *value.Attributes.Name != "later" || *value.Attributes.Description != "later description" || *value.Attributes.SnapshotID != "later snapshot" || *value.Attributes.VolumeType != "later type" || value.Attributes.Metadata == nil || len(value.Attributes.Metadata) != 0 || value.Attributes.Fields == nil || len(value.Attributes.Fields) != 0 || value.Attributes.SchedulerHints == nil || len(value.Attributes.SchedulerHints) != 0 {
		t.Fatalf("attribute replacement=%+v error=%v", value, err)
	}
}

func TestCreateVolumeOptionsCallbacksRunOnceAndPreparedStateDoesNotAliasRetainedHandles(t *testing.T) {
	var retained *blockstorage.CreateVolumeOpts
	name, disabled := "worker", false
	calls := 0
	value, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), func(v *blockstorage.CreateVolumeOpts) error {
		calls++
		v.Wait = &disabled
		v.Attributes.Name = &name
		v.Attributes.Metadata = map[string]string{"owner": "original"}
		v.Attributes.Fields = map[string]json.RawMessage{"volume_image_metadata": json.RawMessage(`{"number":9007199254740993}`)}
		retained = v
		return nil
	}, func(v *blockstorage.CreateVolumeOpts) error {
		calls++
		*retained.Wait = true
		*retained.Attributes.Name = "late"
		retained.Attributes.Metadata["owner"] = "late"
		retained.Attributes.Fields["volume_image_metadata"][0] = '!'
		if *v.Wait || *v.Attributes.Name != "worker" || v.Attributes.Metadata["owner"] != "original" || string(v.Attributes.Fields["volume_image_metadata"]) != `{"number":9007199254740993}` {
			t.Errorf("retained state aliases next callback: %+v", v)
		}
		return nil
	})
	if err != nil || calls != 2 || *value.Wait || *value.Attributes.Name != "worker" || value.Attributes.Metadata["owner"] != "original" {
		t.Fatalf("policy=%+v error=%v calls=%d", value, err, calls)
	}
	cause := errors.New("create option failed")
	calls = 0
	_, err = blockstorage.PrepareCreateVolumeOptions(context.Background(), func(*blockstorage.CreateVolumeOpts) error { return cause }, func(*blockstorage.CreateVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, cause) || calls != 0 {
		t.Fatalf("cause=%v calls=%d", err, calls)
	}
}

func TestCreateVolumeOptionsValidateKnownFieldBoundaryInactiveWaitAndCancellation(t *testing.T) {
	negative, zero := -time.Second, time.Duration(0)
	for _, option := range []blockstorage.CreateVolumeOption{nil, blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{Timeout: &negative}), blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{PollInterval: &zero}), blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{FailureStates: []string{" "}}), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"unknown_vendor": json.RawMessage(`true`)}), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"base_path": json.RawMessage(`"/other"`)}), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"microversion": json.RawMessage(`"3.99"`)}), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"volume_image_metadata": json.RawMessage(`{"unfinished":`)})} {
		_, err := blockstorage.PrepareCreateVolumeOptions(context.Background(), blockstorage.WithCreateVolumeWait(false), option)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("preflight error=%v", err)
		}
	}
	calls := 0
	_, err := blockstorage.PrepareCreateVolumeOptions(nil, func(*blockstorage.CreateVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatalf("nil context=%v calls=%d", err, calls)
	}
	cause := errors.New("cancel create preparation")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err = blockstorage.PrepareCreateVolumeOptions(ctx, func(*blockstorage.CreateVolumeOpts) error { calls++; cancel(cause); return nil }, func(*blockstorage.CreateVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 {
		t.Fatalf("canceled callback error=%v calls=%d", err, calls)
	}
}
