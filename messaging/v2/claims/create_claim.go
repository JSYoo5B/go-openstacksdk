package claims

import (
	"context"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/messaging/v2/claims"
)

// ClaimedMessages is the result of CreateClaim. ID is the claim identity taken
// from the response Location header, as openstacksdk's Claim resource does.
// ID and Location are empty when the queue had nothing to claim (204) or when
// the server sent no Location header; Messages is then empty as well.
type ClaimedMessages struct {
	ID       string
	Location string
	Messages []Messages
}

// CreateClaim performs the same POST as Create but also returns the claim
// identity that later Get, Update and Delete calls need. The identity is the
// text after the first "claims/" in the Location header, matching the pinned
// Python Claim._translate_response. A Location without that marker leaves ID
// empty instead of failing. The call is never replayed automatically; if it
// fails after the server accepted it, the claim may exist remotely until its
// TTL expires.
func (a *API) CreateClaim(ctx context.Context, queueName string, opts CreateOpts, options ...CreateOption) (*ClaimedMessages, error) {
	cfg, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("CreateClaim", "claims", err)
	}
	if err = request.ValidateCapabilities(cfg, true, true, false); err != nil {
		return nil, request.Wrap("CreateClaim", "claims", err)
	}
	result := upstream.Create(ctx, a.client, queueName, createOptsBuilder{base: cfg.Options, config: cfg})
	messages, err := result.Extract()
	if err != nil {
		return nil, request.Wrap("CreateClaim", "claims", err)
	}
	claimed := &ClaimedMessages{Messages: messages}
	// A 204 response has no body; Python returns before reading Location.
	if result.Body != nil {
		claimed.Location = result.Header.Get("Location")
		if _, id, ok := strings.Cut(claimed.Location, "claims/"); ok {
			claimed.ID = id
		}
	}
	return claimed, nil
}
