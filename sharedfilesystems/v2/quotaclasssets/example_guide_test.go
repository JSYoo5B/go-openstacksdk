package quotaclasssets_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotaclasssets"
)

func sharedFileSystems(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.SharedFileSystemV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	scope, err := quotaclasssets.New(sharedFileSystems(ctx).RawClient()).InClass(ctx, "default")
	if err != nil {
		log.Fatal(err)
	}
	zero := int64(0)
	result, err := scope.Update(ctx, quotaclasssets.UpdateOpts{Shares: &zero})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
