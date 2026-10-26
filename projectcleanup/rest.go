package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// row is one raw list row.
type row struct {
	fields map[string]json.RawMessage
	raw    json.RawMessage
}

// str returns a string field; present is false for absent, null or
// non-string values.
func (r row) str(key string) (string, bool) {
	value, ok := r.fields[key]
	if !ok {
		return "", false
	}
	var text string
	if json.Unmarshal(value, &text) != nil {
		return "", false
	}
	return text, true
}

func rawID(value *resource.RawResource) string {
	var id string
	_ = json.Unmarshal(value.Body["id"], &id)
	return id
}

func rawStatus(value *resource.RawResource) string {
	var status string
	_ = json.Unmarshal(value.Body["status"], &status)
	return status
}

// restSpec is an SDK-owned raw collection: rows keep every wire field, so
// filters can tell absent, null and unparsable times apart and plans can
// carry the original rows.
func restSpec(client *gophercloud.ServiceClient, path, kind, single, plural string, deleteCodes ...int) rest.CollectionSpec[resource.RawResource] {
	return rest.CollectionSpec[resource.RawResource]{Client: client, Path: path, Kind: kind, SingleKey: single, PluralKey: plural,
		ID: rawID, Status: rawStatus, Metadata: func(v *resource.RawResource) *resource.Metadata { return &v.Metadata },
		Get: true, Delete: true, DeleteCodes: deleteCodes, Paging: rest.PagePolicy[resource.RawResource]{HTTPLink: true}}
}

// listRows yields the rows of a project-scoped list (no all-projects query).
func listRows(ctx context.Context, client *gophercloud.ServiceClient, path, plural string, query url.Values) iter.Seq2[row, error] {
	spec := restSpec(client, path, plural, "", plural)
	return func(yield func(row, error) bool) {
		for value, err := range rest.List(ctx, spec, query) {
			if err != nil {
				yield(row{}, err)
				return
			}
			raw, err := json.Marshal(value.Body)
			if err != nil {
				yield(row{}, err)
				return
			}
			if !yield(row{fields: value.Body, raw: raw}, nil) {
				return
			}
		}
	}
}

// restRemove sends one DELETE to path/id. A 404 is returned as an error
// matching resource.ErrNotFound.
func restRemove(path, kind string, codes ...int) func(context.Context, *gophercloud.ServiceClient, Candidate) error {
	return func(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
		return rest.Collection(restSpec(client, path, kind, kind, path, codes...)).Delete(ctx, resource.ID(c.ID), resource.WithMissingError())
	}
}

// restAbsent polls GET path/id with the shared resource wait engine until a
// 404 or a deleted status. Polling errors are returned immediately.
func restAbsent(path, single string) func(context.Context, *gophercloud.ServiceClient, Candidate, time.Duration, time.Duration) error {
	return func(ctx context.Context, client *gophercloud.ServiceClient, c Candidate, timeout, interval time.Duration) error {
		collection := rest.Collection(restSpec(client, path, single, single, path))
		return collection.WaitDeleted(ctx, resource.ID(c.ID), resource.WithTimeout(timeout), resource.WithPollInterval(interval))
	}
}

func ownerMismatch(r row, project string, keys ...string) (string, bool) {
	for _, key := range keys {
		if owner, ok := r.str(key); ok && owner != "" && owner != project {
			return fmt.Sprintf("owned by project %s (%s)", owner, key), true
		}
	}
	return "", false
}
