package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// imageCleanup follows openstack/image/v2/_proxy.py _service_cleanup: it
// lists images with owner=<project ID> and deletes each one that the
// filters and the selector accept. Python does not wait for image deletion,
// and neither does this unit; Glance deletes images synchronously.
type imageCleanup struct{}

func (imageCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "image", createdField: "created_at", updatedField: "updated_at",
			remove: restRemove("images", "image", 204)},
	}
}

func (imageCleanup) discover(ctx context.Context, d *discovery) error {
	if d.Skipped("image") {
		return nil
	}
	project := d.Project().ID
	// Python sends only the owner filter. Glance then omits hidden images
	// (os_hidden=true) and images that other projects own, whatever their
	// visibility; images that the project owns are listed at any visibility.
	for r, err := range listImageRows(ctx, d.Client(), url.Values{"owner": {project}}) {
		if err != nil {
			return fmt.Errorf("list images: %w", err)
		}
		id, name, ok := rowIdentity(d, "image", r)
		if !ok {
			continue
		}
		// The owner query is the ownership boundary; a row that does not
		// report this project as its owner is never deleted.
		if reason, foreign := ownerMismatch(r, project, "owner"); foreign {
			d.Exclude(Ref{Kind: "image", ID: id}, name, reason, r.raw)
			continue
		}
		if owner, _ := r.str("owner"); owner != project {
			d.Exclude(Ref{Kind: "image", ID: id}, name, "owner is not reported", r.raw)
			continue
		}
		c := d.Row("image", r, id, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if keep {
			c.Reason = selectionReason(d)
			d.Add(c, 0)
		}
	}
	return nil
}

// listImageRows lists Glance images. Glance advertises the next page as a
// top-level "next" path that starts with /v2/images, which is accepted as
// the same collection when the endpoint has a different path prefix.
func listImageRows(ctx context.Context, client *gophercloud.ServiceClient, query url.Values) iter.Seq2[row, error] {
	spec := restSpec(client, "images", "images", "", "images")
	spec.Paging.VersionedPath = "/v2/images"
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
