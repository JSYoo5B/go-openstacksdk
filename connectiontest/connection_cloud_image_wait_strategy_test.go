package connectiontest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestConnectionCloudImageCreateWaitStrategy(t *testing.T) {
	duplicates := image.WithImageRecordCreateAllowDuplicates(true)
	wait := image.WithImageRecordCreateWait(true)
	invalid := resource.LinearWait(0, time.Millisecond, 0)
	valid := resource.FixedWait(time.Millisecond)
	for name, request := range map[string]sdk.CloudImageCreateRequest{
		"invalid": {Image: image.ImageRecordCreateRequest{Name: "image"}, WaitStrategy: &invalid},
		"both":    {Image: image.ImageRecordCreateRequest{Name: "image"}, WaitStrategy: &valid, PollInterval: time.Millisecond},
	} {
		calls := 0
		conn := cloudCreateConnection(t, nil, &calls)
		if _, err := conn.CreateCloudImageRecord(context.Background(), request, duplicates, wait); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(name, err, calls)
		}
	}
	ms := time.Millisecond
	growing := resource.ExponentialWait(2*ms, 2, 0)
	calls := 0
	conn := cloudCreateConnection(t, []cloudCreateRoute{
		{cloudCreatePost, 201, `{"id":"new","status":"queued"}`},
		{cloudCreateGet, 200, `{"id":"new","status":"queued"}`},
		{cloudCreateGet, 200, `{"id":"new","status":"saving"}`},
		{cloudCreateGet, 200, `{"id":"new","status":"saving"}`},
		{cloudCreateGet, 200, `{"id":"new","status":"active"}`},
	}, &calls)
	started := time.Now()
	result, err := conn.CreateCloudImageRecord(context.Background(), sdk.CloudImageCreateRequest{Image: image.ImageRecordCreateRequest{Name: "image"}, WaitStrategy: &growing}, duplicates, wait)
	// Three pauses of at least 2, 4 and 8 milliseconds precede the last lookup.
	if err != nil || calls != 5 || result.WaitLookups != 4 || time.Since(started) < 14*ms {
		t.Fatal(result, err, calls, time.Since(started))
	}
}
