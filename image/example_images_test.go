package image_test

import (
	"context"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func getImageExampleImages(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
	return svc.GetImage(ctx, resource.ID(imageID),
		image.WithGetImageOpts(image.GetImageOpts{}),
		image.WithGetImageHeaders(map[string]string{"X-Example": "image-get"}),
		request.Header("X-Request-Id", "image-get-example"),
	)
}

func classicImagesExampleImages(ctx context.Context, svc *image.Service, imageName, ownerID string) ([]*image.ImageInfo, error) {
	options := []image.ListImagesOption{
		image.WithListImagesOpts(image.ListImagesOpts{MaxItems: 0}),
		image.WithListImagesHeaders(map[string]string{"X-Example": "image-list"}),
		request.Header("X-Request-Id", "image-list-example"),
		request.Limit(2),
		request.Marker(""),
		request.Custom(func(c *image.ListImagesOpts) error { v := imageName; c.Name = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "private"; c.Visibility = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "all"; c.MemberStatus = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := ownerID; c.Owner = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "active"; c.Status = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := int64(0); c.SizeMin = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := int64(1 << 40); c.SizeMax = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := false; c.Protected = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := false; c.Hidden = &v; return nil }),
		image.WithListImagesSortKeys("name", "created_at"),
		image.WithListImagesSortDirs("asc", "desc"),
		image.WithListImagesTags("ready", "approved"),
		request.Custom(func(c *image.ListImagesOpts) error { v := "gte:2020-01-01T00:00:00Z"; c.CreatedAt = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "lte:2030-01-01T00:00:00Z"; c.UpdatedAt = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "bare"; c.ContainerFormat = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "qcow2"; c.DiskFormat = &v; return nil }),
		image.WithListImagesFilters(url.Values{
			"os_distro": {"ubuntu"}, "unused": {"remove-me"},
		}),
		image.WithListImagesFilter("architecture", "x86_64"),
		image.WithListImagesFilter("unused"),
		request.MaxItems(20),
		request.Custom(func(c *image.ListImagesOpts) error { c.SinglePage = false; return nil }),
	}
	rows := make([]*image.ImageInfo, 0)
	for value, err := range svc.ListImages(ctx, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, value)
	}
	return rows, nil
}

func modernImagesExampleImages(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageInfo, error) {
	return svc.AllImages(ctx,
		request.Custom(func(c *image.ListImagesOpts) error { v := imageID; c.ID = &v; return nil }),
		request.Custom(func(c *image.ListImagesOpts) error { v := "name:asc,created_at:desc"; c.Sort = &v; return nil }),
		request.MaxItems(20),
	)
}
