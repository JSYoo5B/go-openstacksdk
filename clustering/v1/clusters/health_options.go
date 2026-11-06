package clusters

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type CheckOpts struct{}

// RecoverOpts retains omission, explicit null, empty strings and false. Senlin
// defines the recovery operation and the meaning of its parameter object.
type RecoverOpts struct {
	Operation       request.Optional[string] `json:"operation,omitzero"`
	OperationParams json.RawMessage          `json:"operation_params,omitempty"`
	Check           request.Optional[bool]   `json:"check,omitzero"`
	CheckCapacity   request.Optional[bool]   `json:"check_capacity,omitzero"`
}

// PerformOperationOpts contains the published node filter and profile inputs.
// Omission leaves controller defaults; explicit null is passed for the server
// to validate rather than interpreted as an empty object by the SDK.
type PerformOperationOpts struct {
	Filters json.RawMessage `json:"filters,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// CheckOption is one option of Check: WithCheckOptions, the WithCheck field
// options, request.Header, request.Field or request.Custom over
// *request.Config[CheckOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CheckOption = request.WriteCall

// RecoverOption is one option of Recover: WithRecoverOptions, the WithRecover
// field options, request.Header, request.Field or request.Custom over
// *request.Config[RecoverOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecoverOption = request.WriteCall

// PerformOperationOption is one option of PerformOperation:
// WithPerformOperationOptions, the WithPerformOperation field options,
// request.Header, request.Field or request.Custom over
// *request.Config[PerformOperationOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type PerformOperationOption = request.WriteCall

func WithCheckOptions(value CheckOpts) CheckOption { return request.Custom(senlin.Snapshot(value)) }
func WithRecoverOptions(value RecoverOpts) RecoverOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithPerformOperationOptions(value PerformOperationOpts) PerformOperationOption {
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
	return request.Custom(withJSON(value, func(options *RecoverOpts, raw json.RawMessage) { options.OperationParams = raw }))
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
func WithRecoverCheckCapacity(value bool) RecoverOption {
	return request.Custom(func(config *request.Config[RecoverOpts]) error {
		config.Options.CheckCapacity = request.Present(value)
		return nil
	})
}
func WithRecoverCheckCapacityNull() RecoverOption {
	return request.Custom(func(config *request.Config[RecoverOpts]) error {
		config.Options.CheckCapacity = request.Null[bool]()
		return nil
	})
}
func WithPerformOperationFilters(value any) PerformOperationOption {
	return request.Custom(withJSON(value, func(options *PerformOperationOpts, raw json.RawMessage) { options.Filters = raw }))
}
func WithPerformOperationParams(value any) PerformOperationOption {
	return request.Custom(withJSON(value, func(options *PerformOperationOpts, raw json.RawMessage) { options.Params = raw }))
}
