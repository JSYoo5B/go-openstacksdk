package metadeftags_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type TagEvidenceMetadeftags struct {
	Created *metadeftags.Tag
	Read    *metadeftags.Tag
	Updated *metadeftags.Tag
	Set     *metadeftags.SetResult
	Rows    []*metadeftags.Tag
	All     []*metadeftags.Tag
	Deleted *metadeftags.Acknowledgement
	Cleared *metadeftags.Acknowledgement
}

func manageNamespaceTagsMetadeftags(ctx context.Context, service *image.Service, namespace, current, replacement string, setNames []string, clearNamespace bool) (out TagEvidenceMetadeftags, err error) {
	scope, err := service.API.MetadefTags.InNamespace(ctx, namespace)
	if err != nil {
		return out, err
	}
	out.Created, err = scope.Create(ctx, current,
		metadeftags.WithCreateOpts(metadeftags.CreateOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithCreateHeaders(map[string]string{"X-Trace": "create"}))
	if err != nil {
		return out, err
	}
	out.Read, err = scope.Get(ctx, current,
		metadeftags.WithGetOpts(metadeftags.GetOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithGetHeaders(map[string]string{"X-Trace": "get"}))
	if err != nil {
		return out, err
	}
	out.Updated, err = scope.Update(ctx, current,
		metadeftags.WithUpdateOpts(metadeftags.UpdateOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
		request.Custom(func(c *metadeftags.UpdateOpts) error { v := replacement; c.Name = &v; return nil }))
	if err != nil {
		return out, err
	}
	out.Set, err = scope.Set(ctx, setNames,
		metadeftags.WithSetOpts(metadeftags.SetOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithSetHeaders(map[string]string{"X-Trace": "set"}),
		request.Custom(func(c *metadeftags.SetOpts) error { v := true; c.Append = &v; return nil }))
	if err != nil {
		return out, err
	}

	listOptions := []metadeftags.ListOption{
		metadeftags.WithListOpts(metadeftags.ListOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithListHeaders(map[string]string{"X-Trace": "list"}),
		request.Limit(2),
		request.Marker(""),
		request.Custom(func(c *metadeftags.ListOpts) error { v := "name"; c.SortKey = &v; return nil }),
		request.Custom(func(c *metadeftags.ListOpts) error { v := "asc"; c.SortDir = &v; return nil }),
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
	out.Deleted, err = scope.Delete(ctx, replacement,
		metadeftags.WithDeleteOpts(metadeftags.DeleteOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}))
	if err != nil || !clearNamespace {
		return out, err
	}
	out.Cleared, err = scope.DeleteAll(ctx,
		metadeftags.WithDeleteAllOpts(metadeftags.DeleteAllOpts{}),
		request.Header("X-Request-Source", "example"),
		metadeftags.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
	return out, err
}
