package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// deleteVolumeProgressFixture answers lookup and DELETE, then deleting polls;
// after absentAfter polls (0 never) it answers 404. A gated request is
// answered by its gate and not counted.
func deleteVolumeProgressFixture(t *testing.T, absentAfter int32, gates ...*testcloud.Gate) (*atomic.Int32, func(context.Context, ...blockstorage.DeleteVolumeOption) (*blockstorage.DeleteVolumeResult, error)) {
	t.Helper()
	cloud := testcloud.New(t)
	client := deleteVolumeContractClient(cloud, "3.23")
	var calls atomic.Int32
	cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		for _, gate := range gates {
			if handled, err := gate.Intercept(r); handled {
				return nil, err
			}
		}
		step := calls.Add(1)
		switch {
		case step == 1:
			return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(deleteVolumeContractBody(`"available"`))), "located"), nil
		case step == 2:
			return deleteVolumeContractResponse(r, 202, io.NopCloser(strings.NewReader("")), "deletion"), nil
		case absentAfter > 0 && step-2 > absentAfter:
			return deleteVolumeContractResponse(r, 404, io.NopCloser(strings.NewReader("")), "absent"), nil
		}
		return deleteVolumeContractResponse(r, 200, io.NopCloser(strings.NewReader(deleteVolumeContractBody(`"deleting"`))), "poll"), nil
	}))
	run := func(ctx context.Context, options ...blockstorage.DeleteVolumeOption) (*blockstorage.DeleteVolumeResult, error) {
		return blockstorage.DeleteVolume(ctx, client, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume-1")}, options...)
	}
	return &calls, run
}

func TestDeleteVolumeProgressChannelSendsZeroPerNonterminalPoll(t *testing.T) {
	calls, run := deleteVolumeProgressFixture(t, 2)
	var got []int
	observer := testcloud.NewProgress(t, func(v int) { got = append(got, v) })
	interval := time.Millisecond
	result, err := run(context.Background(), blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval, Progress: observer.C}))
	observer.Done()
	if err != nil || result == nil || !result.Deleted || calls.Load() != 5 || !reflect.DeepEqual(got, []int{0, 0}) {
		t.Fatal(result, err, calls.Load(), got)
	}
}

func TestDeleteVolumeProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	cause := errors.New("stopped while progress blocked")
	// Nobody receives; the context is cancelled once the send blocks.
	ctx := testcloud.CancelWhenBlocked(t, context.Background(), cause)
	calls, run := deleteVolumeProgressFixture(t, 0)
	blocked := make(chan int)
	interval := time.Millisecond
	result, err := run(ctx, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval, Progress: blocked}))
	select {
	case v := <-blocked:
		t.Fatal("progress sent after the wait returned", v)
	default:
	}
	var accepted *resource.ResponseError
	if result == nil || result.Deleted || result.LastAccepted == nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &accepted) || calls.Load() != 3 {
		t.Fatal(result, err, calls.Load())
	}
}

// The receiver takes the first value and cancels only after the next poll
// reaches the gate, which answers it with the request context error. This
// forces the interleaving in which the wait already started its next poll.
func TestDeleteVolumeProgressChannelReceiverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("receiver stopped")
	// Requests: lookup, DELETE, first poll, then the gated second poll.
	gate := testcloud.NewGate(4, func(*http.Request) bool { return true })
	calls, run := deleteVolumeProgressFixture(t, 0, gate)
	var got []int
	observer := testcloud.NewProgress(t, func(value int) {
		got = append(got, value)
		gate.AwaitThen(t, func() { cancel(cause) })
	})
	interval := time.Millisecond
	result, err := run(ctx, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval, Progress: observer.C}))
	observer.Done()
	var accepted *resource.ResponseError
	if result == nil || result.Deleted || result.LastAccepted == nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || errors.As(err, &accepted) || calls.Load() != 3 || gate.Seen() != 4 || !reflect.DeepEqual(got, []int{0}) {
		t.Fatal(result, err, calls.Load(), gate.Seen(), got)
	}
}

func TestDeleteVolumeProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	_, run := deleteVolumeProgressFixture(t, 2)
	// A buffered channel shows what was already sent when each callback runs.
	sent := make(chan int, 4)
	var queuedAtCallback []int
	interval := time.Millisecond
	result, err := run(context.Background(), blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval, Progress: sent,
		ProgressCallback: func(int) error { queuedAtCallback = append(queuedAtCallback, len(sent)); return nil }}))
	if err != nil || result == nil || !result.Deleted || !reflect.DeepEqual(queuedAtCallback, []int{0, 1}) || len(sent) != 2 {
		t.Fatal(result, err, queuedAtCallback, len(sent))
	}
	// A callback error stops the wait and nothing is sent for that step.
	calls, run := deleteVolumeProgressFixture(t, 0)
	stop := errors.New("callback stop")
	sent = make(chan int, 4)
	result, err = run(context.Background(), blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &interval, Progress: sent,
		ProgressCallback: func(int) error { return stop }}))
	if result == nil || result.Deleted || !errors.Is(err, stop) || calls.Load() != 3 || len(sent) != 0 {
		t.Fatal(result, err, calls.Load(), len(sent))
	}
}

func TestAttachAndCreateVolumeProgressChannelSendZeroAfterCallback(t *testing.T) {
	t.Run("attach", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := attachVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
			case 2:
				testcloud.JSON(w, 200, attachVolumeContractCreated())
			case 3:
				testcloud.JSON(w, 200, attachVolumeContractObservation("attaching", `[]`))
			default:
				testcloud.JSON(w, 200, attachVolumeContractObservation("in-use", `null`))
			}
		})
		sent := make(chan int, 4)
		var queuedAtCallback []int
		interval := time.Millisecond
		result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{PollInterval: &interval, Progress: sent,
			ProgressCallback: func(int) error { queuedAtCallback = append(queuedAtCallback, len(sent)); return nil }}))
		if err != nil || result.Ready == nil || calls.Load() != 4 || !reflect.DeepEqual(queuedAtCallback, []int{0}) || len(sent) != 1 || <-sent != 0 {
			t.Fatal(result, err, calls.Load(), queuedAtCallback)
		}
	})
	t.Run("create", func(t *testing.T) {
		cloud := testcloud.New(t)
		cinder, _ := createVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				testcloud.JSON(w, 202, createVolumeContractBody("creating"))
			case 2:
				testcloud.JSON(w, 200, createVolumeContractBody("creating"))
			default:
				testcloud.JSON(w, 200, createVolumeContractBody("available"))
			}
		})
		var got []int
		observer := testcloud.NewProgress(t, func(v int) { got = append(got, v) })
		interval := time.Millisecond
		result, err := blockstorage.CreateVolume(context.Background(), cinder, nil, blockstorage.CreateVolumeRequest{Size: 2}, blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{PollInterval: &interval, Progress: observer.C}))
		observer.Done()
		if err != nil || result.Ready == nil || calls.Load() != 3 || !reflect.DeepEqual(got, []int{0}) {
			t.Fatal(result, err, calls.Load(), got)
		}
	})
}
