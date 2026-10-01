package nodes

import (
	"encoding/json"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
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
// its operation schema, supplied through WithPerformOperationField.
type PerformOperationOpts struct{}

type CheckOption = request.Option[CheckOpts]
type RecoverOption = request.Option[RecoverOpts]
type PerformOperationOption = request.Option[PerformOperationOpts]

func WithCheckField(key string, value any) CheckOption {
	return request.WithField[CheckOpts](key, value)
}
func WithCheckHeader(key, value string) CheckOption {
	return request.WithHeader[CheckOpts](key, value)
}
func WithRecoverOptions(value RecoverOpts) RecoverOption { return senlin.Snapshot(value) }
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
	return withJSON(value, func(opts *RecoverOpts, raw json.RawMessage) { opts.OperationParams = raw })
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
func WithRecoverField(key string, value any) RecoverOption {
	return request.WithField[RecoverOpts](key, value)
}
func WithRecoverHeader(key, value string) RecoverOption {
	return request.WithHeader[RecoverOpts](key, value)
}
func WithPerformOperationField(key string, value any) PerformOperationOption {
	return request.WithField[PerformOperationOpts](key, value)
}
func WithPerformOperationHeader(key, value string) PerformOperationOption {
	return request.WithHeader[PerformOperationOpts](key, value)
}
