package resource_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type progressChannelEntry struct {
	ID       string
	Status   string
	Progress int
}

func progressChannelCollection(get func(context.Context, int32) (*progressChannelEntry, error)) (*resource.Collection[progressChannelEntry], *atomic.Int32) {
	var gets atomic.Int32
	return resource.NewCollection(resource.Adapter[progressChannelEntry]{
		Kind:   "entry",
		ID:     func(value *progressChannelEntry) string { return value.ID },
		Status: func(value *progressChannelEntry) string { return value.Status },
		Get: func(ctx context.Context, _ string) (*progressChannelEntry, error) {
			return get(ctx, gets.Add(1))
		},
	}), &gets
}

func TestWaitProgressChannelSendsNonterminalValuesAndRejectsNil(t *testing.T) {
	for _, deletion := range []bool{false, true} {
		collection, gets := progressChannelCollection(func(_ context.Context, n int32) (*progressChannelEntry, error) {
			if n == 3 {
				if deletion {
					return &progressChannelEntry{ID: "fixed", Status: "deleted"}, nil
				}
				return &progressChannelEntry{ID: "fixed", Status: "ACTIVE", Progress: 100}, nil
			}
			return &progressChannelEntry{ID: "fixed", Status: "BUILD", Progress: int(n) * 10}, nil
		})
		var got []int
		observer := testcloud.NewProgress(t, func(value int) { got = append(got, value) })
		options := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithProgress(observer.C)}
		var err error
		if deletion {
			err = collection.WaitDeleted(context.Background(), resource.ID("fixed"), options...)
		} else {
			_, err = collection.Wait(context.Background(), resource.ID("fixed"), "ACTIVE", options...)
		}
		observer.Done()
		if err != nil || gets.Load() != 3 || !reflect.DeepEqual(got, []int{10, 20}) {
			t.Fatalf("deletion=%t err=%v gets=%d progress=%v", deletion, err, gets.Load(), got)
		}
	}
	collection, gets := progressChannelCollection(func(context.Context, int32) (*progressChannelEntry, error) {
		return &progressChannelEntry{ID: "fixed", Status: "BUILD"}, nil
	})
	if _, err := collection.Wait(context.Background(), resource.ID("fixed"), "ACTIVE", resource.WithProgress(nil)); !errors.Is(err, resource.ErrInvalidOption) || gets.Load() != 0 {
		t.Fatalf("nil progress channel: err=%v gets=%d", err, gets.Load())
	}
	if err := resource.ValidateWaitOptionsFor[progressChannelEntry](resource.WithProgress(nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil progress channel validation: %v", err)
	}
}

func TestWaitProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	for _, deletion := range []bool{false, true} {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("stopped while progress blocked")
		collection, gets := progressChannelCollection(func(_ context.Context, n int32) (*progressChannelEntry, error) {
			if n == 1 {
				// Nobody receives, so the send after this response blocks
				// until the caller cancels.
				time.AfterFunc(20*time.Millisecond, func() { cancel(cause) })
			}
			return &progressChannelEntry{ID: "fixed", Status: "BUILD", Progress: 5}, nil
		})
		blocked := make(chan int)
		options := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithProgress(blocked)}
		var err error
		if deletion {
			err = collection.WaitDeleted(ctx, resource.ID("fixed"), options...)
		} else {
			_, err = collection.Wait(ctx, resource.ID("fixed"), "ACTIVE", options...)
		}
		select {
		case v := <-blocked:
			t.Fatal("progress sent after the wait returned", v)
		default:
		}
		if !errors.Is(err, context.Canceled) || gets.Load() != 1 {
			t.Fatalf("deletion=%t err=%v gets=%d", deletion, err, gets.Load())
		}
	}
}

func TestWaitProgressChannelReceiverCancellationStopsBeforeNextGET(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collection, gets := progressChannelCollection(func(context.Context, int32) (*progressChannelEntry, error) {
		return &progressChannelEntry{ID: "fixed", Status: "BUILD", Progress: 9}, nil
	})
	var got []int
	observer := testcloud.NewProgress(t, func(value int) { got = append(got, value); cancel() })
	// The receiver cancels concurrently; the hour-long pause observes it.
	_, err := collection.Wait(ctx, resource.ID("fixed"), "ACTIVE", resource.WithPollInterval(time.Hour), resource.WithProgress(observer.C))
	observer.Done()
	if !errors.Is(err, context.Canceled) || gets.Load() != 1 || !reflect.DeepEqual(got, []int{9}) {
		t.Fatalf("err=%v gets=%d progress=%v", err, gets.Load(), got)
	}
}

func TestWaitProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	collection, _ := progressChannelCollection(func(_ context.Context, n int32) (*progressChannelEntry, error) {
		if n == 3 {
			return &progressChannelEntry{ID: "fixed", Status: "ACTIVE"}, nil
		}
		return &progressChannelEntry{ID: "fixed", Status: "BUILD", Progress: int(n)}, nil
	})
	// A buffered channel shows what was already sent when each callback runs.
	sent := make(chan int, 4)
	var queuedAtCallback []int
	_, err := collection.Wait(context.Background(), resource.ID("fixed"), "ACTIVE", resource.WithPollInterval(time.Millisecond),
		resource.WithProgress(sent), resource.WithProgressCallback(func(int) { queuedAtCallback = append(queuedAtCallback, len(sent)) }))
	if err != nil || !reflect.DeepEqual(queuedAtCallback, []int{0, 1}) || len(sent) != 2 || <-sent != 1 || <-sent != 2 {
		t.Fatalf("err=%v queued at callback=%v", err, queuedAtCallback)
	}
}
