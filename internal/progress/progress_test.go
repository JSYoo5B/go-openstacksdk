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
