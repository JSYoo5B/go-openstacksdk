package image_test

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func shareAndReadMembers(ctx context.Context, owner, member *image.Service, parent resource.Ref, memberID string) (*image.ImageMemberAcknowledgement, error) {
	common := []image.ImageMemberOption{
		image.WithImageMemberOpts(image.ImageMemberOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		request.Header("X-Member-Request", "single"),
		image.WithImageMemberHeaders(map[string]string{"X-Example": "members"}),
	}
	created, err := owner.AddImageMember(ctx, parent, memberID, common...)
	if err != nil {
		return nil, err
	}
	fmt.Println("created response", created.StatusCode)
	current, err := owner.GetImageMember(ctx, parent, memberID, common...)
	if err != nil {
		return nil, err
	}
	fmt.Println("current response", current.StatusCode)
	updated, err := member.UpdateImageMember(ctx, parent, memberID, "accepted", common...)
	if err != nil {
		return nil, err
	}
	fmt.Println("updated response", updated.StatusCode)

	found, err := owner.FindImageMember(ctx, parent, memberID,
		image.WithFindImageMemberOpts(image.FindImageMemberOpts{}),
		request.Header("X-Member-Request", "find"),
		image.WithFindImageMemberHeaders(map[string]string{"X-Example": "members"}),
		request.IgnoreMissing(false))
	if err != nil {
		return nil, err
	}
	fmt.Println("found response", found.StatusCode)

	listOptions := []image.ListImageMembersOption{
		image.WithListImageMembersOpts(image.ListImageMembersOpts{}),
		request.Header("X-Member-Request", "list"),
		image.WithListImageMembersHeaders(map[string]string{"X-Example": "members"}),
		request.MaxItems(20),
	}
	for row, err := range owner.ListImageMembers(ctx, parent, listOptions...) {
		if err != nil {
			return nil, err
		}
		fmt.Println("member row fields", len(row.Body))
	}
	all, err := owner.AllImageMembers(ctx, parent, listOptions...)
	if err != nil {
		return nil, err
	}
	fmt.Println("collected members", len(all))

	ack, err := owner.RemoveImageMember(ctx, parent, memberID,
		image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{}),
		request.Header("X-Member-Request", "remove"),
		image.WithRemoveImageMemberHeaders(map[string]string{"X-Example": "members"}),
		request.IgnoreMissing(true))
	return ack, err
}
