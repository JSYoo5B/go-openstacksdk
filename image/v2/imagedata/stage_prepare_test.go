package imagedata

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestPrepareStageOptionsFreezesOnceWithoutRequests(t *testing.T) {
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://example.test/v2/", Type: "image"}
	api := New(client)
	var calls atomic.Int32
	var captured *StageOpts
	option := func(value *StageOpts) error {
		calls.Add(1)
		zero := int64(0)
		value.Size = &zero
		value.Headers = map[string]string{"x-note": "before"}
		captured = value
		return nil
	}
	prepared, err := api.PrepareStageOptions(context.Background(), option)
	if err != nil || calls.Load() != 1 || prepared.Size == nil || *prepared.Size != 0 || prepared.Headers["X-Note"] != "before" {
		t.Fatalf("prepared=%+v calls=%d err=%v", prepared, calls.Load(), err)
	}
	*captured.Size = 7
	captured.Headers["x-note"] = "after"
	if *prepared.Size != 0 || prepared.Headers["X-Note"] != "before" {
		t.Fatal("retained callback configuration mutated prepared policy")
	}
	client.MoreHeaders = map[string]string{"X-OpenStack-Image-Size": "4"}
	_, err = api.PrepareStageOptions(context.Background(), option)
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatalf("source preflight err=%v calls=%d", err, calls.Load())
	}
}
