package waits_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonKeymanagerWaitProgressChannelSendsZeroAndRejectsNil(t *testing.T) {
	api, _, calls := fixture(t, func(_ *http.Request, n int) *http.Response {
		if n < 3 {
			return wire(200, `{"status":"BUILD","progress":999}`)
		}
		return wire(200, `{"status":"ACTIVE"}`)
	})
	var got []int
	observer := testcloud.NewProgress(t, func(v int) { got = append(got, v) })
	value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithInterval(time.Nanosecond), waits.WithProgress(observer.C))
	observer.Done()
	if err != nil || value == nil || len(*calls) != 3 || !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatal(value, err, *calls, got)
	}
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response { t.Error("nil channel reached HTTP"); return wire(500, "") })
	if value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithProgress(nil)); value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(value, err, *calls)
	}
	if value, err := api.WaitForDelete(context.Background(), waits.Container("c"), waits.WithProgress(nil)); value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(value, err, *calls)
	}
}

func TestPythonKeymanagerWaitProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	for _, deleting := range []bool{false, true} {
		cause := errors.New("stopped while progress blocked")
		// Nobody receives; the context is cancelled once the send blocks.
		ctx := testcloud.CancelWhenBlocked(t, context.Background(), cause)
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"BUILD"}`) })
		blocked := make(chan int)
		var value *waits.Record
		var err error
		if deleting {
			value, err = api.WaitForDelete(ctx, waits.Container("c"), waits.WithInterval(time.Nanosecond), waits.WithProgress(blocked))
		} else {
			value, err = api.WaitForStatus(ctx, waits.Container("c"), str("active"), waits.WithInterval(time.Nanosecond), waits.WithProgress(blocked))
		}
		select {
		case v := <-blocked:
			t.Fatal("progress sent after the wait returned", v)
		default:
		}
		var proof *resource.ResponseError
		if value == nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || len(*calls) != 1 {
			t.Fatal(deleting, value, err, *calls)
		}
	}
}

func TestPythonKeymanagerWaitProgressChannelReceiverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("receiver stopped")
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"BUILD"}`) })
	observer := testcloud.NewProgress(t, func(int) { cancel(cause) })
	// The receiver cancels concurrently; the hour-long pause observes it.
	value, err := api.WaitForStatus(ctx, waits.Container("c"), str("active"), waits.WithInterval(time.Hour), waits.WithProgress(observer.C))
	observer.Done()
	if value == nil || !errors.Is(err, cause) || len(*calls) != 1 {
		t.Fatal(value, err, *calls)
	}
}

func TestPythonKeymanagerWaitProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	api, _, calls := fixture(t, func(_ *http.Request, n int) *http.Response {
		if n < 3 {
			return wire(200, `{"status":"BUILD"}`)
		}
		return wire(200, `{"status":"ACTIVE"}`)
	})
	// A buffered channel shows what was already sent when each callback runs.
	sent := make(chan int, 4)
	var queuedAtCallback []int
	value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithInterval(time.Nanosecond), waits.WithProgress(sent),
		waits.WithProgressCallback(func(int) error { queuedAtCallback = append(queuedAtCallback, len(sent)); return nil }))
	if err != nil || value == nil || len(*calls) != 3 || !reflect.DeepEqual(queuedAtCallback, []int{0, 1}) || len(sent) != 2 {
		t.Fatal(value, err, *calls, queuedAtCallback, len(sent))
	}
	// A callback error stops the wait and nothing is sent for that step.
	stop := errors.New("callback stop")
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"BUILD"}`) })
	sent = make(chan int, 4)
	value, err = api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithInterval(time.Nanosecond), waits.WithProgress(sent),
		waits.WithProgressCallback(func(int) error { return stop }))
	if value == nil || !errors.Is(err, stop) || len(*calls) != 1 || len(sent) != 0 {
		t.Fatal(value, err, *calls, len(sent))
	}
}
