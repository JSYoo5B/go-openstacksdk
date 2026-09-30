package resource_test

import (
	"context"
	"errors"
	"testing"

	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestInvalidReferencesAndIteratorOptionsMakeNoRequests(t *testing.T) {
	cloud := testcloud.New(t)
	service := compute.New(cloud.Client("compute", "/v2.1/project"), compute.Dependencies{})
	ctx := context.Background()
	for _, id := range []string{"", "../servers", "id?query=1", "id%2Fpath"} {
		if _, err := service.Servers.Get(ctx, id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, option := range []resource.ListOption{resource.WithPageSize(0), resource.WithQuery("", "value"), nil} {
		count := 0
		for v, err := range service.Servers.List(ctx, option) {
			count++
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("v=%v err=%v", v, err)
			}
		}
		if count != 1 {
			t.Fatalf("error yields=%d", count)
		}
	}
	if err := resource.Name("name/with?punctuation").Validate(); err != nil {
		t.Fatal(err)
	}
}
