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
	result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWait(resource.WithPollInterval(time.Millisecond)), objects.WithObjectWait(resource.WithProgress(observer.C)))
	observer.Done()
	if err != nil || result == nil || !result.Deleted || calls != 3 || !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatal(result, err, calls, got)
	}
	calls = 0
	unused := make(chan int, 4)
	result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWait(resource.WithProgress(unused)), objects.WithObjectWaitOpts(objects.ObjectWaitOpts{}), objects.WithObjectWait(resource.WithPollInterval(time.Millisecond)))
	if err != nil || result == nil || !result.Deleted || len(unused) != 0 {
		t.Fatal(result, err, len(unused))
	}
	calls = 0
	result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWait(resource.WithProgress(nil)))
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
	result, err := objects.New(client).WaitForDelete(ctx, "c", "o", objects.WithObjectWait(resource.WithPollInterval(time.Millisecond)), objects.WithObjectWait(resource.WithProgress(blocked)))
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

// The receiver takes the first value and cancels only after the next poll
// reaches the gate, which answers it with the request context error. This
// forces the interleaving in which the wait already started its next poll.
func TestObjectWaitProgressChannelReceiverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("receiver stopped")
	gate := testcloud.NewGate(2, func(*http.Request) bool { return true })
	calls := 0
	present := objectWaitProgressClient(&calls, 0)
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		if handled, err := gate.Intercept(r); handled {
			return nil, err
		}
		return present(r)
	})
	var got []int
	observer := testcloud.NewProgress(t, func(value int) {
		got = append(got, value)
		gate.AwaitThen(t, func() { cancel(cause) })
	})
	result, err := objects.New(client).WaitForDelete(ctx, "c", "o", objects.WithObjectWait(resource.WithPollInterval(time.Millisecond)), objects.WithObjectWait(resource.WithProgress(observer.C)))
	observer.Done()
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || result == nil || result.Complete || calls != 1 || gate.Seen() != 2 || !reflect.DeepEqual(got, []int{0}) {
		t.Fatal(result, err, calls, gate.Seen(), got)
	}
	// The gated poll started a physical attempt, so Last records it without
	// an acknowledgement.
	if result.Polls != 2 || result.Last == nil || result.Last.Acknowledgement != nil || len(result.Last.Attempts) != 1 || result.Last.Attempts[0].Response != nil {
		t.Fatalf("last phase %+v", result)
	}
}

func TestObjectWaitProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	calls := 0
	client := objectMetadataClient(objectWaitProgressClient(&calls, 3))
	// A buffered channel shows what was already sent when each callback runs.
	sent := make(chan int, 4)
	var queuedAtCallback []int
	result, err := objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWait(resource.WithPollInterval(time.Millisecond)), objects.WithObjectWait(resource.WithProgress(sent)),
		objects.WithObjectWait(resource.WithProgressCallback(func(int) error { queuedAtCallback = append(queuedAtCallback, len(sent)); return nil })))
	if err != nil || result == nil || !result.Deleted || !reflect.DeepEqual(queuedAtCallback, []int{0, 1}) || len(sent) != 2 {
		t.Fatal(result, err, queuedAtCallback, len(sent))
	}
	// A callback error stops the wait and nothing is sent for that step.
	calls = 0
	stop := errors.New("callback stop")
	sent = make(chan int, 4)
	result, err = objects.New(client).WaitForDelete(context.Background(), "c", "o", objects.WithObjectWait(resource.WithPollInterval(time.Millisecond)), objects.WithObjectWait(resource.WithProgress(sent)),
		objects.WithObjectWait(resource.WithProgressCallback(func(int) error { return stop })))
	if !errors.Is(err, stop) || result == nil || result.Complete || calls != 1 || len(sent) != 0 {
		t.Fatal(result, err, calls, len(sent))
	}
}
