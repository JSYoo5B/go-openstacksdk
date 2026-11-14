package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func addDefaultLocationLocations(ctx context.Context, service *image.Service, ref resource.Ref, locationURL string) (*image.AddImageLocationResult, error) {
	return service.AddImageLocation(ctx, ref, locationURL)
}

func addValidatedThenGetLocations(ctx context.Context, service *image.Service, ref resource.Ref, locationURL, algorithm, value string) (ack *image.AddImageLocationResult, locations *image.ImageLocationsResult, err error) {
	ack, err = service.AddImageLocation(ctx, ref, locationURL,
		image.WithAddImageLocationOpts(image.AddImageLocationOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		image.WithImageLocationValidation(algorithm, value),
		request.Header("X-Location-Request", "add"))
	if err != nil {
		return ack, nil, err
	}
	locations, err = service.GetImageLocations(ctx, ref,
		image.WithGetImageLocationsOpts(image.GetImageLocationsOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		request.Header("X-Location-Request", "get"))
	return ack, locations, err
}
