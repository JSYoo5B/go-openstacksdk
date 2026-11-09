package clusters

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ScaleInOpts and ScaleOutOpts preserve Python's default count:null request.
// Senlin decides how a count combines with attached scaling policies.
type ScaleInOpts struct {
	Count request.Optional[int] `json:"count"`
}
type ScaleOutOpts struct {
	Count request.Optional[int] `json:"count"`
}

type AdjustmentType string

const (
	ExactCapacity      AdjustmentType = "EXACT_CAPACITY"
	ChangeInCapacity   AdjustmentType = "CHANGE_IN_CAPACITY"
	ChangeInPercentage AdjustmentType = "CHANGE_IN_PERCENTAGE"
)

// ResizeOpts describes the published resize fields. Number retains decimal
// precision; omission, explicit null, zero and false remain distinct. Senlin
// validates adjustment combinations and cluster size constraints.
type ResizeOpts struct {
	AdjustmentType request.Optional[AdjustmentType] `json:"adjustment_type,omitzero"`
	Number         request.Optional[json.Number]    `json:"number,omitzero"`
	MinSize        request.Optional[int]            `json:"min_size,omitzero"`
	MaxSize        request.Optional[int]            `json:"max_size,omitzero"`
	MinStep        request.Optional[int]            `json:"min_step,omitzero"`
	Strict         request.Optional[bool]           `json:"strict,omitzero"`
}

type AddNodesOpts struct {
	Nodes []string `json:"nodes"`
}

// Any explicit DestroyAfterDeletion, including false or null, requires 1.4.
type RemoveNodesOpts struct {
	Nodes                []string               `json:"nodes"`
	DestroyAfterDeletion request.Optional[bool] `json:"destroy_after_deletion,omitzero"`
}

// Nodes maps each member identity to the replacement identity. Identities can
// be controller names, UUIDs or short IDs; the SDK does not resolve each node.
type ReplaceNodesOpts struct {
	Nodes map[string]string `json:"nodes"`
}

// ScaleInOption is one option of ScaleIn: WithScaleInOptions, the WithScaleIn
// field options, request.Header, request.Field or request.Custom over
// *request.Config[ScaleInOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ScaleInOption = request.WriteCall

// ScaleOutOption is one option of ScaleOut: WithScaleOutOptions, the
// WithScaleOut field options, request.Header, request.Field or request.Custom
// over *request.Config[ScaleOutOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ScaleOutOption = request.WriteCall

// ResizeOption is one option of Resize: WithResizeOptions, the WithResize
// field options, request.Header, request.Field or request.Custom over
// *request.Config[ResizeOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ResizeOption = request.WriteCall

// AddNodesOption is one option of AddNodes: WithAddNodesOptions, the
// WithAddNodes field options, request.Header, request.Field or request.Custom
// over *request.Config[AddNodesOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type AddNodesOption = request.WriteCall

// RemoveNodesOption is one option of RemoveNodes: WithRemoveNodesOptions, the
// WithRemoveNodes field options, request.Header, request.Field or
// request.Custom over *request.Config[RemoveNodesOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type RemoveNodesOption = request.WriteCall

// ReplaceNodesOption is one option of ReplaceNodes: WithReplaceNodesOptions,
// the WithReplaceNodes field options, request.Header, request.Field or
// request.Custom over *request.Config[ReplaceNodesOpts]. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type ReplaceNodesOption = request.WriteCall

func WithScaleInOptions(value ScaleInOpts) ScaleInOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithScaleOutOptions(value ScaleOutOpts) ScaleOutOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithResizeOptions(value ResizeOpts) ResizeOption { return request.Custom(senlin.Snapshot(value)) }
func WithAddNodesOptions(value AddNodesOpts) AddNodesOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithRemoveNodesOptions(value RemoveNodesOpts) RemoveNodesOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithReplaceNodesOptions(value ReplaceNodesOpts) ReplaceNodesOption {
	return request.Custom(senlin.Snapshot(value))
}
