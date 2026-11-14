package v1_test

import (
	"context"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/instanceha/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_notificationsWaitForStatus() {
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
	notification, err := service.Notifications.WaitForStatus(ctx,
		resource.ID("22222222-2222-4222-8222-222222222222"), "finished",
		resource.WithFailureStates("ERROR", "FAILED"),
		resource.WithTimeout(3*time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(notification.UUID, notification.Status, notification.Header, notification.Body)
}
