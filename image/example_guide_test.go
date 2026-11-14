package image_test

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	image "github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"os"
	"time"
)

func imageService(ctx context.Context) *image.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Image(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func ExampleService_Upload() {
	ctx := context.Background()
	service := imageService(ctx)
	data, err := os.Open("ubuntu.qcow2")
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()
	uploaded, err := service.Upload(ctx, image.UploadImageRequest{Name: "ubuntu", Data: data},
		image.WithWait(resource.WithTimeout(5*time.Minute)))
	if err != nil {
		if uploaded != nil {
			fmt.Println("created image:", uploaded.ID)
		}
		log.Fatal(err)
	}
	fmt.Println(uploaded.ID, uploaded.Status)
}

func ExampleWithProperty() {
	ctx := context.Background()
	service := imageService(ctx)
	data, err := os.Open("arm.raw")
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()
	uploaded, err := service.Upload(ctx, image.UploadImageRequest{Name: "arm-image", Data: data},
		image.WithDiskFormat("raw"),
		image.WithMinDisk(0), image.WithMinRAM(512),
		image.WithVisibility(image.VisibilityShared),
		image.WithProtected(false), image.WithHidden(false),
		image.WithTags("linux", "arm64"),
		image.WithProperty("hw_architecture", "aarch64"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(uploaded.ID)
}

func Example_waitAndDelete() {
	ctx := context.Background()
	service := imageService(ctx)
	ready, err := service.Images.Wait(ctx, resource.Name("ubuntu"), "active")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ready.Status)
	if err := service.Images.Delete(ctx, resource.ID(ready.ID)); err != nil {
		log.Fatal(err)
	}
}

func ExampleService_DownloadImageRecord() {
	ctx := context.Background()
	service := imageService(ctx)
	var seed resource.RawResource
	if err := json.Unmarshal([]byte(`{
  "id": "literal-image-id",
  "os_hash_algo": "sha256",
  "os_hash_value": "expected-hex",
  "properties": {"purpose": "download-example"}
}`), &seed); err != nil {
		log.Fatal(err)
	}
	result, err := service.DownloadImageRecord(ctx, image.ImageRecordDownloadRequest{Resource: &seed, Filename: "ubuntu.qcow2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}

func ExampleWithCreateTaskNullInput() {
	ctx := context.Background()
	task, err := imageService(ctx).CreateTask(ctx, "import", image.WithCreateTaskNullInput())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(task)
}

func Example_metadefPropertyResourceType() {
	ctx := context.Background()
	scope, err := imageService(ctx).API.MetadefProperties.InNamespace(ctx, "OS::Compute::CPUPinning")
	if err != nil {
		log.Fatal(err)
	}
	value, err := scope.Get(ctx, "hw_cpu_policy",
		request.Custom(func(c *metadefproperties.GetOpts) error { v := "OS::Nova::Server"; c.ResourceType = &v; return nil }))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(value)
}
