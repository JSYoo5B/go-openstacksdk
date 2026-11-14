package containers_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/keymanager/v1"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
)

func keyManager(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.KeyManagerV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_Create() {
	ctx := context.Background()
	api := containers.New(keyManager(ctx).RawClient())
	created, err := api.Create(ctx, containers.CreateOpts{
		Type:       containers.GenericContainer,
		Name:       "web-tls",
		SecretRefs: []containers.SecretRef{{Name: "cert", SecretRef: "https://barbican.example/v1/secrets/cert-id"}},
	})
	if err != nil {
		log.Fatal(err)
	}
	for value, err := range api.List(ctx, containers.WithListOptions(containers.ListOpts{Name: "web-tls"})) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(value.ContainerRef, value.Status)
	}
	fmt.Println(created.ContainerRef)
}
