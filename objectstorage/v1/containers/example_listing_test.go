package containers_test

import (
	"context"
	"flag"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func Example_listing() {
	prefix := flag.String("prefix", "assets/", "container name prefix")
	pageSize := flag.Int("page-size", 0, "server page size; zero disables limit fallback")
	maxItems := flag.Int("max-items", 0, "raw rows to consume; zero means no row cap")
	flag.Parse()
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	swift, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	options := []containers.RecordListOption{
		request.Custom(func(c *containers.RecordListOpts) error { v := *prefix; c.Prefix = &v; return nil }),
		request.Limit(*pageSize),
		request.MaxItems(*maxItems),
	}
	for record, err := range swift.Containers.ListRecords(ctx, options...) {
		if err != nil {
			if record != nil {
				log.Printf("actual page status=%d body=%s", record.StatusCode, record.Envelope)
			}
			log.Fatal(err)
		}
		fmt.Printf("name=%s count=%s bytes=%s HTTP=%d\n",
			record.Resource.Body["name"], record.Resource.Body["count"],
			record.Resource.Body["bytes"], record.StatusCode)
	}
}
