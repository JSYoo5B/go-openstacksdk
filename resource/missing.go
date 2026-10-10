package resource

import (
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
)

// IgnoreMissing returns nil when err reports HTTP 404 and err otherwise. It
// gives a direct API delete the openstacksdk Proxy default ignore_missing=True;
// Collection.Delete already applies it. Transport, decode, context and joined
// failures are returned even when they carry a 404, as Collection.Delete does.
func IgnoreMissing(err error) error {
	if err != nil && !terminalDeleteError(err) && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil
	}
	return err
}
