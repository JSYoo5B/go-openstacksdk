package cloudsnapshot

import (
	"context"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Gophercloud discards Read/Close errors on unexpected status responses.
// A compatible status may enable name lookup, so only a fault-free native
// rejection may establish that the member route is unavailable. Observe the
// native body without accepting the status or fabricating an admitted page.
func (p *reader) memberGet(ctx context.Context, target string) (*rest.Response, error) {
	client, err := fixedrequest.NewGuarded(&p.source.Client, http.MethodGet, target, p.source.Guard)
	if err != nil {
		return nil, err
	}
	var faults rejectedPollFaults
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedPollTransport(func(req *http.Request) (*http.Response, error) {
		response, err := parent.RoundTrip(req)
		if response != nil && response.Body != nil {
			switch response.StatusCode {
			case http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound:
				response.Body = &rejectedPollBody{ReadCloser: response.Body, faults: &faults}
			}
		}
		return response, err
	})
	guard := func(ctx context.Context) error {
		return errors.Join(p.source.Guard(ctx), faults.error())
	}
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, sourceCodes()...)
}
