package compute

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DetachFloatingIPRequest names the Nova server and the floating IP to detach.
// FloatingIPID is the identifier used for the lookup and, on Neutron, the PUT.
// The Neutron branch never uses ServerID except in error context; like the
// pinned helper it does not check that the IP is attached to that server.
type DetachFloatingIPRequest struct {
	ServerID     string
	FloatingIPID string
}

// DetachFloatingIPOutcome states what the helper established.
type DetachFloatingIPOutcome string

const (
	// DetachedNeutron means PUT floatingips/{id} {"floatingip":{"port_id":null}}
	// was accepted. This is acknowledgement, not proof of a later state.
	DetachedNeutron DetachFloatingIPOutcome = "detached"
	// DetachNotAttached means the Neutron lookup found no row or a row without
	// a port_id, so nothing was sent.
	DetachNotAttached DetachFloatingIPOutcome = "not_attached"
	// DetachNovaActionAccepted means the legacy removeFloatingIp server action
	// was accepted. The pinned helper returns None for this branch.
	DetachNovaActionAccepted DetachFloatingIPOutcome = "nova_action_accepted"
)

// DetachFloatingIPResult keeps lookup and mutation evidence separate. Detached
// is the pinned Neutron boolean: true only for DetachedNeutron. It stays false
// for the Nova branch, whose Python result is None; use Outcome to tell them
// apart.
//
// Backend is the branch the helper ran: FloatingIPNeutron whenever the
// Neutron floating source is in use (Python _use_neutron_floating), even if
// the lookup itself fell back to Nova rows (Lookup.Backend and
// Lookup.FallbackError show that). NeutronLookup and NeutronNotFound are set
// only when the Neutron branch's lookup failed with a clean 404 and the helper
// ran the Nova branch, which repeats the lookup into Lookup.
type DetachFloatingIPResult struct {
	Backend         FloatingIPSource
	Outcome         DetachFloatingIPOutcome
	Detached        bool
	Lookup          *GetFloatingIPResult
	NeutronLookup   *GetFloatingIPResult
	NeutronNotFound error
	Response        *FloatingIPQueryResponse
	Failure         *FloatingIPQueryResponse
}

// DetachFloatingIPError reports a failed mutation after a successful lookup.
// It wraps the transport error, as the pinned helper wraps its Neutron failure
// in an SDKException naming the IP and server.
type DetachFloatingIPError struct {
	ServerID, FloatingIPID string
	Err                    error
}

func (e *DetachFloatingIPError) Error() string {
	return "error detaching IP " + e.FloatingIPID + " from server " + e.ServerID + ": " + e.Err.Error()
}
func (e *DetachFloatingIPError) Unwrap() error { return e.Err }

// DetachIPFromServer implements NetworkCommonCloudMixin.detach_ip_from_server.
//
// With the Neutron backend it looks the floating IP up through the cloud
// get_floating_ip search (query options apply), returns DetachNotAttached when
// the row is absent or has no port_id, and otherwise sends one PUT that nulls
// port_id. Any failure of that PUT, including 404, is reported as
// *DetachFloatingIPError and never falls back to Nova. With the legacy Nova
// backend it looks the IP up, fails when it is missing, and sends one
// removeFloatingIp server action with the row's address.
//
// A mutation is sent at most once by this helper: it is never replayed after
// an uncertain result, and cancelling ctx does not undo an accepted request.
func (s *Service) DetachIPFromServer(ctx context.Context, input DetachFloatingIPRequest, options ...FloatingIPQueryOption) (*DetachFloatingIPResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := resource.ID(input.ServerID).Validate(); err != nil {
		return nil, err
	}
	if err := resource.ID(input.FloatingIPID).Validate(); err != nil {
		return nil, err
	}
	p, err := s.captureFloatingIPQuery(ctx, options)
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	result := &DetachFloatingIPResult{}
	lookup, err := p.get(input.FloatingIPID)
	result.Lookup = lookup
	if lookup != nil {
		result.Backend, result.Failure = lookup.Backend, lookup.Failure
	}
	// p.get selected the backend: neutronMode is Python _use_neutron_floating.
	if p.neutronMode {
		result.Backend = FloatingIPNeutron
		if err == nil {
			// The Neutron branch also runs on Nova rows that the lookup
			// normalized after a Neutron list 404.
			return p.detachNeutron(input, result)
		}
		if p.ctx.Err() != nil || !queryNotFound(err) {
			return result, err
		}
		// Python catches NotFoundException from its Neutron branch and runs
		// the Nova branch, whose own get_floating_ip repeats the lookup.
		result.NeutronLookup, result.NeutronNotFound = lookup, err
		lookup, err = p.get(input.FloatingIPID)
		result.Lookup, result.Backend, result.Failure = lookup, FloatingIPNova, nil
		if lookup != nil {
			result.Failure = lookup.Failure
		}
	}
	if err != nil {
		return result, err
	}
	if result.Backend == FloatingIPNeutron {
		return p.detachNeutron(input, result)
	}
	return p.detachNova(input, result)
}

