package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type MutationProofMutations struct {
	Added       *image.ImageTagResult
	Removed     *image.ImageTagResult
	Deactivated *image.ImageActionResult
	Reactivated *image.ImageActionResult
}

func mutateImageMutations(ctx context.Context, service *image.Service, ref resource.Ref, tag string) (proof MutationProofMutations, err error) {
	option := request.Header("X-Request-Source", "example")
	proof.Added, err = service.AddTag(ctx, ref, tag, option)
	if err != nil {
		return proof, err
	}
	proof.Removed, err = service.RemoveTag(ctx, ref, tag, option)
	if err != nil {
		return proof, err
	}
	proof.Deactivated, err = service.DeactivateImage(ctx, ref, option)
	if err != nil {
		return proof, err
	}
	proof.Reactivated, err = service.ReactivateImage(ctx, ref, option)
	return proof, err
}
