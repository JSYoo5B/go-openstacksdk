package servers

import (
	"context"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// EvacuateServerOpts mirrors Server.evacuate: a nil field is omitted, so the
// default body is {"evacuate": {}}, while empty strings and false are sent.
// The generated Evacuate always sends onSharedStorage, which Nova rejects
// from 2.14.
type EvacuateServerOpts struct {
	Host            *string `json:"host"`
	AdminPass       *string `json:"adminPass"`
	Force           *bool   `json:"force"`
	OnSharedStorage *bool   `json:"onSharedStorage"`
}
type EvacuateServerOption = request.Option[EvacuateServerOpts]

func WithEvacuateServerOptions(value EvacuateServerOpts) EvacuateServerOption {
	value = cloneEvacuateServerOpts(value)
	return func(c *request.Config[EvacuateServerOpts]) error {
		c.Options = cloneEvacuateServerOpts(value)
		return nil
	}
}
func WithEvacuateServerHost(value string) EvacuateServerOption {
	return func(c *request.Config[EvacuateServerOpts]) error { c.Options.Host = &value; return nil }
}
func WithEvacuateServerAdminPass(value string) EvacuateServerOption {
	return func(c *request.Config[EvacuateServerOpts]) error { c.Options.AdminPass = &value; return nil }
}
func WithEvacuateServerForce(value bool) EvacuateServerOption {
	return func(c *request.Config[EvacuateServerOpts]) error { c.Options.Force = &value; return nil }
}
func WithEvacuateServerOnSharedStorage(value bool) EvacuateServerOption {
	return func(c *request.Config[EvacuateServerOpts]) error { c.Options.OnSharedStorage = &value; return nil }
}
func WithEvacuateServerHeader(key, value string) EvacuateServerOption {
	return request.WithHeader[EvacuateServerOpts](key, value)
}

// WithEvacuateServerField adds an extension key inside the evacuate object.
// Keys owned by EvacuateServerOpts are rejected.
func WithEvacuateServerField(key string, value any) EvacuateServerOption {
	return request.WithField[EvacuateServerOpts](key, value)
}

func cloneEvacuateServerOpts(v EvacuateServerOpts) EvacuateServerOpts {
	return EvacuateServerOpts{Host: clonePointer(v.Host), AdminPass: clonePointer(v.AdminPass), Force: clonePointer(v.Force), OnSharedStorage: clonePointer(v.OnSharedStorage)}
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// EvacuateWithOptions implements evacuate_server. It posts the evacuate action
// with the client microversion, the empty Accept header Python sends and every
// status below 400 accepted. It does not gate keys by microversion.
func (a *API) EvacuateWithOptions(ctx context.Context, serverID string, options ...EvacuateServerOption) error {
	const op = "EvacuateWithOptions"
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, op, err)
	}
	c, err := applyUserActionOptions(p, EvacuateServerOpts{}, cloneEvacuateServerOpts, true, options)
	if err == nil {
		err = p.version("2.1", "")
	}
	fields := map[string]any{}
	if err == nil {
		if c.Options.Host != nil {
			fields["host"] = *c.Options.Host
		}
		if c.Options.AdminPass != nil {
			fields["adminPass"] = *c.Options.AdminPass
		}
		if c.Options.Force != nil {
			fields["force"] = *c.Options.Force
		}
		if c.Options.OnSharedStorage != nil {
			fields["onSharedStorage"] = *c.Options.OnSharedStorage
		}
	}
	var body map[string]any
	if err == nil {
		body, err = userActionBody("evacuate", fields, c)
	}
	if err == nil {
		err = p.send(http.MethodPost, "action", body)
	}
	return userActionWrap(ctx, op, err)
}

// MigrateServerOpts mirrors Server.migrate. An empty Host is omitted because
// Python tests the host for truthiness.
type MigrateServerOpts struct {
	Host string `json:"host"`
}
type MigrateServerOption = request.Option[MigrateServerOpts]

func WithMigrateServerHost(value string) MigrateServerOption {
	return func(c *request.Config[MigrateServerOpts]) error { c.Options.Host = value; return nil }
}
func WithMigrateServerHeader(key, value string) MigrateServerOption {
	return request.WithHeader[MigrateServerOpts](key, value)
}

// WithMigrateServerField adds an extension key inside the migrate object.
func WithMigrateServerField(key string, value any) MigrateServerOption {
	return request.WithField[MigrateServerOpts](key, value)
}

// MigrateWithOptions implements migrate_server: {"migrate": null} without a
// host and {"migrate": {"host": h}} with one. A host needs selected Nova 2.56
// or later and fails with resource.ErrUnsupported before HTTP otherwise.
func (a *API) MigrateWithOptions(ctx context.Context, serverID string, options ...MigrateServerOption) error {
	const op = "MigrateWithOptions"
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, op, err)
	}
	c, err := applyUserActionOptions(p, MigrateServerOpts{}, func(v MigrateServerOpts) MigrateServerOpts { return v }, true, options)
	if err == nil {
		err = p.version("2.1", "")
	}
	var fields map[string]any
	if err == nil && c.Options.Host != "" {
		err = p.version("2.56", "")
		fields = map[string]any{"host": c.Options.Host}
	}
	var body map[string]any
	if err == nil {
		body, err = userActionBody("migrate", fields, c)
	}
	if err == nil {
		err = p.send(http.MethodPost, "action", body)
	}
	return userActionWrap(ctx, op, err)
}

