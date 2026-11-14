package introspection_test

import (
	"context"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_introspectionFragment1() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.BareMetalIntrospectionV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	value, err := service.Introspection.WaitUntilFinished(ctx, resource.ID("node-uuid"),
		resource.WithTimeout(10*time.Minute),
		resource.WithPollInterval(3*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(value.UUID, value.Finished, value.State)
}
