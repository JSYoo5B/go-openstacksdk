package clusters

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

// AttachPolicyOption is one option of AttachPolicy: WithAttachPolicyOptions,
// the WithAttachPolicy field options, request.Header, request.Field or
// request.Custom over *request.Config[AttachPolicyOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type AttachPolicyOption = request.WriteCall

// DetachPolicyOption is one option of DetachPolicy: WithDetachPolicyOptions,
// the WithDetachPolicy field options, request.Header, request.Field or
// request.Custom over *request.Config[DetachPolicyOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type DetachPolicyOption = request.WriteCall

// UpdatePolicyOption is one option of UpdatePolicy: WithUpdatePolicyOptions,
// the WithUpdatePolicy field options, request.Header, request.Field or
// request.Custom over *request.Config[UpdatePolicyOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type UpdatePolicyOption = request.WriteCall

func WithAttachPolicyOptions(value AttachPolicyOpts) AttachPolicyOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithDetachPolicyOptions(value DetachPolicyOpts) DetachPolicyOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithUpdatePolicyOptions(value UpdatePolicyOpts) UpdatePolicyOption {
	return request.Custom(senlin.Snapshot(value))
}

func WithAttachPolicyEnabled(value bool) AttachPolicyOption {
	return request.Custom(func(config *request.Config[AttachPolicyOpts]) error {
		config.Options.Enabled = request.Present(value)
		return nil
	})
}

func WithAttachPolicyEnabledNull() AttachPolicyOption {
	return request.Custom(func(config *request.Config[AttachPolicyOpts]) error {
		config.Options.Enabled = request.Null[bool]()
		return nil
	})
}

func WithUpdatePolicyEnabled(value bool) UpdatePolicyOption {
	return request.Custom(func(config *request.Config[UpdatePolicyOpts]) error {
		config.Options.Enabled = request.Present(value)
		return nil
	})
}

func WithUpdatePolicyEnabledNull() UpdatePolicyOption {
	return request.Custom(func(config *request.Config[UpdatePolicyOpts]) error {
		config.Options.Enabled = request.Null[bool]()
		return nil
	})
}
