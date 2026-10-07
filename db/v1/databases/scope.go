package databases

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/db/v1/instances"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"

	upstream "github.com/gophercloud/gophercloud/v2/openstack/db/v1/databases"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// DatabaseScope fixes a Trove instance. Database names are the identifiers;
// The pinned SDKs expose no per-database fetch, so Get searches this instance's
// complete list.
type DatabaseScope struct {
	*resource.Collection[Database]
	api        *API
	instanceID string
}

// InInstance resolves the instance once. Explicit instance IDs avoid lookup.
func (a *API) InInstance(ctx context.Context, parent resource.Ref) (*DatabaseScope, error) {
	id, err := instances.New(a.client).Resources.ResolveID(ctx, parent)
	if err != nil {
		return nil, err
	}
	scope := &DatabaseScope{api: a, instanceID: id}
	scope.Collection = resource.NewCollection(resource.Adapter[Database]{
		Kind:       "databases",
		ValidateID: validateDatabaseName,
		Get: func(ctx context.Context, name string) (*Database, error) {
			return scope.Collection.Find(ctx, resource.Name(name))
		},
		ID:   func(value *Database) string { return value.Name },
		Name: func(value *Database) string { return value.Name },
		Delete: func(ctx context.Context, name string) error {
			if err := validateDatabaseName(name); err != nil {
				return err
			}
			return a.Delete(ctx, id, url.PathEscape(name))
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*Database, error] {
			return scope.listControlled(ctx, query, resource.ListControl{})
		},
		IterateControlled: scope.listControlled,
	})
	return scope, nil
}

func (s *DatabaseScope) listControlled(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*Database, error] {
	if len(query) != 0 {
		return func(yield func(*Database, error) bool) { yield(nil, resource.ErrUnsupported) }
	}
	return resource.StreamWithControl(ctx, upstream.List(s.api.client, s.instanceID), extractScopedDatabases, control)
}

// Gophercloud's Database lacks a character_set tag. Own the wire conversion so
// the scoped SDK resource preserves the charset provided by Trove.
func extractScopedDatabases(page pagination.Page) ([]Database, error) {
	var wire struct {
		Databases []struct {
			Name    string `json:"name"`
			CharSet string `json:"character_set"`
			Collate string `json:"collate"`
		} `json:"databases"`
	}
	if err := page.(upstream.DBPage).ExtractInto(&wire); err != nil {
		return nil, err
	}
	values := make([]Database, len(wire.Databases))
	for i, value := range wire.Databases {
		values[i] = Database{Name: value.Name, CharSet: value.CharSet, Collate: value.Collate}
	}
	return values, nil
}

// Create submits one database in Trove's databases array. The asynchronous API
// returns no database body, so success is reported as a nil error.
func (s *DatabaseScope) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) error {
	return s.create(ctx, BatchCreateOpts{opts}, true, options...)
}

// CreateBatch submits several databases in one request. It validates the whole
// batch before POST; Trove determines atomicity and completion.
func (s *DatabaseScope) CreateBatch(ctx context.Context, opts BatchCreateOpts, options ...CreateOption) error {
	return s.create(ctx, opts, false, options...)
}

func (s *DatabaseScope) create(ctx context.Context, opts BatchCreateOpts, single bool, options ...CreateOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg, err := request.Apply(opts, options...)
	if err != nil {
		return request.Wrap("create", "databases", err)
	}
	if len(cfg.Options) == 0 {
		return databaseInvalid("create batch must not be empty")
	}
	if single && len(cfg.Options) != 1 {
		return databaseInvalid("single create requires exactly one database")
	}
	if err := request.ValidateCapabilities(cfg, true, false, false); err != nil {
		return request.Wrap("create", "databases", err)
	}
	seen := make(map[string]bool, len(cfg.Options))
	for _, database := range cfg.Options {
		if err := validateDatabaseName(database.Name); err != nil {
			return err
		}
		if len(database.Name) > 64 {
			return databaseInvalid("database name exceeds 64 bytes")
		}
		if seen[database.Name] {
			return databaseInvalid("duplicate database name in create batch")
		}
		seen[database.Name] = true
	}
	// Serialize before the native call so invalid extension fields fail before
	// a request and the single-create contract cannot become a batch via options.
	builder := createOptsBuilder{base: cfg.Options, config: cfg}
	if _, err := builder.ToDBCreateMap(); err != nil {
		return request.Wrap("create", "databases", err)
	}
	return request.Wrap("create", "databases", upstream.Create(ctx, s.api.client, s.instanceID, builder).ExtractErr())
}

func validateDatabaseName(name string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00\r\n") {
		return databaseInvalid("database name must identify one non-empty path segment")
	}
	return nil
}

func databaseInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
