package objects_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func objectWaitProgressClient(calls *int, terminal int) func(*http.Request) (*http.Response, error) {
	return func(*http.Request) (*http.Response, error) {
		*calls++
		if terminal > 0 && *calls >= terminal {
			return objectMetadataResponse(404, "gone"), nil
		}
		return objectMetadataResponse(200, "present"), nil
	}
}

func TestObjectWaitProgressChannelSendsZeroClearsAndRejectsNil(t *testing.T) {
	calls := 0
	client := objectMetadataClient(objectWaitProgressClient(&calls, 3))
	var got []int
	observer := testcloud.NewProgress(t, func(v int) { got = append(got, v) })
	result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgress(observer.C))
	observer.Done()
	if err != nil || result == nil || !result.Deleted || calls != 3 || !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatal(result, err, calls, got)
	}
	calls = 0
	unused := make(chan int, 4)
	result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgress(unused), objects.WithoutObjectWaitProgress())
	if err != nil || result == nil || !result.Deleted || len(unused) != 0 {
		t.Fatal(result, err, len(unused))
	}
	calls = 0
	result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitProgress(nil))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(result, err, calls)
	}
}

func TestObjectWaitProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	cause := errors.New("stopped while progress blocked")
	// Nobody receives; the context is cancelled once the send blocks.
	ctx := testcloud.CancelWhenBlocked(t, context.Background(), cause)
	calls := 0
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
		calls++
		return objectMetadataResponse(200, "blocked proof"), nil
	})
	blocked := make(chan int)
	result, err := objects.New(client).WaitForDelete(ctx, "c", "o", objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgress(blocked))
	select {
	case v := <-blocked:
		t.Fatal("progress sent after the wait returned", v)
	default:
	}
	objectWaitExternalLast(t, result, 1, 200, "blocked proof")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || result.Complete || calls != 1 {
		t.Fatal(result, err, calls)
	}
}

func TestObjectWaitProgressChannelReceiverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	client := objectMetadataClient(objectWaitProgressClient(&calls, 0))
	observer := testcloud.NewProgress(t, func(int) { cancel() })
	// The receiver cancels concurrently; the hour-long pause observes it.
	result, err := objects.New(client).WaitForDelete(ctx, "c", "o", objects.WithObjectWaitPollInterval(time.Hour), objects.WithObjectWaitProgress(observer.C))
	observer.Done()
	if !errors.Is(err, context.Canceled) || result == nil || result.Complete || calls != 1 {
		t.Fatal(result, err, calls)
	}
}

func TestObjectWaitProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	calls := 0
	client := objectMetadataClient(objectWaitProgressClient(&calls, 3))
	// A buffered channel shows what was already sent when each callback runs.
	sent := make(chan int, 4)
	var queuedAtCallback []int
	result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgress(sent),
		objects.WithObjectWaitProgressCallback(func(int) error { queuedAtCallback = append(queuedAtCallback, len(sent)); return nil }))
	if err != nil || result == nil || !result.Deleted || !reflect.DeepEqual(queuedAtCallback, []int{0, 1}) || len(sent) != 2 {
		t.Fatal(result, err, queuedAtCallback, len(sent))
	}
	// A callback error stops the wait and nothing is sent for that step.
	calls = 0
	stop := errors.New("callback stop")
	sent = make(chan int, 4)
	result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitPollInterval(time.Millisecond), objects.WithObjectWaitProgress(sent),
		objects.WithObjectWaitProgressCallback(func(int) error { return stop }))
	if !errors.Is(err, stop) || result == nil || result.Complete || calls != 1 || len(sent) != 0 {
		t.Fatal(result, err, calls, len(sent))
	}
}
