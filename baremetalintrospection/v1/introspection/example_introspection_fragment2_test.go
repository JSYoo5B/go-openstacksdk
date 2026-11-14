package introspection_test

import (
	"context"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_introspectionStartIntrospection() {
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
	manageBoot := false
	err = service.Introspection.StartIntrospection(ctx, "node-uuid",
		introspection.StartOpts{ManageBoot: &manageBoot},
		request.Query("vendor", "a&b"))
	if err != nil {
		log.Fatal(err)
	}
}