func floatingIPRowString(record *FloatingIPRecord, key string) (value string, present bool) {
	if record == nil || record.Resource == nil {
		return "", false
	}
	raw, ok := record.Resource.Body[key]
	if !ok {
		return "", false
	}
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func (p *floatingIPQueryState) detachNeutron(input DetachFloatingIPRequest, result *DetachFloatingIPResult) (*DetachFloatingIPResult, error) {
	record := result.Lookup.FloatingIP
	if record == nil {
		result.Outcome = DetachNotAttached
		return result, p.state.check(p.ctx)
	}
	raw, present := json.RawMessage(nil), false
	if record.Resource != nil {
		raw, present = record.Resource.Body["port_id"]
	}
	if !present {
		// A strict-mode normalized Nova row has no port_id; Python's
		// attribute access on that Munch raises instead of reading None.
		return result, invalid("floating IP %q row has no port_id attribute", input.FloatingIPID)
	}
	attached, err := cloudfilter.PythonTruthy(raw)
	if err != nil {
		return result, err
	}
	if !attached {
		result.Outcome = DetachNotAttached
		return result, p.state.check(p.ctx)
	}
	_, client, err := p.backend()
	if err != nil {
		return result, err
	}
	body := map[string]any{"floatingip": map[string]any{"port_id": nil}}
	response, err := rest.DoJSONGuarded(p.ctx, client, p.state.check, http.MethodPut, client.ServiceURL("floatingips", url.PathEscape(input.FloatingIPID)), body, nil, floatingIPDeleteCodes...)
	result.Response = queryResponse(FloatingIPNeutron, response)
	err = errors.Join(err, p.state.check(p.ctx))
	if err != nil {
		result.Failure = queryFailure(FloatingIPNeutron, err)
		return result, &DetachFloatingIPError{ServerID: input.ServerID, FloatingIPID: input.FloatingIPID, Err: err}
	}
	result.Outcome, result.Detached = DetachedNeutron, true
	return result, nil
}

func (p *floatingIPQueryState) detachNova(input DetachFloatingIPRequest, result *DetachFloatingIPResult) (*DetachFloatingIPResult, error) {
	if result.Lookup == nil || result.Lookup.FloatingIP == nil {
		return result, invalid("unable to find floating IP %q", input.FloatingIPID)
	}
	address, ok := floatingIPRowString(result.Lookup.FloatingIP, "floating_ip_address")
	if !ok || address == "" {
		return result, invalid("floating IP %q has no floating_ip_address", input.FloatingIPID)
	}
	client, err := p.novaClient()
	if err != nil {
		return result, err
	}
	body := map[string]any{"removeFloatingIp": map[string]any{"address": address}}
	response, err := rest.DoJSONGuarded(p.ctx, client, p.state.check, http.MethodPost, client.ServiceURL("servers", url.PathEscape(input.ServerID), "action"), body, nil, floatingIPDeleteCodes...)
	result.Response = queryResponse(FloatingIPNova, response)
	err = errors.Join(err, p.state.check(p.ctx))
	if err != nil {
		result.Failure = queryFailure(FloatingIPNova, err)
		return result, &DetachFloatingIPError{ServerID: input.ServerID, FloatingIPID: input.FloatingIPID, Err: err}
	}
	result.Outcome = DetachNovaActionAccepted
	return result, nil
}
