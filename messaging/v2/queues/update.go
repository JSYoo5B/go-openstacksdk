package queues

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/messaging/v2/queues"
)

// updateQueue serializes the builder through ToQueueUpdateMap. The pinned
// native Update passes the builder value itself to the request body, which
// reflects over it and loses the patch list when the value is an SDK wrapper.
// sdkgen audits that native declaration before emitting this call; the URL,
// content type, accepted status codes, and result type stay native.
func updateQueue(ctx context.Context, client *gophercloud.ServiceClient, queueName string, opts upstream.UpdateOptsBuilder) (r upstream.UpdateResult) {
	body, err := opts.ToQueueUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	resp, err := client.Patch(ctx, client.ServiceURL("v2", "queues", queueName), body, &r.Body, &gophercloud.RequestOpts{
		OkCodes: []int{200, 201, 204},
		MoreHeaders: map[string]string{
			"Content-Type": "application/openstack-messaging-v2.0-json-patch"},
	})
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
