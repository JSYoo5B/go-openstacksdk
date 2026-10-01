package clusters

import (
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
)

// PolicyID can be a controller name, UUID or short ID. Enabled distinguishes
// omission, an explicit false value and null; Senlin decides their semantics.
type AttachPolicyOpts struct {
	PolicyID string                 `json:"policy_id"`
	Enabled  request.Optional[bool] `json:"enabled,omitzero"`
}

type DetachPolicyOpts struct {
	PolicyID string `json:"policy_id"`
}

type UpdatePolicyOpts struct {
	PolicyID string                 `json:"policy_id"`
	Enabled  request.Optional[bool] `json:"enabled,omitzero"`
}

type AttachPolicyOption = request.Option[AttachPolicyOpts]
type DetachPolicyOption = request.Option[DetachPolicyOpts]
type UpdatePolicyOption = request.Option[UpdatePolicyOpts]

func WithAttachPolicyOptions(value AttachPolicyOpts) AttachPolicyOption {
	return senlin.Snapshot(value)
}
func WithDetachPolicyOptions(value DetachPolicyOpts) DetachPolicyOption {
	return senlin.Snapshot(value)
}
func WithUpdatePolicyOptions(value UpdatePolicyOpts) UpdatePolicyOption {
	return senlin.Snapshot(value)
}

func WithAttachPolicyEnabled(value bool) AttachPolicyOption {
	return func(config *request.Config[AttachPolicyOpts]) error {
		config.Options.Enabled = request.Present(value)
		return nil
	}
}

func WithAttachPolicyEnabledNull() AttachPolicyOption {
	return func(config *request.Config[AttachPolicyOpts]) error {
		config.Options.Enabled = request.Null[bool]()
		return nil
	}
}

func WithUpdatePolicyEnabled(value bool) UpdatePolicyOption {
	return func(config *request.Config[UpdatePolicyOpts]) error {
		config.Options.Enabled = request.Present(value)
		return nil
	}
}

func WithUpdatePolicyEnabledNull() UpdatePolicyOption {
	return func(config *request.Config[UpdatePolicyOpts]) error {
		config.Options.Enabled = request.Null[bool]()
		return nil
	}
}

func WithAttachPolicyField(key string, value any) AttachPolicyOption {
	return request.WithField[AttachPolicyOpts](key, value)
}

// WithDetachPolicyField supplies a Go extension beyond Python's fixed body.
func WithDetachPolicyField(key string, value any) DetachPolicyOption {
	return request.WithField[DetachPolicyOpts](key, value)
}

func WithUpdatePolicyField(key string, value any) UpdatePolicyOption {
	return request.WithField[UpdatePolicyOpts](key, value)
}

func WithAttachPolicyHeader(key, value string) AttachPolicyOption {
	return request.WithHeader[AttachPolicyOpts](key, value)
}

func WithDetachPolicyHeader(key, value string) DetachPolicyOption {
	return request.WithHeader[DetachPolicyOpts](key, value)
}

func WithUpdatePolicyHeader(key, value string) UpdatePolicyOption {
	return request.WithHeader[UpdatePolicyOpts](key, value)
}
