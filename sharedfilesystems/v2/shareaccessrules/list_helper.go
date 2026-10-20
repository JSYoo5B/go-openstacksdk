package shareaccessrules

import (
	"context"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/sharedfilesystems/v2/shareaccessrules"
)

// listAccessRules issues the same GET as the pinned native List but escapes the
// share ID as a query value. The native URL builder concatenates the raw ID
// after "?share_id=", so an ID containing "&" or "#" would inject or truncate
// query parameters. sdkgen audits that native declaration before emitting this
// call; the 200-only success handling and result type stay native.
func listAccessRules(ctx context.Context, client *gophercloud.ServiceClient, shareID string) (r upstream.ListResult) {
	resp, err := client.Get(ctx, client.ServiceURL("share-access-rules")+"?share_id="+url.QueryEscape(shareID), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
