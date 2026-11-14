package objects_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v1 "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func objectStorage(ctx context.Context) *v1.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.ObjectStorageV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleAPI_InContainer() {
	ctx := context.Background()
	scope, err := objectStorage(ctx).Objects.InContainer(ctx, resource.ID("assets"))
	if err != nil {
		log.Fatal(err)
	}
	object, err := scope.Get(ctx, "images/logo ?.png")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(object.Name, object.Bytes, object.ContentType, object.Metadata)
	for object, err := range scope.List(ctx, resource.WithPageSize(100)) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(object.Name, object.Hash)
	}
	header, err := scope.Create(ctx, "docs/readme.txt", objects.CreateOpts{
		Content:  strings.NewReader("hello"),
		Metadata: map[string]string{"Owner": "sdk"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(header)
	download, err := scope.Download(ctx, resource.ID("docs/readme.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer download.Close()
	if _, err := io.Copy(os.Stdout, download); err != nil {
		log.Fatal(err)
	}
}

func ExampleAPI_SetMetadataFromFresh() {
	ctx := context.Background()
	api := objects.New(objectStorage(ctx).RawClient())
	change, err := api.SetMetadataFromFresh(ctx, "assets", "folder/file ?.txt",
		map[string]any{"owner": "ops", "empty": "", "is_content_type_detected": true},
		objects.WithMetadataHeaders(map[string]string{"X-Trans-Id-Extra": "metadata"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(change.Acknowledgement)
	removed, err := api.DeleteMetadataAndRefresh(ctx, "assets", "folder/file ?.txt", []string{"owner"})
	if err != nil {
		log.Fatal(err)
	}
	if !removed.NoOp {
		fmt.Println(removed.Refreshed.Metadata.Values)
	}
}
