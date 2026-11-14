package metadefnamespaces_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type NamespaceEvidenceMetadefnamespaces1 struct {
	Created *metadefnamespaces.Namespace
	Read    *metadefnamespaces.Namespace
	Rows    []*metadefnamespaces.Namespace
	All     []*metadefnamespaces.Namespace
	Updated *metadefnamespaces.Namespace
	Deleted *metadefnamespaces.Acknowledgement
}

func manageNamespaceMetadefnamespaces1(ctx context.Context, service *image.Service, namespace, replacementNamespace, owner, resourceType string, pageLimit int) (out NamespaceEvidenceMetadefnamespaces1, err error) {
	api := service.API.MetadefNamespaces
	description := "First line\nSecond line"
	protected := false
	out.Created, err = api.Create(ctx, namespace,
		metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{
			Description: &description, Protected: &protected,
		}),
		request.Header("X-Request-Source", "example"),
		metadefnamespaces.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
		request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := "Example"; c.DisplayName = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := description; c.Description = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := "private"; c.Visibility = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := owner; c.Owner = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := false; c.Protected = &v; return nil }))
	if err != nil {
		return out, err
	}

	out.Read, err = api.Get(ctx, namespace,
		metadefnamespaces.WithGetOpts(metadefnamespaces.GetOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefnamespaces.WithGetHeaders(map[string]string{"X-Trace": "get"}),
		request.Custom(func(c *metadefnamespaces.GetOpts) error { v := resourceType; c.ResourceType = &v; return nil }))
	if err != nil {
		return out, err
	}

	listOptions := []metadefnamespaces.ListOption{
		metadefnamespaces.WithListOpts(metadefnamespaces.ListOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefnamespaces.WithListHeaders(map[string]string{"X-Trace": "list"}),
		request.Limit(pageLimit),
		request.Marker(""),
		request.Custom(func(c *metadefnamespaces.ListOpts) error { c.Visibility = "private"; return nil }),
		request.Custom(func(c *metadefnamespaces.ListOpts) error { c.SortKey = "created_at"; return nil }),
		request.Custom(func(c *metadefnamespaces.ListOpts) error { c.SortDir = "desc"; return nil }),
		request.Custom(func(c *metadefnamespaces.ListOpts) error {
			c.ResourceTypes = "OS::Glance::Image,OS::Nova::Server"
			return nil
		}),
		request.MaxItems(20),
		request.Custom(func(c *metadefnamespaces.ListOpts) error { c.SinglePage = false; return nil }),
	}
	for row, listErr := range api.List(ctx, listOptions...) {
		if listErr != nil {
			return out, listErr
		}
		out.Rows = append(out.Rows, row)
	}
	out.All, err = api.All(ctx, listOptions...)
	if err != nil {
		return out, err
	}

	out.Updated, err = api.Update(ctx, namespace,
		metadefnamespaces.WithUpdateOpts(metadefnamespaces.UpdateOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefnamespaces.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
		request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := replacementNamespace; c.Namespace = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := "Example"; c.DisplayName = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := description; c.Description = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := "private"; c.Visibility = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := owner; c.Owner = &v; return nil }),
		request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := false; c.Protected = &v; return nil }))
	if err != nil {
		return out, err
	}

	out.Deleted, err = api.Delete(ctx, replacementNamespace,
		metadefnamespaces.WithDeleteOpts(metadefnamespaces.DeleteOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefnamespaces.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
		request.IgnoreMissing(false))
	return out, err
}
