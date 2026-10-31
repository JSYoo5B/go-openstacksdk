package volumeconnectors

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/ironic"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts is the typed create body. Pointer fields and maps are omitted only when nil,
// so explicit false, zero, empty string and empty map values are sent. Required string
// fields are omitted when empty and the server reports the missing value.
type CreateOpts struct {
	UUID        *string
	NodeUUID    string
	Type        string
	ConnectorID string
	Extra       map[string]any
}

// CreateOption is one option of Create: WithCreateOptions, request.Custom
// over *request.Config[CreateOpts] or request.Field. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = calls.Spec[request.Config[CreateOpts]]{Field: calls.ConfigField[CreateOpts]}

// WithCreateOptions replaces the typed create options.
func WithCreateOptions(value CreateOpts) CreateOption {
	return request.Custom(request.WithOptions(value))
}

var createReserved = []string{"uuid", "node_uuid", "type", "connector_id", "extra", "created_at", "updated_at", "links"}

func prepareCreate(opts CreateOpts, options ...CreateOption) (map[string]any, error) {
	cfg, err := calls.ApplyConfig(opts, createCalls, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(cfg, true, false, false); err != nil {
		return nil, err
	}
	o := cfg.Options
	core := map[string]any{}
	if o.UUID != nil {
		core["uuid"] = o.UUID
	}
	if o.NodeUUID != "" {
		core["node_uuid"] = o.NodeUUID
	}
	if o.Type != "" {
		core["type"] = o.Type
	}
	if o.ConnectorID != "" {
		core["connector_id"] = o.ConnectorID
	}
	if o.Extra != nil {
		core["extra"] = o.Extra
	}
	return ironic.BuildBody(core, cfg.Fields, createReserved...)
}

// GetOpts selects wire field names for a field-limited read.
type GetOpts struct{ Fields []string }

// GetOption is one option of Get: WithGetOptions, WithGetFields or
// request.Custom over *request.Config[GetOpts]. Shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

func WithGetOptions(value GetOpts) GetOption { return request.Custom(request.WithOptions(value)) }

// WithGetFields sends fields=<comma list>; names are wire names and must be nonempty and comma-free.
func WithGetFields(fields ...string) GetOption {
	copied := append([]string{}, fields...)
	return request.Custom(func(c *request.Config[GetOpts]) error { c.Options.Fields = copied; return nil })
}

func prepareGet(options ...GetOption) ([]string, error) {
	cfg, err := calls.ApplyConfig(GetOpts{}, calls.Spec[request.Config[GetOpts]]{}, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(cfg, false, false, false); err != nil {
		return nil, err
	}
	if cfg.Options.Fields != nil {
		if _, err := ironic.JoinFields(cfg.Options.Fields); err != nil {
			return nil, err
		}
	}
	return cfg.Options.Fields, nil
}

// PatchOperation is one JSON Patch operation. Use Add, Replace and Remove.
type PatchOperation = ironic.PatchOperation

// Add, Replace and Remove build operations. Add and Replace always send "value" (nil is JSON null).
func Add(path string, value any) PatchOperation     { return ironic.Add(path, value) }
func Replace(path string, value any) PatchOperation { return ironic.Replace(path, value) }
func Remove(path string) PatchOperation             { return ironic.Remove(path) }

// FieldPatches builds the operations Python update_volumeconnectors(**attrs) sends: one add per top-level
// attribute, sorted by path.
func FieldPatches(fields map[string]any) []PatchOperation { return ironic.FieldPatches(fields) }

// UpdateOpts is the JSON Patch document for one update.
type UpdateOpts struct{ Operations []PatchOperation }

// UpdateOption is one option of Update: WithUpdateOptions,
// WithUpdateOperations or request.Custom over *request.Config[UpdateOpts].
// Shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

// WithUpdateOptions replaces the initial operation list; it does not append.
func WithUpdateOptions(value UpdateOpts) UpdateOption {
	return request.Custom(request.WithOptions(value))
}

// WithUpdateOperations appends operations in the given order.
func WithUpdateOperations(operations ...PatchOperation) UpdateOption {
	copied := append([]PatchOperation(nil), operations...)
	return request.Custom(func(c *request.Config[UpdateOpts]) error {
		c.Options.Operations = append(append([]PatchOperation(nil), c.Options.Operations...), copied...)
		return nil
	})
}

func prepareUpdate(opts UpdateOpts, options ...UpdateOption) ([]PatchOperation, error) {
	cfg, err := calls.ApplyConfig(opts, calls.Spec[request.Config[UpdateOpts]]{}, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(cfg, false, false, false); err != nil {
		return nil, err
	}
	return cfg.Options.Operations, nil
}

// ListOpts are the typed list filters. Zero values are omitted. Fields uses wire names and
// is rejected for detailed lists before HTTP.
type ListOpts struct {
	Fields  []string
	Limit   int
	Marker  string
	SortKey string
	SortDir string
	Node    string
}

// ListOption is one option of List and ListDetail: WithListOptions,
// request.Custom over *request.Config[ListOpts] or request.Query with a string
// or []string value. Keys owned by typed filters are rejected before HTTP.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = calls.Spec[request.Config[ListOpts]]{Query: calls.ConfigQuery[ListOpts]}

func WithListOptions(value ListOpts) ListOption { return request.Custom(request.WithOptions(value)) }

func prepareList(options ...ListOption) (map[string][]string, error) {
	cfg, err := calls.ApplyConfig(ListOpts{}, listCalls, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(cfg, false, true, false); err != nil {
		return nil, err
	}
	o := cfg.Options
	values, err := ironic.ListQuery{Fields: o.Fields, Limit: o.Limit, Marker: o.Marker, SortKey: o.SortKey, SortDir: o.SortDir,
		Node: o.Node, Address: "", AllowNode: true, AllowAddress: false}.Values()
	if err != nil {
		return nil, err
	}
	return ironic.MergeQuery(values, cfg.Query)
}
