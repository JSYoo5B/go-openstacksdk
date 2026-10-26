package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// keyManagerCleanup follows openstack/key_manager/v1/_proxy.py
// _service_cleanup: every container, then every secret, each evaluated with
// the filters or the selector. Python does not clean orders, and neither
// does this unit. Python declares no cleanup dependencies for key_manager
// and does not wait after a deletion; Barbican deletes synchronously.
type keyManagerCleanup struct{}

// keyManagerDeleteCodes are the success statuses of a Barbican DELETE.
var keyManagerDeleteCodes = []int{200, 202, 204}

func (keyManagerCleanup) kinds() []kindSpec {
	return []kindSpec{
		// Container and Secret map created_at and updated_at to the wire
		// fields created and updated.
		{kind: "container", createdField: "created", updatedField: "updated",
			remove: restRemove("containers", "container", keyManagerDeleteCodes...)},
		{kind: "secret", createdField: "created", updatedField: "updated",
			remove: restRemove("secrets", "secret", keyManagerDeleteCodes...)},
	}
}

func (keyManagerCleanup) discover(ctx context.Context, d *discovery) error {
	for _, kind := range []struct{ kind, plural, ref string }{
		{"container", "containers", "container_ref"},
		{"secret", "secrets", "secret_ref"},
	} {
		if d.Skipped(kind.kind) {
			continue
		}
		for r, err := range listKeyManagerRows(ctx, d.Client(), kind.plural) {
			if err != nil {
				return fmt.Errorf("list %s: %w", kind.plural, err)
			}
			id, name, ok := barbicanIdentity(d, kind.kind, kind.ref, r)
			if !ok {
				continue
			}
			// Barbican lists only the token project's containers and
			// secrets (no acl_only query is sent) and its rows carry no
			// project field; a row that reports another project is kept.
			if reason, foreign := ownerMismatch(r, d.Project().ID, "project_id"); foreign {
				d.Exclude(Ref{Kind: kind.kind, ID: id}, name, reason, r.raw)
				continue
			}
			c := d.Row(kind.kind, r, id, name)
			keep, err := d.Evaluate(c)
			if err != nil {
				return err
			}
			if keep {
				c.Reason = selectionReason(d)
				d.Add(c, 0)
			}
		}
	}
	return nil
}

// barbicanIdentity derives the ID like Python's HREFToUUID formatter behind
// container_id and secret_id: the last path component of an absolute
// reference with scheme, authority and path. A row whose reference is
// missing, null, not a string or not such a URL, or whose last component is
// not a single path segment, is excluded.
func barbicanIdentity(d *discovery, kind, refField string, r row) (string, string, bool) {
	name, _ := r.str("name")
	raw, present := r.fields[refField]
	if !present || string(raw) == "null" {
		d.Exclude(Ref{Kind: kind}, name, refField+" is not available", r.raw)
		return "", "", false
	}
	projected, err := jsonfilter.ReferenceLastComponent(raw)
	if err != nil {
		d.Exclude(Ref{Kind: kind}, name, fmt.Sprintf("%s cannot be converted to an id: %v", refField, err), r.raw)
		return "", "", false
	}
	var id string
	if json.Unmarshal(projected, &id) != nil || resource.ID(id).Validate() != nil {
		d.Exclude(Ref{Kind: kind, ID: id}, name, refField+" has no usable id", r.raw)
		return "", "", false
	}
	return id, name, true
}

// listKeyManagerRows yields the rows of a Barbican list. Barbican pages with
// limit and offset and advertises the next page in the top-level next field,
// which Python's Resource.list follows.
func listKeyManagerRows(ctx context.Context, client *gophercloud.ServiceClient, plural string) iter.Seq2[row, error] {
	spec := restSpec(client, plural, plural, "", plural)
	spec.Paging = rest.PagePolicy[resource.RawResource]{OffsetPagination: true, HTTPLink: true, IgnoreFalseyNext: true,
		StopOnEmptyPage: true, VersionedPath: "/v1/" + plural}
	return func(yield func(row, error) bool) {
		for value, err := range rest.List(ctx, spec, nil) {
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
