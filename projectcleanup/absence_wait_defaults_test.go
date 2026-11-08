package projectcleanup

import (
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestAbsenceWaitDefaultsUnchanged(t *testing.T) {
	o, err := parseExecuteOptions(nil)
	if err != nil || o.wait.Timeout != 120*time.Second || o.wait.Unlimited || o.wait.Strategy.Delay(1) != 2*time.Second || o.wait.Strategy.Delay(5) != 2*time.Second {
		t.Fatalf("settings=%+v err=%v", o.wait, err)
	}
	o, err = parseExecuteOptions([]ExecuteOption{WithAbsenceWait(resource.WithUnlimitedWait()), WithAbsenceWait(resource.WithWaitStrategy(resource.LinearWait(time.Second, time.Second, 0)))})
	if err != nil || !o.wait.Unlimited || o.wait.Timeout != 0 || o.wait.Strategy.Delay(3) != 3*time.Second {
		t.Fatalf("settings=%+v err=%v", o.wait, err)
	}
}

func TestWithAbsenceWaitOwnsOptionSlice(t *testing.T) {
	options := []resource.WaitOption{resource.WithPollInterval(time.Second)}
	option := WithAbsenceWait(options...)
	options[0] = resource.WithPollInterval(time.Hour)
	o, err := parseExecuteOptions([]ExecuteOption{option})
	if err != nil || o.wait.Strategy.Delay(1) != time.Second {
		t.Fatalf("settings=%+v err=%v", o.wait, err)
	}
}
