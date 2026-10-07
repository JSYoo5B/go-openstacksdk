package senlin

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CommandActionID validates the published command response: a top-level action
// string and one Location at the selected action collection must agree. It does
// not fetch the action, invent status, or resend an accepted mutation.
func CommandActionID(client *gophercloud.ServiceClient, response *rest.Response) (string, error) {
	object, err := response.Object("")
	if err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(object, &fields); err != nil {
		return "", response.Fail(err)
	}
	var identity string
	if err := json.Unmarshal(fields["action"], &identity); err != nil {
		return "", response.Fail(fmt.Errorf("%w: Senlin command response requires an action string: %v", resource.ErrInvalidOption, err))
	}
	if err := Identifier(identity); err != nil {
		return "", response.Fail(err)
	}
	locationID, err := ActionID(client, response)
	if err != nil {
		return "", err
	}
	if locationID != identity {
		return "", response.Fail(fmt.Errorf("%w: Senlin command body and Location identify different actions", resource.ErrInvalidOption))
	}
	return identity, nil
}
