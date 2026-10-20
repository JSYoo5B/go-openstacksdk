package quotasets

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/gophercloud/gophercloud/v2"
)

// User quota requests retain their complete target across redirects and retries.
func guardedQuotaClient(source *gophercloud.ServiceClient, method, target string) (*gophercloud.ServiceClient, error) {
	return fixedrequest.New(source, method, target)
}
