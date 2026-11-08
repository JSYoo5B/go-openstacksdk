package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestHeatCloudWaitStrategy(t *testing.T) {
	t.Parallel()
	script, client := newHeatScript(t)
	for _, strategy := range []resource.WaitStrategy{{}, resource.ExponentialWait(time.Millisecond, 0.5, 0)} {
		if _, err := stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app", Wait: true}, stackcloud.WithWaitStrategy(strategy)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls := script.calls(); len(calls) != 0 {
		t.Fatal("HTTP before validation", calls)
	}
	// Four pauses: two empty event polls, the event that is not the stack,
	// and one more empty poll before the fallback lookup ends the wait.
	_, client = newHeatScript(t,
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("CREATE_COMPLETE", "")},
		heatStep{"GET", "/stacks/app/events?limit=1&sort_dir=desc", "", 200, `{"events":[` + heatEvent("m1", "app", heatStackID, "CREATE_COMPLETE") + `]}`},
		heatStep{"DELETE", "/stacks/" + heatStackID, "", 204, ""},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_IN_PROGRESS", "")},
		heatStep{"GET", "/stacks/app/events?marker=m1&sort_dir=asc", "", 200, `{"events":[` + heatEvent("e9", "server", "srv", "DELETE_COMPLETE") + `]}`},
		heatStep{"GET", "/stacks/app/events?marker=e9&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app/events?marker=e9&sort_dir=asc", "", 200, `{"events":[]}`},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_FAILED", "")},
		heatStep{"GET", "/stacks/app?resolve_outputs=False", "", 200, heatStackBody("DELETE_FAILED", "")})
	started := time.Now()
	// The strategy given after the zero interval wins: 1 + 4 + 16 + 64 ms.
	result, err := stackcloud.DeleteStack(context.Background(), client, stackcloud.DeleteRequest{NameOrID: "app", Wait: true}, heatFast(), stackcloud.WithWaitStrategy(resource.ExponentialWait(time.Millisecond, 4, 0)))
	if !errors.Is(err, stackcloud.ErrDeleteFailed) || result.WaitStatus != "DELETE_FAILED" || time.Since(started) < 85*time.Millisecond {
		t.Fatal(result, err, time.Since(started))
	}
}
