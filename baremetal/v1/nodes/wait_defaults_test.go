package nodes

import (
	"testing"
	"time"
)

func TestNodeWaitDefaultsUnchanged(t *testing.T) {
	for name, cfg := range map[string]waitConfig{"helpers": waitConfigFrom(nil), "provision": provisionWaitFrom(nil).waitConfig} {
		if cfg.err != nil || !cfg.settings.Unlimited || cfg.settings.Timeout != 0 || cfg.settings.Strategy.Delay(1) != 2*time.Second || cfg.settings.Strategy.Delay(5) != 2*time.Second {
			t.Fatalf("%s: settings=%+v err=%v", name, cfg.settings, cfg.err)
		}
	}
	if !provisionWaitFrom(nil).abort {
		t.Fatal("abort on failed state must stay the default")
	}
}
