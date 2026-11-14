package objectstorage_test

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

func Example_cloudContainers() {
	ctx := context.Background()
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	swift, err := conn.ObjectStorage(ctx)
	if err != nil {
		log.Fatal(err)
	}

	created, err := swift.Cloud.CreateContainer(ctx, "photos",
		func(c *objectstorage.ContainerCreateOpts) error { c.Public = true; return nil })
	if err != nil {
		log.Fatal(err)
	}
	if created.Container != nil {
		fmt.Printf("existing=%t name=%s ACL=%s\n", created.Existing,
			created.Container.Resource.Body["name"],
			created.Container.Resource.Body["read_ACL"])
	}

	updated, err := swift.Cloud.SetContainerAccess(ctx, "photos", "private")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("POST=%d requested=%s\n", updated.Acknowledgement.StatusCode, updated.Requested)

	observed, err := swift.Cloud.SetContainerAccess(ctx, "photos", "public",
		func(c *objectstorage.ContainerAccessOpts) error { c.Refresh = true; return nil })
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("POST=%d HEAD=%d observed ACL=%s\n",
		observed.Acknowledgement.StatusCode, observed.Refreshed.StatusCode,
		observed.Container.Resource.Body["read_ACL"])

	access, err := swift.Cloud.GetContainerAccess(ctx, "photos")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("access=%s HEAD=%d\n", access.Access, access.Container.StatusCode)
}
