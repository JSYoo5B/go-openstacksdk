package notifications_test

import (
	"context"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/instanceha/v1"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_notificationsFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.InstanceHAV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := service.RawClient()
	api := notifications.New(client)
	notification, err := api.Create(ctx, notifications.CreateOpts{
		Type: "COMPUTE_HOST", Hostname: "compute-01",
		GeneratedTime: time.Now().UTC().Format(time.RFC3339),
	}, notifications.WithCreatePayload(map[string]any{
		"event": "STOPPED", "host_status": "UNKNOWN", "cluster_status": "OFFLINE",
	}))
	if err != nil {
		log.Fatal(err)
	}
	notification, err = api.Wait(ctx, resource.ID(notification.UUID), "finished",
		resource.WithUnlimitedWait(), resource.WithFailureStates("error", "failed"))
}
