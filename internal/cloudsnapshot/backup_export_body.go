package cloudsnapshot

import (
	"context"
	"errors"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// Redirect hooks can change the physical request after native options checks.
// Keep a body rejection sticky so a retry cannot restore and resend it.
func (p *reader) backupExportExchange(ctx context.Context, target string) (*rest.Response, error) {
	var faults rejectedPollFaults
	guard := func(ctx context.Context) error {
		return errors.Join(p.source.Guard(ctx), faults.error())
	}
	client, err := fixedrequest.NewGuarded(&p.source.Client, http.MethodGet, target, guard)
	if err != nil {
		return nil, err
	}
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedPollTransport(func(req *http.Request) (*http.Response, error) {
		if (req.Body != nil && req.Body != http.NoBody) || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
			faults.add(invalid("backup export request must remain bodyless"))
			if req.Body != nil {
				faults.add(req.Body.Close())
			}
		}
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		return parent.RoundTrip(req)
	})
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, sourceCodes()...)
}
