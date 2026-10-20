package tokens

import (
	"encoding/json"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
)

// Authentication contains all views of a Keystone v2 access response. Body
// preserves fields not represented by the upstream models, including extensions.
type Authentication struct {
	Token   Token
	User    User
	Catalog ServiceCatalog
	Header  http.Header
	Body    json.RawMessage
}

func extractAuthentication(result gophercloud.Result) (*Authentication, error) {
	created := upstream.CreateResult{Result: result}
	token, err := created.ExtractToken()
	if err != nil {
		return nil, err
	}
	catalog, err := created.ExtractServiceCatalog()
	if err != nil {
		return nil, err
	}
	user, err := (upstream.GetResult{CreateResult: created}).ExtractUser()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(result.Body)
	if err != nil {
		return nil, err
	}
	return &Authentication{
		Token: *token, User: *user, Catalog: *catalog,
		Header: result.Header.Clone(), Body: body,
	}, nil
}
