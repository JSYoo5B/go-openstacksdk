package compute_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestFlavorUnsupportedPoliciesAndCreateValidation(t *testing.T) {
	cloud := testcloud.New(t)
	service := compute.New(cloud.Client("compute", "/v2.1/project"), compute.Dependencies{})
	ctx := context.Background()
	if _, err := service.Flavors.All(ctx, resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := service.Servers.Create(ctx, compute.CreateServerRequest{Name: "server", Image: resource.ID("image"), Flavor: resource.Ref{}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := service.Servers.Create(ctx, compute.CreateServerRequest{Name: "server", Image: resource.Name("ubuntu"), Flavor: resource.ID("flavor")}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}
