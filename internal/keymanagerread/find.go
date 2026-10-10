package keymanagerread

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FindIdentity implements Python Resource.find for containers and orders:
// one strict direct Fetch, then on a clean 400/403/404 a complete list whose
// rows are projected without a request seed and matched by their literal id
// (or full alternate reference) or name. Neither resource maps a name query.
func FindIdentity(ctx context.Context, client *gophercloud.ServiceClient, kind, identity string, prepared resource.IdentityFindOpts) (*Record, error) {
	if kind != "containers" && kind != "orders" {
		return nil, invalid("unsupported key-manager find resource")
	}
	// The pinned direct find declarations accept only ignore_missing.
	if len(prepared.Query) != 0 || prepared.Details != nil || prepared.AllProjects != nil || prepared.GetExtraSpecs != nil {
		return nil, resource.ErrUnsupported
	}
	if err := validateClient(ctx, client); err != nil {
		return nil, err
	}
	snapshot := *client
	snapshot.MoreHeaders = maps.Clone(client.MoreHeaders)
	validate := func(ctx context.Context) error { return validateClient(ctx, client) }
	spec := rest.CollectionSpec[resource.RawResource]{
		Client: &snapshot, Path: kind, Kind: kind, PluralKey: kind,
		Metadata:  func(value *resource.RawResource) *resource.Metadata { return &value.Metadata },
		Validate:  validate,
		ListCodes: []int{http.StatusOK},
		Paging:    rest.PagePolicy[resource.RawResource]{OffsetPagination: true, HTTPLink: true, StopOnEmptyPage: true},
	}
	binding := resource.NewCollection(resource.Adapter[Record]{
		Kind: kind, IdentityFind: true, ValidateID: validateFindID,
		IdentityResponseID: recordResponseID,
		Name:               recordName,
		Get: func(ctx context.Context, id string) (*Record, error) {
			if err := validate(ctx); err != nil {
				return nil, err
			}
			value, err := Fetch[struct{}](ctx, &snapshot, kind, resource.ID(id))
			return value, errors.Join(err, validate(ctx))
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*Record, error] {
			return func(yield func(*Record, error) bool) {
				for row, err := range rest.List(ctx, spec, query) {
					if err != nil {
						yield(nil, err)
						return
					}
					view, err := projectRecord(row, kind, nil)
					if err != nil {
						yield(nil, err)
						return
					}
					record := &Record{Resource: view, Wire: row.Clone(), Header: row.Header.Clone(), StatusCode: row.StatusCode}
					if !yield(record, nil) {
						return
					}
				}
			}
		},
	})
	return binding.FindIdentity(ctx, identity, resource.WithIdentityFindOptions(prepared))
}

func validateClient(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return invalid("key-manager service client is required")
	}
	if client.Type != "key-manager" {
		return resource.ErrUnsupported
	}
	return nil
}

func validateFindID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) || strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return invalid("find identity must be valid text without spaces or controls")
	}
	return nil
}

// recordResponseID is Python _get_id: the literal projected id, which is the
// request ID for a direct fetch or the full alternate reference for a list row.
// A null or non-string id cannot equal the string input.
func recordResponseID(value *Record) (string, error) {
	if value == nil || value.Resource == nil {
		return "", resource.ErrInvalidOption
	}
	var id string
	if err := json.Unmarshal(value.Resource.Body["id"], &id); err != nil {
		return "", nil
	}
	return id, nil
}

// recordName is maybe_result.name compared with ==: only a JSON string can
// equal the input. Other values map to "", which no validated identity equals.
func recordName(value *Record) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	raw := bytes.TrimSpace(value.Resource.Body["name"])
	var name string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &name) != nil {
		return ""
	}
	return name
}
