package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// objectStoreCleanup follows openstack/object_store/v1/_proxy.py
// _service_cleanup: for each container of the account, every object is
// evaluated; a container is evaluated only when all of its objects were
// selected, and it then depends on those objects. Python's skip names
// object_store.container and object_store.object each skip the whole
// service.
//
// Python's Object has updated_at (an alias of the listing's last_modified)
// and no created_at; Python's Container has neither, so any time filter
// keeps every container.
//
// Swift names are not single URL segments in general (object names may
// contain "/", spaces or "%"), so this unit addresses them with path
// escaping instead of the resource ID helpers.
type objectStoreCleanup struct{}

var objectStoreDeleteCodes = []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}

func (objectStoreCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "object", updatedField: "last_modified", remove: swiftRemove("object")},
		{kind: "container", remove: swiftRemove("container")},
	}
}

func (objectStoreCleanup) discover(ctx context.Context, d *discovery) error {
	if d.Skipped("container") || d.Skipped("object") {
		return nil
	}
	client := d.Client()
	for container, err := range swiftRows(ctx, client, "", "X-Account-Container-Count") {
		if err != nil {
			return fmt.Errorf("list containers: %w", err)
		}
		name, ok := swiftName(d, "container", "", container)
		if !ok {
			continue
		}
		var depends []Ref
		var remaining int
		for object, err := range swiftRows(ctx, client, name, "X-Container-Object-Count") {
			if err != nil {
				return fmt.Errorf("list objects of container %s: %w", name, err)
			}
			objectName, ok := swiftName(d, "object", name, object)
			if !ok {
				remaining++
				continue
			}
			c := d.Row("object", object, objectName, objectName)
			c.Parent = name
			keep, err := d.Evaluate(c)
			if err != nil {
				return err
			}
			if !keep {
				remaining++
				continue
			}
			c.Reason = selectionReason(d)
			d.Add(c, 0)
			depends = append(depends, Ref{Service: "object_store", Kind: "object", ID: objectName, Parent: name})
		}
		if remaining > 0 {
			d.Exclude(Ref{Kind: "container", ID: name}, name, fmt.Sprintf("container keeps %d object(s) that are not cleanup candidates", remaining), container.raw)
			continue
		}
		c := d.Row("container", container, name, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if !keep {
			continue
		}
		c.Reason = selectionReason(d)
		if len(depends) > 0 {
			c.DependsOn = depends
			c.Reason += "; all objects are object candidates"
		}
		d.Add(c, 0)
	}
	return nil
}

// swiftName reads the name of a listing row. Rows without a name (for
// example pseudo-directory entries) cannot be addressed and are excluded.
func swiftName(d *discovery, kind, parent string, r row) (string, bool) {
	name, ok := r.str("name")
	if !ok || name == "" {
		subdir, _ := r.str("subdir")
		d.Exclude(Ref{Kind: kind, ID: subdir, Parent: parent}, subdir, "row has no usable name", r.raw)
		return "", false
	}
	return name, true
}

// swiftRows lists the account (container == "") or one container with
// format=json and marker paging. Like Python, paging ends when the count
// header shows that every row was read; it also ends on an empty page,
// which also covers a missing count header.
func swiftRows(ctx context.Context, client *gophercloud.ServiceClient, container, countHeader string) iter.Seq2[row, error] {
	return func(yield func(row, error) bool) {
		endpoint := client.ServiceURL()
		if container != "" {
			endpoint = client.ServiceURL(url.PathEscape(container))
		}
		marker, total := "", 0
		for {
			query := url.Values{"format": {"json"}}
			if marker != "" {
				query.Set("marker", marker)
			}
			response, err := rest.DoJSON(ctx, client, http.MethodGet, endpoint+"?"+query.Encode(), nil, nil, http.StatusOK, http.StatusNoContent)
			if err != nil {
				yield(row{}, err)
				return
			}
			var page []json.RawMessage
			if len(strings.TrimSpace(string(response.Body))) > 0 {
				if err := json.Unmarshal(response.Body, &page); err != nil {
					yield(row{}, response.Fail(fmt.Errorf("listing is not a JSON array: %w", err)))
					return
				}
			}
			if len(page) == 0 {
				return
			}
			last := ""
			for _, raw := range page {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(raw, &fields); err != nil {
					yield(row{}, response.Fail(fmt.Errorf("listing row is not a JSON object: %w", err)))
					return
				}
				r := row{fields: fields, raw: append(json.RawMessage(nil), raw...)}
				if name, ok := r.str("name"); ok {
					last = name
				} else if subdir, ok := r.str("subdir"); ok {
					last = subdir
				}
				total++
				if !yield(r, nil) {
					return
				}
			}
			if count, err := strconv.Atoi(response.Header.Get(countHeader)); err == nil && total >= count {
				return
			}
			if last == "" || last <= marker {
				yield(row{}, response.Fail(fmt.Errorf("%w: listing marker does not advance after %q", resource.ErrPaginationCycle, marker)))
				return
			}
			marker = last
		}
	}
}

// swiftRemove sends one DELETE to the escaped container or object path. A
// 404 is returned as an error matching resource.ErrNotFound.
func swiftRemove(kind string) func(context.Context, *gophercloud.ServiceClient, Candidate) error {
	return func(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
		if c.ID == "" || kind == "object" && c.Parent == "" {
			return fmt.Errorf("%w: %s has no name", resource.ErrInvalidOption, c.Ref)
		}
		endpoint := client.ServiceURL(url.PathEscape(c.ID))
		reference := c.ID
		if kind == "object" {
			endpoint = client.ServiceURL(url.PathEscape(c.Parent), url.PathEscape(c.ID))
			reference = c.Parent + "/" + c.ID
		}
		_, err := rest.DoJSON(ctx, client, http.MethodDelete, endpoint, nil, nil, objectStoreDeleteCodes...)
		if err != nil && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return &resource.NotFoundError{Resource: kind, Reference: reference, Cause: err}
		}
		return err
	}
}
