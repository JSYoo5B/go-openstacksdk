package nodes

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CheckOpts has no required parameters. Extensions are sent inside check.
type CheckOpts struct{}

// RecoverOpts leaves unspecified defaults to the server. Operation retains
// absence, an empty string, or explicit null. OperationParams retains absence,
// an object, or explicit null; the service validates operation values and null.
type RecoverOpts struct {
	Operation       request.Optional[string] `json:"operation,omitzero"`
	OperationParams json.RawMessage          `json:"operation_params,omitempty"`
	Check           request.Optional[bool]   `json:"check,omitzero"`
}

// PerformOperationOpts has no universal parameters. The profile type defines
// its operation schema, supplied through request.Field.
type PerformOperationOpts struct{}

// CheckOption is one option of Check: request.Header, request.Field (sent
// inside check) or request.Custom over *request.Config[CheckOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type CheckOption = request.WriteCall

var checkCalls = senlin.WriteCalls[CheckOpts]()

// RecoverOption is one option of Recover: WithRecoverOptions, the WithRecover
// field options, request.Header, request.Field or request.Custom over
// *request.Config[RecoverOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecoverOption = request.WriteCall

var recoverCalls = senlin.WriteCalls[RecoverOpts]()

// PerformOperationOption is one option of PerformOperation: request.Header,
// request.Field (an operation parameter) or request.Custom over
// *request.Config[PerformOperationOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type PerformOperationOption = request.WriteCall

var performOperationCalls = senlin.WriteCalls[PerformOperationOpts]()

func WithRecoverOptions(value RecoverOpts) RecoverOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithRecoverOperation(value string) RecoverOption {
	return request.Custom(func(config *request.Config[RecoverOpts]) error {
		config.Options.Operation = request.Present(value)
		return nil
	})
}
func WithRecoverOperationNull() RecoverOption {
	return request.Custom(func(config *request.Config[RecoverOpts]) error {
		config.Options.Operation = request.Null[string]()
		return nil
	})
}
func WithRecoverOperationParams(value any) RecoverOption {
	return request.Custom(withJSON(value, func(opts *RecoverOpts, raw json.RawMessage) { opts.OperationParams = raw }))
}
func WithRecoverCheck(value bool) RecoverOption {
	return request.Custom(func(config *request.Config[RecoverOpts]) error {
		config.Options.Check = request.Present(value)
		return nil
	})
}
func WithRecoverCheckNull() RecoverOption {
	return request.Custom(func(config *request.Config[RecoverOpts]) error {
		config.Options.Check = request.Null[bool]()
		return nil
	})
}
