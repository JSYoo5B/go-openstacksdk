package compute

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// The cloud convenience creates a name and only a truthy public key. The
// versioned CreateKeypair remains available for nullable full-body attributes.
type CloudKeypairCreateOpts struct{ PublicKey string }

// CloudKeypairCreateOption is one option of CreateKeypair:
// WithCloudKeypairPublicKey, request.Header or request.Custom over
// *request.Config[CloudKeypairCreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CloudKeypairCreateOption = request.WriteCall

var cloudKeypairCreateCalls = calls.Spec[request.Config[CloudKeypairCreateOpts]]{Header: calls.ConfigHeader[CloudKeypairCreateOpts]}

func WithCloudKeypairPublicKey(value string) CloudKeypairCreateOption {
	return request.Custom(func(c *request.Config[CloudKeypairCreateOpts]) error { c.Options.PublicKey = value; return nil })
}
func (s *Service) CreateKeypair(ctx context.Context, name string, options ...CloudKeypairCreateOption) (*keypairs.KeypairRecord, error) {
	p, err := s.captureKeypairs(ctx)
	if err != nil {
		return nil, keypairCloudFailure("CreateKeypair", err)
	}
	if err := p.captureLocation(); err != nil {
		return nil, keypairCloudFailure("CreateKeypair", err)
	}
	translated := calls.Options(options, cloudKeypairCreateCalls)
	guarded := make([]request.Option[CloudKeypairCreateOpts], len(translated))
	for i, option := range translated {
		apply := option
		guarded[i] = func(c *request.Config[CloudKeypairCreateOpts]) error {
			if err := p.check(p.ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil cloud keypair create option")
			}
			c.Headers = maps.Clone(c.Headers)
			if c.Headers == nil {
				c.Headers = map[string]string{}
			}
			err := apply(c)
			c.Headers = maps.Clone(c.Headers)
			return errors.Join(err, p.check(p.ctx))
		}
	}
	c, err := request.Apply(CloudKeypairCreateOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(c, false, false, true)
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, keypairCloudFailure("CreateKeypair", err)
	}
	leaf := []keypairs.KeypairCreateOption{keypairs.WithKeypairCreateName(name)}
	if c.Options.PublicKey != "" {
		leaf = append(leaf, keypairs.WithKeypairCreatePublicKey(c.Options.PublicKey))
	}
	for key, value := range c.Headers {
		leaf = append(leaf, request.Header(key, value))
	}
	result, err := p.api.CreateKeypair(p.ctx, leaf...)
	p.locate(result)
	return result, keypairCloudFailure("CreateKeypair", errors.Join(err, p.check(p.ctx)))
}

// DeleteKeypair provides cloud's true-on-success/false-on-clean-missing result
// without resolving, waiting, or hiding accepted-response/source failures.
func (s *Service) DeleteKeypair(ctx context.Context, name string) (bool, error) {
	p, err := s.captureKeypairs(ctx)
	if err != nil {
		return false, keypairCloudFailure("DeleteKeypair", err)
	}
	err = p.api.DeleteKeypair(p.ctx, name, request.IgnoreMissing(false))
	err = errors.Join(err, p.check(p.ctx))
	if err == nil {
		return true, nil
	}
	var missing *resource.NotFoundError
	native := queryHTTPFailure(err)
	expected := p.api.RawClient().ServiceURL("os-keypairs", url.PathEscape(name))
	if errors.As(err, &missing) && queryNotFound(err) && native != nil && native.Method == http.MethodDelete && native.URL == expected {
		return false, nil
	}
	return false, keypairCloudFailure("DeleteKeypair", err)
}
