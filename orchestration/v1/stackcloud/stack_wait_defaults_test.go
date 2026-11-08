package stackcloud

import (
	"context"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestStackWaitDefaultsUnchanged(t *testing.T) {
	client := &gophercloud.ServiceClient{}
	config, err := apply(context.Background(), client, nil)
	if err != nil || !config.wait.Unlimited || config.wait.Timeout != 0 || config.wait.Strategy.Delay(1) != 5*time.Second || config.wait.Strategy.Delay(4) != 5*time.Second {
		t.Fatalf("settings=%+v err=%v", config.wait, err)
	}
	options := []resource.WaitOption{resource.WithPollInterval(time.Second)}
	option := WithStackWait(options...)
	options[0] = resource.WithPollInterval(time.Hour)
	config, err = apply(context.Background(), client, []Option{option, WithStackWait(resource.WithTimeout(time.Minute))})
	if err != nil || config.wait.Unlimited || config.wait.Timeout != time.Minute || config.wait.Strategy.Delay(1) != time.Second {
		t.Fatalf("settings=%+v err=%v", config.wait, err)
	}
}
