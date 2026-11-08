package image

import (
	"context"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageWaitBundlesKeepDefaultsAndSourceValues(t *testing.T) {
	policy, err := resolveImageRecordCloudWait(nil, nil)
	if err != nil || policy.Timeout != 3600*time.Second || policy.Unlimited || policy.Strategy.Delay(3) != 2*time.Second {
		t.Fatal(policy, err)
	}
	zero := time.Duration(0)
	policy, err = resolveImageRecordCloudWait(&zero, nil)
	if err != nil || policy.Timeout != 0 || policy.Unlimited {
		t.Fatal("source zero timeout must stay an expiring budget", policy, err)
	}
	policy, err = resolveImageRecordCloudWait(&zero, []resource.WaitOption{resource.WithTimeout(5 * time.Second), resource.WithUnlimitedWait()})
	if err != nil || !policy.Unlimited {
		t.Fatal(policy, err)
	}
	check := func(context.Context) error { return nil }
	record, err := prepareImageRecordWait(context.Background(), check, nil, false)
	if err != nil || record.Attribute != "status" || record.settings.TimeoutSet || record.settings.StrategySet || record.settings.FailureStatesSet {
		t.Fatal(record, err)
	}
	if strategy := imageRecordCreateTaskStrategy(ImageRecordCreateOpts{}); strategy.Delay(3) != 2*time.Second {
		t.Fatal(strategy.Delay(3))
	}
}

func TestImageWaitBundlesSnapshotStructWaitSlices(t *testing.T) {
	check := func(context.Context) error { return nil }
	ctx := context.Background()
	wait := []resource.WaitOption{resource.WithPollInterval(time.Second)}
	recordOption := WithImageRecordWaitOpts(ImageRecordWaitOpts{Wait: wait})
	cloudOption := WithImageRecordCloudWaitOpts(ImageRecordCloudWaitOpts{Wait: wait})
	deleteOption := WithImageRecordCloudDeleteOpts(ImageRecordCloudDeleteOpts{WaitOptions: wait})
	wait[0] = resource.WithPollInterval(time.Hour)
	record, err := prepareImageRecordWait(ctx, check, []ImageRecordWaitOption{recordOption}, false)
	if err != nil || record.settings.Strategy.Delay(1) != time.Second {
		t.Fatal(record, err)
	}
	cloud, err := prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{cloudOption})
	if err != nil || cloud.Strategy.Delay(1) != time.Second {
		t.Fatal(cloud, err)
	}
	_, deletion, err := prepareImageRecordCloudDelete(ctx, check, []ImageRecordCloudDeleteOption{deleteOption})
	if err != nil || deletion.Strategy.Delay(1) != time.Second {
		t.Fatal(deletion, err)
	}
	// Repeated bundle options append, and the later option wins per control.
	cloud, err = prepareImageRecordCloudWait(ctx, check, []ImageRecordCloudWaitOption{
		WithImageRecordCloudWait(resource.WithTimeout(time.Minute), resource.WithPollInterval(time.Second)),
		WithImageRecordCloudWait(resource.WithTimeout(time.Hour)),
	})
	if err != nil || cloud.Timeout != time.Hour || cloud.Strategy.Delay(1) != time.Second {
		t.Fatal(cloud, err)
	}
}
