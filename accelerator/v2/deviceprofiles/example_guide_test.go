package deviceprofiles_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/accelerator/v2"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"time"
)

func accelerator(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Accelerator(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example() {
	ctx := context.Background()
	profiles := deviceprofiles.New(accelerator(ctx).RawClient())
	profile, err := profiles.Create(ctx, deviceprofiles.CreateOpts{
		Name:   "fpga-profile",
		Groups: []deviceprofiles.Group{{"resources:CUSTOM_FPGA": "1", "trait:CUSTOM_ACCEL": "required"}},
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := profiles.Delete(ctx, resource.ID(profile.UUID), resource.WithMissingError()); err != nil {
		log.Fatal(err)
	}
	if err := profiles.WaitDeleted(ctx, resource.ID(profile.UUID), resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Second)); err != nil {
		log.Fatal(err)
	}
	fmt.Println(profile.UUID)
}
