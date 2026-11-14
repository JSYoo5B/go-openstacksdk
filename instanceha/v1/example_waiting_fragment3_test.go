package v1_test

import (
	"context"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/instanceha/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_segmentsDelete() {
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
	ref := resource.ID("11111111-1111-4111-8111-111111111111")
	if err := service.Segments.Delete(ctx, ref); err != nil {
		log.Fatal(err)
	}
	if err := service.Segments.WaitForDelete(ctx, ref,
		resource.WithTimeout(10*time.Minute),
		resource.WithPollInterval(time.Second)); err != nil {
		log.Fatal(err)
	}
}
