package tapmirrors_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func networkV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_listQuery() {
	ctx := context.Background()
	api := tapmirrors.New(networkV2(ctx).RawClient())
	for mirror, err := range api.List(ctx,
		request.Query("name", "mirror"),
		request.Query("fields", []string{"id", "name"}),
		request.Query("sort_key", []string{"name", "id"}),
		request.Query("sort_dir", []string{"asc", "desc"}),
	) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(mirror)
	}
}
