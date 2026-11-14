package v1_test

import (
	"context"
	"fmt"
	"log"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	svc "github.com/JSYoo5B/go-openstacksdk/clustering/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_waitingFragment2() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var service *svc.Service
	service, err = conn.ClusteringV1(ctx)
	if err != nil {
		log.Fatal(err)
	}
	profile, err := service.Profiles.WaitForStatus(ctx,
		resource.ID("PROFILE_ID"), "renamed-profile",
		resource.WithStatusAttribute("name"),
		resource.WithFailureStates(),
		resource.WithTimeout(30*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(profile.Name)
}
