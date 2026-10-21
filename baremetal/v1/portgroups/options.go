package portgroups

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/ironic"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts is the typed create body. Pointer fields and maps are omitted only when nil,
// so explicit false, zero, empty string and empty map values are sent. Required string
// fields are omitted when empty and the server reports the missing value.
type CreateOpts struct {
	UUID                     *string
	NodeUUID                 string
	Name                     *string
	Address                  *string
	StandalonePortsSupported *bool
	Mode                     *string
	Properties               map[string]any
	Extra                    map[string]any
}

type CreateOption = request.Option[CreateOpts]

// WithCreateOptions replaces the typed create options.
func WithCreateOptions(value CreateOpts) CreateOption { return request.WithOptions(value) }

// WithCreateField adds one extension body field. Typed and server-owned names are rejected before HTTP.
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

var createReserved = []string{"uuid", "node_uuid", "name", "address", "standalone_ports_supported", "mode", "properties", "extra", "created_at", "updated_at", "links", "ports", "internal_info"}

func prepareCreate(opts CreateOpts, options ...CreateOption) (map[string]any, error) {
	cfg, err := request.Apply(opts, options...)
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
	if o.Name != nil {
		core["name"] = o.Name
	}
	if o.Address != nil {
		core["address"] = o.Address
	}
	if o.StandalonePortsSupported != nil {
		core["standalone_ports_supported"] = o.StandalonePortsSupported
	}
	if o.Mode != nil {
		core["mode"] = o.Mode
	}
	if o.Properties != nil {
		core["properties"] = o.Properties
	}
	if o.Extra != nil {
		core["extra"] = o.Extra
	}
	return ironic.BuildBody(core, cfg.Fields, createReserved...)
}

// GetOpts selects wire field names for a field-limited read.
type GetOpts struct{ Fields []string }

type GetOption = request.Option[GetOpts]

func WithGetOptions(value GetOpts) GetOption { return request.WithOptions(value) }

// WithGetFields sends fields=<comma list>; names are wire names and must be nonempty and comma-free.
func WithGetFields(fields ...string) GetOption {
	copied := append([]string{}, fields...)
	return func(c *request.Config[GetOpts]) error { c.Options.Fields = copied; return nil }
}

func prepareGet(options ...GetOption) ([]string, error) {
	cfg, err := request.Apply(GetOpts{}, options...)
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

// FieldPatches builds the operations Python update_portgroups(**attrs) sends: one add per top-level
// attribute, sorted by path.
func FieldPatches(fields map[string]any) []PatchOperation { return ironic.FieldPatches(fields) }

// UpdateOpts is the JSON Patch document for one update.
type UpdateOpts struct{ Operations []PatchOperation }

type UpdateOption = request.Option[UpdateOpts]

// WithUpdateOptions replaces the initial operation list; it does not append.
func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.WithOptions(value) }

// WithUpdateOperations appends operations in the given order.
func WithUpdateOperations(operations ...PatchOperation) UpdateOption {
	copied := append([]PatchOperation(nil), operations...)
	return func(c *request.Config[UpdateOpts]) error {
		c.Options.Operations = append(append([]PatchOperation(nil), c.Options.Operations...), copied...)
		return nil
	}
}

func prepareUpdate(opts UpdateOpts, options ...UpdateOption) ([]PatchOperation, error) {
	cfg, err := request.Apply(opts, options...)
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
	Address string
}

type ListOption = request.Option[ListOpts]

func WithListOptions(value ListOpts) ListOption { return request.WithOptions(value) }

// WithListQuery adds an extension query. Keys owned by typed filters are rejected before HTTP.
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func prepareList(options ...ListOption) (map[string][]string, error) {
	cfg, err := request.Apply(ListOpts{}, options...)
	if err != nil {
		return nil, err
	}
	if err := request.ValidateCapabilities(cfg, false, true, false); err != nil {
		return nil, err
	}
	o := cfg.Options
	values, err := ironic.ListQuery{Fields: o.Fields, Limit: o.Limit, Marker: o.Marker, SortKey: o.SortKey, SortDir: o.SortDir,
		Node: o.Node, Address: o.Address, AllowNode: true, AllowAddress: true}.Values()
	if err != nil {
		return nil, err
	}
	return ironic.MergeQuery(values, cfg.Query)
}
