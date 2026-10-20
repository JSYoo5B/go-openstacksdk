package shareaccessrules

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const listControlArgument = "shareaccessrules.list_control"

// WithListMaxItems caps consumed access rules locally. Zero is unlimited; a
// negative value is rejected before HTTP. It never adds a wire limit or marker.
func WithListMaxItems(value int) ListOption {
	return withListControl(func(control *resource.ListControl) { control.MaxItems = value })
}

// WithListPaginated selects one response when false. The modern access endpoint
// already returns a single collection, so neither value follows advertised links.
func WithListPaginated(value bool) ListOption {
	return withListControl(func(control *resource.ListControl) { control.SinglePage = !value })
}

func withListControl(edit func(*resource.ListControl)) ListOption {
	return func(cfg *request.Config[ListOpts]) error {
		control, _, err := request.Argument[resource.ListControl](*cfg, listControlArgument)
		if err != nil {
			return err
		}
		edit(&control)
		if cfg.Arguments == nil {
			cfg.Arguments = make(map[string]any)
		}
		cfg.Arguments[listControlArgument] = control
		return nil
	}
}
