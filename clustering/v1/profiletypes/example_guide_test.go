package profiletypes_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func clustering(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	values, err := profiletypes.New(clustering(ctx).RawClient()).All(ctx,
		profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
		request.Query("vendor_filter", "enabled"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(values))
}
