package keymanagerread

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Delete owns one fixed member request. Accepted bodies are passive bytes;
// decoding them would add a result contract absent from the Python proxy.
func Delete(ctx context.Context, client *gophercloud.ServiceClient, kind, id string) error {
	if err := cloudread.Context(ctx); err != nil {
		return err
	}
	if kind != "containers" && kind != "orders" && kind != "secrets" {
		return invalid("unsupported key-manager delete resource")
	}
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return invalid("delete request ID must be valid text without controls")
	}
	source, err := cloudread.Capture(ctx, client, "key-manager")
	if err != nil {
		return err
	}
	guard := func(ctx context.Context) error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	if err := guard(ctx); err != nil {
		return err
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	target := source.Client.ServiceURL(kind, url.PathEscape(id))
	response, err := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodDelete, target, nil, nil, codes...)
	if final := guard(ctx); final != nil {
		if response != nil && err == nil {
			err = response.Fail(final)
		} else {
			err = errors.Join(err, final)
		}
	}
	return cloudread.ContextError(ctx, err)
}

// Remove keeps the original collection's name, duplicate and missing policies.
// Capture precedes option preparation and lookup; the outer source invariant
// survives through the binding's owned Delete and the final empty-name result.
func Remove[T any](ctx context.Context, client *gophercloud.ServiceClient, kind string, collection *resource.Collection[T], ref resource.Ref, options ...resource.LookupOption) error {
	if err := cloudread.Context(ctx); err != nil {
		return err
	}
	if kind != "containers" && kind != "orders" && kind != "secrets" {
		return invalid("unsupported key-manager delete resource")
	}
	if collection == nil {
		return invalid("resource collection is required")
	}
	source, err := cloudread.Capture(ctx, client, "key-manager")
	if err != nil {
		return err
	}
	ctx = rest.WithOperationGuard(ctx, source.Guard)
	if err := rest.CheckOperationGuard(ctx); err != nil {
		return err
	}
	err = collection.Delete(ctx, ref, slices.Clone(options)...)
	return cloudread.ContextError(ctx, errors.Join(err, rest.CheckOperationGuard(ctx)))
}
