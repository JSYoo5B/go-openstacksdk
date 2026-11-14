package metadefobjects_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type ObjectEvidenceMetadefobjects struct {
	Created *metadefobjects.Object
	Read    *metadefobjects.Object
	Rows    []*metadefobjects.Object
	All     []*metadefobjects.Object
	Updated *metadefobjects.Object
	Deleted *metadefobjects.Acknowledgement
	Cleared *metadefobjects.Acknowledgement
}

func manageObjectsMetadefobjects(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out ObjectEvidenceMetadefobjects, err error) {
	scope, err := service.API.MetadefObjects.InNamespace(ctx, namespace)
	if err != nil {
		return out, err
	}
	description := "First line\nSecond line"
	properties := map[string]json.RawMessage{
		"vcpus": json.RawMessage(`{"type":"integer","title":"Virtual CPUs","minimum":1}`),
	}
	required := []string{"vcpus"}
	out.Created, err = scope.Create(ctx, name,
		metadefobjects.WithCreateOpts(metadefobjects.CreateOpts{Description: &description}),
		request.Header("X-Request-Source", "example"),
		metadefobjects.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
		request.Custom(func(c *metadefobjects.CreateOpts) error { v := description; c.Description = &v; return nil }),
		metadefobjects.WithCreateProperties(properties),
		metadefobjects.WithCreateRequired(required))
	if err != nil {
		return out, err
	}

	out.Read, err = scope.Get(ctx, name,
		metadefobjects.WithGetOpts(metadefobjects.GetOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefobjects.WithGetHeaders(map[string]string{"X-Trace": "get"}))
	if err != nil {
		return out, err
	}

	listOptions := []metadefobjects.ListOption{
		metadefobjects.WithListOpts(metadefobjects.ListOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefobjects.WithListHeaders(map[string]string{"X-Trace": "list"}),
		request.MaxItems(20),
	}
	for row, listErr := range scope.List(ctx, listOptions...) {
		if listErr != nil {
			return out, listErr
		}
		out.Rows = append(out.Rows, row)
	}
	out.All, err = scope.All(ctx, listOptions...)
	if err != nil {
		return out, err
	}

	out.Updated, err = scope.Update(ctx, name,
		metadefobjects.WithUpdateOpts(metadefobjects.UpdateOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefobjects.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
		request.Custom(func(c *metadefobjects.UpdateOpts) error { v := replacementName; c.Name = &v; return nil }),
		request.Custom(func(c *metadefobjects.UpdateOpts) error { v := description; c.Description = &v; return nil }),
		metadefobjects.WithUpdateProperties(properties),
		metadefobjects.WithUpdateRequired(required))
	if err != nil {
		return out, err
	}

	out.Deleted, err = scope.Delete(ctx, replacementName,
		metadefobjects.WithDeleteOpts(metadefobjects.DeleteOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefobjects.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
		request.IgnoreMissing(false))
	if err != nil || !clearNamespace {
		return out, err
	}
	out.Cleared, err = scope.DeleteAll(ctx,
		metadefobjects.WithDeleteAllOpts(metadefobjects.DeleteAllOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefobjects.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
	return out, err
}
