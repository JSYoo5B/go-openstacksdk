package waits_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestKeymanagerWaitDefaultDeadlines(t *testing.T) {
	var deadlines []time.Duration
	api, _, _ := fixture(t, func(req *http.Request, _ int) *http.Response {
		if deadline, ok := req.Context().Deadline(); ok {
			deadlines = append(deadlines, time.Until(deadline))
		} else {
			deadlines = append(deadlines, 0)
		}
		return wire(404, "")
	})
	if _, err := api.WaitForDelete(context.Background(), waits.Container("c")); err != nil {
		t.Fatal(err)
	}
	api2, _, _ := fixture(t, func(req *http.Request, _ int) *http.Response {
		if _, ok := req.Context().Deadline(); ok {
			t.Error("status wait has an SDK deadline by default")
		}
		return wire(200, `{"status":"ACTIVE"}`)
	})
	if _, err := api2.WaitForStatus(context.Background(), waits.Container("c"), str("active")); err != nil {
		t.Fatal(err)
	}
	if len(deadlines) != 1 || deadlines[0] <= 110*time.Second || deadlines[0] > 120*time.Second {
		t.Fatal("delete default deadline", deadlines)
	}
}

func TestKeymanagerWaitLaterOptionWinsAcrossSourceAndSharedControls(t *testing.T) {
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"ACTIVE"}`) })
	if _, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithWait(resource.WithTimeout(time.Second)), waits.WithTimeout(0)); !errors.Is(err, context.DeadlineExceeded) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	if _, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithTimeout(0), waits.WithWait(resource.WithUnlimitedWait())); err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
	*calls = nil
	if _, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithWait(resource.WithPollInterval(time.Millisecond)), waits.WithInterval(-time.Second)); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	if _, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithInterval(-time.Second), waits.WithWait(resource.WithPollInterval(time.Millisecond))); err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
	*calls = nil
	failing, _, failCalls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"B"}`) })
	_, err := failing.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithWait(resource.WithFailureStates("A")), waits.WithWait(resource.WithFailureStates("B")))
	var failed *resource.FailedStateError
	if !errors.As(err, &failed) || failed.Status != "B" || len(*failCalls) != 1 {
		t.Fatal(err, *failCalls)
	}
}

func TestKeymanagerWaitRejectsUnsupportedAndInvalidSharedControlsBeforeHTTP(t *testing.T) {
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Error("reached HTTP"); return wire(500, "") })
	for name, option := range map[string]waits.Option{
		"attribute": waits.WithWait(resource.WithStatusAttribute("status")),
		"failures":  waits.WithWait(resource.WithFailureStates("ERROR")),
	} {
		if _, err := api.WaitForDelete(context.Background(), waits.Container("c"), option); !errors.Is(err, resource.ErrUnsupported) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, option := range map[string]waits.Option{
		"nil":          waits.WithWait(nil),
		"zero timeout": waits.WithWait(resource.WithTimeout(0)),
		"nil callback": waits.WithWait(resource.WithProgressCallback(nil)),
	} {
		if _, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
}
