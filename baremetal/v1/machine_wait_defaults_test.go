package v1

import (
	"context"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestMachineWaitDefaultsUnchanged(t *testing.T) {
	c := newMachineConfig(nil)
	if c.err != nil || c.pacing.Delay(1) != 2*time.Second || c.pacing.Delay(3) != 2*time.Second {
		t.Fatalf("pacing=%v err=%v", c.pacing.Delay(1), c.err)
	}
	for _, def := range []time.Duration{3600 * time.Second, 1200 * time.Second, 600 * time.Second} {
		start := time.Now()
		ctx, cancel := c.bound(context.Background(), def)
		deadline, ok := ctx.Deadline()
		cancel()
		if !ok || deadline.Before(start.Add(def)) || deadline.After(time.Now().Add(def)) {
			t.Fatalf("default %v: deadline=%v ok=%t", def, deadline, ok)
		}
	}
	unlimited := newMachineConfig([]MachineOption{WithMachineWaitOptions(resource.WithTimeout(time.Second)), WithMachineWaitOptions(resource.WithUnlimitedWait())})
	ctx, cancel := unlimited.bound(context.Background(), time.Hour)
	defer cancel()
	if _, ok := ctx.Deadline(); ok || unlimited.err != nil {
		t.Fatalf("later unlimited wait must remove the default timeout: err=%v", unlimited.err)
	}
}
