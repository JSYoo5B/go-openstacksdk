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

type CheckOption = request.Option[CheckOpts]
type RecoverOption = request.Option[RecoverOpts]
type PerformOperationOption = request.Option[PerformOperationOpts]

func WithCheckOptions(value CheckOpts) CheckOption       { return senlin.Snapshot(value) }
func WithRecoverOptions(value RecoverOpts) RecoverOption { return senlin.Snapshot(value) }
func WithPerformOperationOptions(value PerformOperationOpts) PerformOperationOption {
	return senlin.Snapshot(value)
}
func WithRecoverOperation(value string) RecoverOption {
	return func(config *request.Config[RecoverOpts]) error {
		config.Options.Operation = request.Present(value)
		return nil
	}
}
func WithRecoverOperationNull() RecoverOption {
	return func(config *request.Config[RecoverOpts]) error {
		config.Options.Operation = request.Null[string]()
		return nil
	}
}
func WithRecoverOperationParams(value any) RecoverOption {
	return withJSON(value, func(options *RecoverOpts, raw json.RawMessage) { options.OperationParams = raw })
}
func WithRecoverCheck(value bool) RecoverOption {
	return func(config *request.Config[RecoverOpts]) error {
		config.Options.Check = request.Present(value)
		return nil
	}
}
func WithRecoverCheckNull() RecoverOption {
	return func(config *request.Config[RecoverOpts]) error {
		config.Options.Check = request.Null[bool]()
		return nil
	}
}
func WithRecoverCheckCapacity(value bool) RecoverOption {
	return func(config *request.Config[RecoverOpts]) error {
		config.Options.CheckCapacity = request.Present(value)
		return nil
	}
}
func WithRecoverCheckCapacityNull() RecoverOption {
	return func(config *request.Config[RecoverOpts]) error {
		config.Options.CheckCapacity = request.Null[bool]()
		return nil
	}
}
func WithPerformOperationFilters(value any) PerformOperationOption {
	return withJSON(value, func(options *PerformOperationOpts, raw json.RawMessage) { options.Filters = raw })
}
func WithPerformOperationParams(value any) PerformOperationOption {
	return withJSON(value, func(options *PerformOperationOpts, raw json.RawMessage) { options.Params = raw })
}
func WithCheckField(key string, value any) CheckOption {
	return request.WithField[CheckOpts](key, value)
}
func WithRecoverField(key string, value any) RecoverOption {
	return request.WithField[RecoverOpts](key, value)
}
func WithPerformOperationField(key string, value any) PerformOperationOption {
	return request.WithField[PerformOperationOpts](key, value)
}
func WithCheckHeader(key, value string) CheckOption { return request.WithHeader[CheckOpts](key, value) }
func WithRecoverHeader(key, value string) RecoverOption {
	return request.WithHeader[RecoverOpts](key, value)
}
func WithPerformOperationHeader(key, value string) PerformOperationOption {
	return request.WithHeader[PerformOperationOpts](key, value)
}
