package allocations

import (
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestAllocationWaitDefaultsUnchanged(t *testing.T) {
	settings, err := resource.ResolveWait(allocationWaitDefaults)
	if err != nil || !settings.Unlimited || settings.Timeout != 0 || settings.Strategy.Delay(1) != 2*time.Second || settings.Strategy.Delay(4) != 2*time.Second {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}
