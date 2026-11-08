package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageRecordWaitProgressChannelSendsZeroAndRejectsNil(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return taskCoreJSON(req, 200, `{"status":"pending","progress":77}`), nil
		}
		return taskCoreJSON(req, 200, `{"status":"active"}`), nil
	})
	var got []int
	observer := testcloud.NewProgress(t, func(value int) { got = append(got, value) })
	record, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitProgress(observer.C))
	observer.Done()
	if record == nil || err != nil || calls != 3 || !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatal(record, err, calls, got)
	}
	calls = 0
	record, err = New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitProgress(nil))
	if record != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(record, err, calls)
	}
}

func TestImageRecordWaitProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	cause := errors.New("stopped while progress blocked")
	// Nobody receives; the context is cancelled once the send blocks.
	ctx := testcloud.CancelWhenBlocked(t, context.Background(), cause)
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return taskCoreJSON(req, 203, `{"status":"pending"}`), nil
	})
	blocked := make(chan int)
	record, err := New(client).WaitForImageRecordStatus(ctx, imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitPollInterval(time.Nanosecond), WithImageRecordWaitProgress(blocked))
	select {
	case v := <-blocked:
		t.Fatal("progress sent after the wait returned", v)
	default:
	}
	if record == nil || record.StatusCode != 203 || !errors.Is(err, context.Canceled) || !errors.Is(context.Cause(ctx), cause) || calls != 1 {
		t.Fatal(record, err, calls)
	}
}

// The receiver takes the first value and cancels only after the next poll
// reaches the gate, which answers it with the request context error. This
// forces the interleaving in which the wait already started its next poll.
func TestImageRecordWaitProgressChannelReceiverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("receiver stopped")
	gate := testcloud.NewGate(2, func(*http.Request) bool { return true })
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		if handled, err := gate.Intercept(req); handled {
			return nil, err
		}
		calls++
		return taskCoreJSON(req, 203, `{"status":"pending"}`), nil
	})
	var got []int
	observer := testcloud.NewProgress(t, func(value int) {
		got = append(got, value)
		gate.AwaitThen(t, func() { cancel(cause) })
	})
	record, err := New(client).WaitForImageRecordDelete(ctx, imageRecordWaitSeed(json.RawMessage(`"pending"`)), WithImageRecordWaitPollInterval(time.Millisecond), WithImageRecordWaitProgress(observer.C))
	observer.Done()
	if record == nil || record.StatusCode != 203 || !errors.Is(err, context.Canceled) || !errors.Is(context.Cause(ctx), cause) || calls != 1 || gate.Seen() != 2 || !reflect.DeepEqual(got, []int{0}) {
		t.Fatal(record, err, calls, gate.Seen(), got)
	}
}

func TestImageRecordWaitProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return taskCoreJSON(req, 200, `{"status":"pending"}`), nil
		}
		return taskCoreJSON(req, 200, `{"status":"active"}`), nil
	})
	// A buffered channel shows what was already sent when each callback runs.
	sent := make(chan int, 4)
	var queuedAtCallback []int
	record, err := New(client).WaitForImageRecordStatus(context.Background(), imageRecordWaitSeed(json.RawMessage(`"pending"`)), "active", WithImageRecordWaitPollInterval(time.Nanosecond),
		WithImageRecordWaitProgress(sent), WithImageRecordWaitCallback(func(int) { queuedAtCallback = append(queuedAtCallback, len(sent)) }))
	if record == nil || err != nil || !reflect.DeepEqual(queuedAtCallback, []int{0, 1}) || len(sent) != 2 {
		t.Fatal(record, err, queuedAtCallback, len(sent))
	}
}
