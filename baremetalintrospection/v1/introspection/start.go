package introspection

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/baremetalintrospection/v1/introspection"
)

// startIntrospection preserves the serialized query discarded by the pinned
// native StartIntrospection. sdkgen audits that native declaration before
// emitting this call; the native result and HTTP status policy stay intact.
func startIntrospection(ctx context.Context, client *gophercloud.ServiceClient, nodeID string, opts upstream.StartOptsBuilder) (r upstream.StartResult) {
	query, err := opts.ToStartIntrospectionQuery()
	if err != nil {
		r.Err = err
		return
	}
	resp, err := client.Post(ctx, client.ServiceURL("introspection", nodeID)+query, nil, nil, &gophercloud.RequestOpts{
		OkCodes: []int{202},
	})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