// LiveMigrateServerOpts mirrors Server.live_migrate. An empty Host is omitted
// (sent as null) like Python's truthiness check. BlockMigration nil means
// Python's None, which becomes "auto" from 2.25 and false before it;
// AutoBlockMigration sends "auto" explicitly and excludes BlockMigration.
// DiskOverCommit is used only by the pre-2.25 body, where nil means false.
type LiveMigrateServerOpts struct {
	Host               string `json:"host"`
	Force              bool   `json:"force"`
	BlockMigration     *bool  `json:"block_migration"`
	AutoBlockMigration bool   `json:"-"`
	DiskOverCommit     *bool  `json:"disk_over_commit"`
}
type LiveMigrateServerOption = request.Option[LiveMigrateServerOpts]

func WithLiveMigrateServerOptions(value LiveMigrateServerOpts) LiveMigrateServerOption {
	value = cloneLiveMigrateServerOpts(value)
	return func(c *request.Config[LiveMigrateServerOpts]) error {
		c.Options = cloneLiveMigrateServerOpts(value)
		return nil
	}
}
func WithLiveMigrateServerHost(value string) LiveMigrateServerOption {
	return func(c *request.Config[LiveMigrateServerOpts]) error { c.Options.Host = value; return nil }
}
func WithLiveMigrateServerForce(value bool) LiveMigrateServerOption {
	return func(c *request.Config[LiveMigrateServerOpts]) error { c.Options.Force = value; return nil }
}

// WithLiveMigrateServerBlockMigration sends a boolean block_migration and
// clears an earlier WithLiveMigrateServerAutoBlockMigration.
func WithLiveMigrateServerBlockMigration(value bool) LiveMigrateServerOption {
	return func(c *request.Config[LiveMigrateServerOpts]) error {
		c.Options.BlockMigration, c.Options.AutoBlockMigration = &value, false
		return nil
	}
}

// WithLiveMigrateServerAutoBlockMigration sends block_migration "auto",
// which needs the 2.25 or 2.30 body.
func WithLiveMigrateServerAutoBlockMigration() LiveMigrateServerOption {
	return func(c *request.Config[LiveMigrateServerOpts]) error {
		c.Options.BlockMigration, c.Options.AutoBlockMigration = nil, true
		return nil
	}
}
func WithLiveMigrateServerDiskOverCommit(value bool) LiveMigrateServerOption {
	return func(c *request.Config[LiveMigrateServerOpts]) error { c.Options.DiskOverCommit = &value; return nil }
}
func WithLiveMigrateServerHeader(key, value string) LiveMigrateServerOption {
	return request.WithHeader[LiveMigrateServerOpts](key, value)
}

// WithLiveMigrateServerField adds an extension key inside os-migrateLive.
func WithLiveMigrateServerField(key string, value any) LiveMigrateServerOption {
	return request.WithField[LiveMigrateServerOpts](key, value)
}

func cloneLiveMigrateServerOpts(v LiveMigrateServerOpts) LiveMigrateServerOpts {
	v.BlockMigration = clonePointer(v.BlockMigration)
	v.DiskOverCommit = clonePointer(v.DiskOverCommit)
	return v
}

// LiveMigrateWithOptions implements live_migrate_server. Python picks the
// body from supported microversions; Go picks it from the selected client
// microversion ("latest" counts as 2.30 or later). The 2.30 and 2.25 bodies
// are sent with microversion 2.30 and 2.25 like Python, and the legacy body
// with the client microversion. Python's ValueError cases (a host without
// force before 2.30, "auto" before 2.25) fail with resource.ErrInvalidOption
// before HTTP.
func (a *API) LiveMigrateWithOptions(ctx context.Context, serverID string, options ...LiveMigrateServerOption) error {
	const op = "LiveMigrateWithOptions"
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return userActionWrap(ctx, op, err)
	}
	c, err := applyUserActionOptions(p, LiveMigrateServerOpts{}, cloneLiveMigrateServerOpts, true, options)
	if err == nil {
		err = p.version("2.1", "")
	}
	if err == nil && c.Options.AutoBlockMigration && c.Options.BlockMigration != nil {
		err = userActionInvalid("block migration cannot be both auto and a boolean")
	}
	var fields map[string]any
	if err == nil {
		fields, err = p.liveMigrateFields(c.Options)
	}
	var body map[string]any
	if err == nil {
		body, err = userActionBody("os-migrateLive", fields, c)
	}
	if err == nil {
		err = p.send(http.MethodPost, "action", body)
	}
	return userActionWrap(ctx, op, err)
}

func (p *pythonUserAction) supports(version string) (bool, error) {
	err := p.version(version, "")
	if errors.Is(err, resource.ErrUnsupported) {
		return false, nil
	}
	return err == nil, err
}

func (p *pythonUserAction) liveMigrateFields(o LiveMigrateServerOpts) (map[string]any, error) {
	modern := ""
	for _, candidate := range []string{"2.30", "2.25"} {
		supported, err := p.supports(candidate)
		if err != nil {
			return nil, err
		}
		if supported {
			modern = candidate
			break
		}
	}
	fields := map[string]any{"host": nil}
	if modern == "" {
		if o.AutoBlockMigration {
			return nil, userActionInvalid("block_migration auto needs Nova 2.25 or later")
		}
		fields["block_migration"] = o.BlockMigration != nil && *o.BlockMigration
		fields["disk_over_commit"] = o.DiskOverCommit != nil && *o.DiskOverCommit
	} else {
		fields["block_migration"] = "auto"
		if o.BlockMigration != nil {
			fields["block_migration"] = *o.BlockMigration
		}
	}
	if o.Host != "" {
		fields["host"] = o.Host
		if modern != "2.30" && !o.Force {
			return nil, userActionInvalid("a live migration host needs force before Nova 2.30")
		}
		if modern == "2.30" && o.Force {
			fields["force"] = true
		}
	}
	if modern != "" {
		// Python passes the body's microversion to the action explicitly.
		if err := p.source.WithPolicy(p.ctx, &modern, nil); err != nil {
			return nil, err
		}
	}
	return fields, nil
}
