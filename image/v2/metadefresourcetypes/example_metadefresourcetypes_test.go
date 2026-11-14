package metadefresourcetypes_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type AssociationEvidenceMetadefresourcetypes struct {
	GlobalRows []*types.ResourceType
	GlobalAll  []*types.ResourceType
	Created    *types.Association
	Rows       []*types.Association
	All        []*types.Association
	Deleted    *types.Acknowledgement
}

func manageResourceTypeAssociationMetadefresourcetypes(
	ctx context.Context,
	service *image.Service,
	namespace, resourceType, prefix, target string,
) (*AssociationEvidenceMetadefresourcetypes, error) {
	evidence := &AssociationEvidenceMetadefresourcetypes{}
	api := service.API.MetadefResourceTypes
	listOptions := []types.ListOption{
		types.WithListOpts(types.ListOpts{MaxItems: 20}),
		request.Header("X-Example", "resource-types"),
		types.WithListHeaders(map[string]string{"X-Request-Group": "example"}),
		request.MaxItems(20),
	}
	for row, err := range api.List(ctx, listOptions...) {
		if err != nil {
			return evidence, err
		}
		evidence.GlobalRows = append(evidence.GlobalRows, row)
	}
	var err error
	evidence.GlobalAll, err = api.All(ctx, listOptions...)
	if err != nil {
		return evidence, err
	}
	scope, err := api.InNamespace(ctx, namespace)
	if err != nil {
		return evidence, err
	}
	evidence.Created, err = scope.Create(ctx, resourceType,
		types.WithCreateOpts(types.CreateOpts{}),
		request.Header("X-Example", "association"),
		types.WithCreateHeaders(map[string]string{"X-Request-Group": "example"}),
		request.Custom(func(c *types.CreateOpts) error { v := prefix; c.Prefix = &v; return nil }),
		request.Custom(func(c *types.CreateOpts) error { v := target; c.PropertiesTarget = &v; return nil }),
	)
	if err != nil {
		return evidence, err
	}
	for row, err := range scope.List(ctx, listOptions...) {
		if err != nil {
			return evidence, err
		}
		evidence.Rows = append(evidence.Rows, row)
	}
	evidence.All, err = scope.All(ctx, listOptions...)
	if err != nil {
		return evidence, err
	}
	evidence.Deleted, err = scope.Delete(ctx, resourceType,
		types.WithDeleteOpts(types.DeleteOpts{}),
		request.Header("X-Example", "association"),
		types.WithDeleteHeaders(map[string]string{"X-Request-Group": "example"}),
		request.IgnoreMissing(false),
	)
	return evidence, err
}
