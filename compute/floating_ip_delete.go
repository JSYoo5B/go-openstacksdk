package compute

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Cloud raises on HTTP>=400; successful deletion responses are passive proof,
// not a JSON resource. Fix that initial status policy through native retries.
var floatingIPDeleteCodes = func() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}()

// DeleteFloatingIP defaults to one extra logical delete with public-Get
// verification after each success. DELETE404 returns false without fallback.
func (s *Service) DeleteFloatingIP(ctx context.Context, input DeleteFloatingIPRequest, options ...FloatingIPDeleteOption) (*DeleteFloatingIPResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := resource.ID(input.ID).Validate(); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || s.Servers.Collection == nil {
		return nil, invalid("floating IP delete facade is required")
	}
	base := s.captureAutomaticComputeSource()
	policy, err := prepareFloatingIPDeleteOptions(options, func() error { return base(ctx) })
	if err != nil {
		return nil, err
	}
	p, err := s.captureFloatingIPQuery(ctx, []FloatingIPQueryOption{WithFloatingIPQueryOptions(policy.query())})
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	return p.deleteWithRetries(input.ID, policy.Retries)
}
func (p *floatingIPQueryState) deleteWithRetries(id string, retry int) (*DeleteFloatingIPResult, error) {
	result := &DeleteFloatingIPResult{ID: id}
	retries := max(0, retry)
	for count := 0; ; count++ {
		attempt, err := p.delete(id)
		if attempt != nil {
			result.Attempts = append(result.Attempts, attempt)
			result.Backend = attempt.Backend
			result.Failure = attempt.Failure
		}
		if err != nil {
			return result, errors.Join(err, p.state.check(p.ctx))
		}
		if attempt.NotFound != nil {
			return result, p.state.check(p.ctx)
		}
		if retry == 0 {
			if err := p.state.check(p.ctx); err != nil {
				return result, err
			}
			result.Deleted = true
			return result, nil
		}
		verification, err := p.get(id)
		attempt.Verification = verification
		result.LastVerification = verification
		if err != nil {
			if verification != nil {
				attempt.Failure = verification.Failure
				result.Failure = verification.Failure
			}
			return result, errors.Join(err, p.state.check(p.ctx))
		}
		attempt.Verified = true
		attempt.Absent = verification == nil || verification.Value == nil
		if verification != nil && verification.FloatingIP != nil && verification.FloatingIP.Resource != nil {
			var status string
			if json.Unmarshal(verification.FloatingIP.Resource.Body["status"], &status) == nil {
				attempt.Down = status == "DOWN"
			}
		}
		if attempt.Absent || attempt.Down {
			if err := p.state.check(p.ctx); err != nil {
				return result, err
			}
			result.Deleted = true
			result.Absent = attempt.Absent
			result.Down = attempt.Down
			return result, nil
		}
		if count == retries {
			return result, errors.Join(&FloatingIPDeleteVerificationError{ID: id, Attempts: len(result.Attempts), FloatingIP: verification.FloatingIP}, p.state.check(p.ctx))
		}
	}
}
func (p *floatingIPQueryState) delete(id string) (*FloatingIPDeleteAttempt, error) {
	backend, client, err := p.backend()
	attempt := &FloatingIPDeleteAttempt{Backend: backend}
	if err != nil {
		return attempt, err
	}
	path := "floatingips"
	if backend == FloatingIPNova {
		client, err = p.novaClient()
		path = "os-floating-ips"
		if err != nil {
			return attempt, err
		}
	}
	response, err := rest.DoJSONGuarded(p.ctx, client, p.state.check, http.MethodDelete, client.ServiceURL(path, url.PathEscape(id)), nil, nil, floatingIPDeleteCodes...)
	attempt.Response = queryResponse(backend, response)
	attempt.Accepted = response != nil
	err = errors.Join(err, p.state.check(p.ctx))
	attempt.Failure = queryFailure(backend, err)
	if queryNotFound(err) {
		attempt.NotFound = err
		return attempt, nil
	}
	return attempt, err
}
