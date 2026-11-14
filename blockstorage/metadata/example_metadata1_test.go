package metadata_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func replaceVolumeMetadataMetadata1(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
	scope, err := volumes.New(client).MetadataIn(ctx, resource.ID(id))
	if err != nil {
		return err
	}
	current, err := scope.Get(ctx)
	if err != nil {
		return err
	}
	var options []volumes.MetadataOption
	if etag := current.Header.Get("ETag"); etag != "" {
		options = append(options, request.Header("If-Match", etag))
	}
	result, err := scope.Replace(ctx, map[string]string{"owner": "team-a"}, options...)
	if err != nil {
		return err
	}
	_ = result.Metadata
	_ = result.Body // original response JSON, including envelope extensions
	return nil
}
