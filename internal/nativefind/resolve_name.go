package nativefind

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ResolveName is the opt-in creation resolver. Each page/retry is guarded;
// ordinary native lookup keeps its existing transport and extraction policy.
func ResolveName[T any](ctx context.Context, client *gophercloud.ServiceClient, path, kind, plural string, ref resource.Ref, id, name func(*T) string, queryName bool, source func(context.Context) error) (string, error) {
	if client == nil || client.ProviderClient == nil {
		return "", fmt.Errorf("%w: resolver client is required", resource.ErrInvalidOption)
	}
	provider, endpoint, base, version, serviceType := client.ProviderClient, client.Endpoint, client.ResourceBase, client.Microversion, client.Type
	var changed error
	var mu sync.Mutex
	sourceOnly := func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if changed == nil && (client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || client.Microversion != version || client.Type != serviceType) {
			changed = fmt.Errorf("%w: %s resolver source changed", resource.ErrInvalidOption, kind)
		}
		var extra error
		if source != nil {
			extra = source(ctx)
		}
		return errors.Join(changed, extra, ctx.Err(), context.Cause(ctx))
	}
	rest.RegisterOperationSource(ctx, sourceOnly)
	guard := func(ctx context.Context) error { return errors.Join(sourceOnly(ctx), rest.CheckOperationGuard(ctx)) }
	spec := rest.CollectionSpec[T]{Client: client, Path: path, Kind: kind, PluralKey: plural, ID: id, Name: name,
		Metadata: func(*T) *resource.Metadata { return &resource.Metadata{} }, Validate: guard, SourceGuard: guard,
		ListCodes: []int{200, 204, 300}}
	if queryName {
		spec.NameQuery = func(name string) string { return name }
	}
	value, err := rest.Collection(spec).Find(ctx, ref)
	if err != nil {
		return "", errors.Join(err, guard(ctx))
	}
	resolved := id(value)
	if err := resource.ID(resolved).Validate(); err != nil {
		return "", err
	}
	return resolved, guard(ctx)
}
