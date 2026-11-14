package metadefproperties_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type PropertyEvidenceMetadefproperties1 struct {
	Created *metadefproperties.Property
	Read    *metadefproperties.Property
	Rows    []*metadefproperties.Property
	All     []*metadefproperties.Property
	Updated *metadefproperties.Property
	Deleted *metadefproperties.Acknowledgement
	Cleared *metadefproperties.Acknowledgement
}

func managePropertiesMetadefproperties1(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out PropertyEvidenceMetadefproperties1, err error) {
	scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
	if err != nil {
		return out, err
	}
	propertyType, title := "string", "Hypervisor Type"
	description := "First line\nSecond line"
	attributes := map[string]any{"enum": []string{"kvm", "qemu"}, "default": "kvm"}
	out.Created, err = scope.Create(ctx, name,
		metadefproperties.WithCreateOpts(metadefproperties.CreateOpts{
			Type: &propertyType, Title: &title,
			Attributes: map[string]json.RawMessage{"enum": json.RawMessage(`["kvm","qemu"]`)},
		}),
		request.Header("X-Request-Source", "example"),
		metadefproperties.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
		request.Custom(func(c *metadefproperties.CreateOpts) error { v := propertyType; c.Type = &v; return nil }),
		request.Custom(func(c *metadefproperties.CreateOpts) error { v := title; c.Title = &v; return nil }),
		request.Custom(func(c *metadefproperties.CreateOpts) error { v := description; c.Description = &v; return nil }),
		metadefproperties.WithCreateAttributes(attributes),
		metadefproperties.WithCreateAttribute("readonly", false))
	if err != nil {
		return out, err
	}

	out.Read, err = scope.Get(ctx, name,
		metadefproperties.WithGetOpts(metadefproperties.GetOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefproperties.WithGetHeaders(map[string]string{"X-Trace": "get"}),
		request.Custom(func(c *metadefproperties.GetOpts) error { v := ""; c.ResourceType = &v; return nil }))
	if err != nil {
		return out, err
	}

	listOptions := []metadefproperties.ListOption{
		metadefproperties.WithListOpts(metadefproperties.ListOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefproperties.WithListHeaders(map[string]string{"X-Trace": "list"}),
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
		metadefproperties.WithUpdateOpts(metadefproperties.UpdateOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefproperties.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
		request.Custom(func(c *metadefproperties.UpdateOpts) error { v := replacementName; c.Name = &v; return nil }),
		request.Custom(func(c *metadefproperties.UpdateOpts) error { v := propertyType; c.Type = &v; return nil }),
		request.Custom(func(c *metadefproperties.UpdateOpts) error { v := title; c.Title = &v; return nil }),
		request.Custom(func(c *metadefproperties.UpdateOpts) error { v := description; c.Description = &v; return nil }),
		metadefproperties.WithUpdateAttributes(attributes),
		metadefproperties.WithUpdateAttribute("readonly", false))
	if err != nil {
		return out, err
	}

	out.Deleted, err = scope.Delete(ctx, replacementName,
		metadefproperties.WithDeleteOpts(metadefproperties.DeleteOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefproperties.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
		request.IgnoreMissing(false))
	if err != nil || !clearNamespace {
		return out, err
	}
	out.Cleared, err = scope.DeleteAll(ctx,
		metadefproperties.WithDeleteAllOpts(metadefproperties.DeleteAllOpts{}),
		request.Header("X-Request-Source", "example"),
		metadefproperties.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
	return out, err
}
