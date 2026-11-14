package members_test

import (
	"context"
	"fmt"
	"log"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	image "github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/members"
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

func ExampleAPI_Create() {
	ctx := context.Background()
	api := imageService(ctx).API.Members
	imageID, projectID := "image-id", "project-id"
	if _, err := api.Create(ctx, imageID, projectID); err != nil {
		log.Fatal(err)
	}
	member, err := api.Update(ctx, imageID, projectID, members.UpdateOpts{Status: "accepted"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(member.Status)
	for value, err := range api.List(ctx, imageID) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(value.MemberID, value.Status)
	}
	if err := api.Delete(ctx, imageID, projectID); err != nil {
		log.Fatal(err)
	}
}
