package nativefind

import (
	"context"
	"fmt"
	"iter"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	volumes "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	servers "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/internal/query"
	"gophercloudsdk/resource"
)

// IterateServers selects Nova's native detailed or summary pager for identity
// fallback. It owns query data and retains the original service/provider client,
// native page extraction and resource.Stream cancellation/continuation policy.
func IterateServers(ctx context.Context, client *gophercloud.ServiceClient, values url.Values, details bool) iter.Seq2[*servers.Server, error] {
	frozen := cloneListQuery(values)
	return func(yield func(*servers.Server, error) bool) {
		parts := []string{"servers"}
		if details {
			parts = append(parts, "detail")
		}
		if _, err := nativeListURL(ctx, client, parts); err != nil {
			yield(nil, err)
			return
		}
		var pager pagination.Pager
		if details {
			pager = servers.List(client, query.Adapter(frozen))
		} else {
			pager = servers.ListSimple(client, query.Adapter(frozen))
		}
		resource.Stream(ctx, pager, servers.ExtractServers)(yield)
	}
}

// IterateVolumes selects Cinder's native detail pager or the /volumes summary
// route with the same native VolumePage/ExtractVolumes contract. ServiceURL
// honors ResourceBase, and no service/provider setting is changed.
func IterateVolumes(ctx context.Context, client *gophercloud.ServiceClient, values url.Values, details bool) iter.Seq2[*volumes.Volume, error] {
	frozen := cloneListQuery(values)
	return func(yield func(*volumes.Volume, error) bool) {
		parts := []string{"volumes"}
		if details {
			parts = append(parts, "detail")
		}
		target, err := nativeListURL(ctx, client, parts)
		if err != nil {
			yield(nil, err)
			return
		}
		var pager pagination.Pager
		if details {
			pager = volumes.List(client, query.Adapter(frozen))
		} else {
			encoded, _ := query.Adapter(frozen).ToVolumeListQuery()
			pager = pagination.NewPager(client, target+encoded, func(result pagination.PageResult) pagination.Page {
				return volumes.VolumePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: result}}
			})
		}
		resource.Stream(ctx, pager, volumes.ExtractVolumes)(yield)
	}
}

func cloneListQuery(values url.Values) url.Values {
	frozen := make(url.Values, len(values))
	for key, entries := range values {
		frozen[key] = append([]string(nil), entries...)
	}
	return frozen
}

func nativeListURL(ctx context.Context, client *gophercloud.ServiceClient, parts []string) (string, error) {
	invalid := func(message string) (string, error) {
		return "", fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
	}
	if ctx == nil {
		return invalid("native identity listing requires a context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if client == nil || client.ProviderClient == nil {
		return invalid("native identity listing requires a service and provider client")
	}
	target := client.ServiceURL(parts...)
	parsed, err := url.Parse(target)
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return invalid("native identity listing requires an absolute service URL without credentials, query or fragment")
	}
	return target, nil
}
