package quotasets

import (
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
)

// User quota requests retain their complete target across redirects and retries.
func guardedQuotaClient(source *gophercloud.ServiceClient, method, target string) (*gophercloud.ServiceClient, error) {
	return fixedrequest.New(source, method, target)
}
