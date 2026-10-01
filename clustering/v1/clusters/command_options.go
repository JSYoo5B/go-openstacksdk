package clusters

import (
	"encoding/json"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
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

type ScaleInOption = request.Option[ScaleInOpts]
type ScaleOutOption = request.Option[ScaleOutOpts]
type ResizeOption = request.Option[ResizeOpts]
type AddNodesOption = request.Option[AddNodesOpts]
type RemoveNodesOption = request.Option[RemoveNodesOpts]
type ReplaceNodesOption = request.Option[ReplaceNodesOpts]

func WithScaleInOptions(value ScaleInOpts) ScaleInOption             { return senlin.Snapshot(value) }
func WithScaleOutOptions(value ScaleOutOpts) ScaleOutOption          { return senlin.Snapshot(value) }
func WithResizeOptions(value ResizeOpts) ResizeOption                { return senlin.Snapshot(value) }
func WithAddNodesOptions(value AddNodesOpts) AddNodesOption          { return senlin.Snapshot(value) }
func WithRemoveNodesOptions(value RemoveNodesOpts) RemoveNodesOption { return senlin.Snapshot(value) }
func WithReplaceNodesOptions(value ReplaceNodesOpts) ReplaceNodesOption {
	return senlin.Snapshot(value)
}

func WithScaleInCount(value int) ScaleInOption {
	return func(config *request.Config[ScaleInOpts]) error {
		config.Options.Count = request.Present(value)
		return nil
	}
}
func WithScaleInCountNull() ScaleInOption {
	return func(config *request.Config[ScaleInOpts]) error {
		config.Options.Count = request.Null[int]()
		return nil
	}
}
func WithScaleOutCount(value int) ScaleOutOption {
	return func(config *request.Config[ScaleOutOpts]) error {
		config.Options.Count = request.Present(value)
		return nil
	}
}
func WithScaleOutCountNull() ScaleOutOption {
	return func(config *request.Config[ScaleOutOpts]) error {
		config.Options.Count = request.Null[int]()
		return nil
	}
}
func WithResizeAdjustmentType(value AdjustmentType) ResizeOption {
	return func(config *request.Config[ResizeOpts]) error {
		config.Options.AdjustmentType = request.Present(value)
		return nil
	}
}
func WithResizeNumber(value json.Number) ResizeOption {
	return func(config *request.Config[ResizeOpts]) error {
		config.Options.Number = request.Present(value)
		return nil
	}
}
func WithResizeMinSize(value int) ResizeOption {
	return func(config *request.Config[ResizeOpts]) error {
		config.Options.MinSize = request.Present(value)
		return nil
	}
}
func WithResizeMaxSize(value int) ResizeOption {
	return func(config *request.Config[ResizeOpts]) error {
		config.Options.MaxSize = request.Present(value)
		return nil
	}
}
func WithResizeMinStep(value int) ResizeOption {
	return func(config *request.Config[ResizeOpts]) error {
		config.Options.MinStep = request.Present(value)
		return nil
	}
}
func WithResizeStrict(value bool) ResizeOption {
	return func(config *request.Config[ResizeOpts]) error {
		config.Options.Strict = request.Present(value)
		return nil
	}
}
func WithRemoveNodesDestroyAfterDeletion(value bool) RemoveNodesOption {
	return func(config *request.Config[RemoveNodesOpts]) error {
		config.Options.DestroyAfterDeletion = request.Present(value)
		return nil
	}
}

func WithScaleInField(key string, value any) ScaleInOption {
	return request.WithField[ScaleInOpts](key, value)
}
func WithScaleOutField(key string, value any) ScaleOutOption {
	return request.WithField[ScaleOutOpts](key, value)
}
func WithResizeField(key string, value any) ResizeOption {
	return request.WithField[ResizeOpts](key, value)
}
func WithAddNodesField(key string, value any) AddNodesOption {
	return request.WithField[AddNodesOpts](key, value)
}
func WithRemoveNodesField(key string, value any) RemoveNodesOption {
	return request.WithField[RemoveNodesOpts](key, value)
}
func WithReplaceNodesField(key string, value any) ReplaceNodesOption {
	return request.WithField[ReplaceNodesOpts](key, value)
}

func WithScaleInHeader(key, value string) ScaleInOption {
	return request.WithHeader[ScaleInOpts](key, value)
}
func WithScaleOutHeader(key, value string) ScaleOutOption {
	return request.WithHeader[ScaleOutOpts](key, value)
}
func WithResizeHeader(key, value string) ResizeOption {
	return request.WithHeader[ResizeOpts](key, value)
}
func WithAddNodesHeader(key, value string) AddNodesOption {
	return request.WithHeader[AddNodesOpts](key, value)
}
func WithRemoveNodesHeader(key, value string) RemoveNodesOption {
	return request.WithHeader[RemoveNodesOpts](key, value)
}
func WithReplaceNodesHeader(key, value string) ReplaceNodesOption {
	return request.WithHeader[ReplaceNodesOpts](key, value)
}
