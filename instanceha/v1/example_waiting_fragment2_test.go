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

func Example_waitingFragment2() {
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
	scope, err := service.VMoves.InNotification(ctx,
		resource.ID("22222222-2222-4222-8222-222222222222"))
	if err != nil {
		log.Fatal(err)
	}
	move, err := scope.WaitForStatus(ctx,
		resource.ID("55555555-5555-4555-8555-555555555555"), "succeeded",
		resource.WithTimeout(5*time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(move.NotificationID, move.UUID, move.Status)
}
