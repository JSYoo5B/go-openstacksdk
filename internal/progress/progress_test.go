package progress

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSendDeliversAndBlocksUntilReceived(t *testing.T) {
	ch := make(chan int)
	done := make(chan error, 1)
	go func() { done <- Send(context.Background(), ch, 7) }()
	select {
	case err := <-done:
		t.Fatalf("Send returned %v before the value was received", err)
	case <-time.After(20 * time.Millisecond):
	}
	if got := <-ch; got != 7 {
		t.Fatalf("received %d, want 7", got)
	}
	if err := <-done; err != nil {
		t.Fatalf("Send = %v", err)
	}
}

func TestSendNilChannelIsNoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Send[int](ctx, nil, 1); err != nil {
		t.Fatalf("Send(nil) = %v", err)
	}
}

func TestSendStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := make(chan int, 1)
	if err := Send(ctx, ch, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send on a done context = %v", err)
	}
	if len(ch) != 0 {
		t.Fatal("a done context must not deliver the value")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := Send(ctx, make(chan int), 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked Send = %v", err)
	}
}

func TestSendNotifiesOnlyWhenBlocking(t *testing.T) {
	blocked := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(NotifyBlocked(context.Background(), blocked))
	defer cancel()

	ready := make(chan int, 1)
	if err := Send(ctx, ready, 1); err != nil || len(blocked) != 0 {
		t.Fatalf("ready receiver: err=%v notified=%d", err, len(blocked))
	}

	unread := make(chan int)
	done := make(chan error, 1)
	go func() { done <- Send(ctx, unread, 2) }()
	select {
	case <-blocked:
	case <-time.After(10 * time.Second):
		t.Fatal("blocked send was not notified")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked Send = %v", err)
	}
}

func TestSendWithoutOrNilBlockedHook(t *testing.T) {
	for _, ctx := range []context.Context{context.Background(), NotifyBlocked(context.Background(), nil)} {
		ch := make(chan int)
		go func() { <-ch }()
		if err := Send(ctx, ch, 1); err != nil {
			t.Fatalf("Send = %v", err)
		}
	}
	// A full hook channel never blocks the send.
	full := make(chan struct{}, 1)
	full <- struct{}{}
	ch := make(chan int)
	go func() { <-ch }()
	if err := Send(NotifyBlocked(context.Background(), full), ch, 1); err != nil {
		t.Fatalf("Send with full hook = %v", err)
	}
}
