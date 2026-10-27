package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"sort"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/ptr/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// dnsCleanup follows openstack/dns/v2/_proxy.py _service_cleanup: it lists
// the project's zones and deletes the selected ones (Designate removes their
// recordsets with them), then lists the project's floating IP PTR records
// and unsets each selected one with PATCH {"ptrdname": null}. Python does
// not wait for either mutation, and neither does this unit. Python's
// dependency graph runs dns before network ("DNS may depend on floating
// ip"), which stageOrder reproduces; a network floating IP candidate also
// depends on the PTR record candidate of the same floating IP.
//
// Lists send no X-Auth-All-Projects or X-Auth-Sudo-Project-ID header, so
// Designate answers for the token project only.
type dnsCleanup struct{}

func (dnsCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "zone", createdField: "created_at", updatedField: "updated_at", remove: removeZone},
		// The Python FloatingIP resource has neither created_at nor
		// updated_at, so a time filter excludes every PTR record ("NOTE:
		// FloatingIPs are not cleaned when filters are set").
		{kind: "floating_ip", remove: unsetFloatingIPPTR},
	}
}

func (dnsCleanup) discover(ctx context.Context, d *discovery) error {
	if !d.Skipped("zone") {
		if err := dnsZones(ctx, d); err != nil {
			return err
		}
	}
	if !d.Skipped("floating_ip") {
		if err := dnsFloatingIPs(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// dnsZones evaluates every zone of the unqueried zone list. Designate also
// lists zones that another project shares with this one; their project_id
// names the owner, so they are excluded like any other foreign row.
func dnsZones(ctx context.Context, d *discovery) error {
	project := d.Project().ID
	for r, err := range listDesignateZones(ctx, d.Client()) {
		if err != nil {
			return fmt.Errorf("list zones: %w", err)
		}
		id, name, ok := rowIdentity(d, "zone", r)
		if !ok {
			continue
		}
		ref := Ref{Kind: "zone", ID: id}
		if reason, foreign := ownerMismatch(r, project, "project_id"); foreign {
			d.Exclude(ref, name, reason, r.raw)
			continue
		}
		if owner, _ := r.str("project_id"); owner != project {
			d.Exclude(ref, name, "project_id is not reported", r.raw)
			continue
		}
		c := d.Row("zone", r, id, name)
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

// dnsFloatingIPs evaluates every row of the floating IP PTR list. Like
// Python, a row whose ptrdname is null is still unset; Designate decides
// the outcome of that PATCH. Rows carry no owner field; a row that reports
// another project_id is excluded.
func dnsFloatingIPs(ctx context.Context, d *discovery) error {
	for value, err := range floatingips.New(d.Client()).List(ctx) {
		if err != nil {
			return fmt.Errorf("list floating IP PTR records: %w", err)
		}
		raw, err := json.Marshal(value.Body)
		if err != nil {
			return fmt.Errorf("list floating IP PTR records: %w", err)
		}
		r := row{fields: value.Body, raw: raw}
		id, _ := r.str("id")
		name, _ := r.str("address")
		ref := Ref{Kind: "floating_ip", ID: id}
		if resource.ID(id).Validate() != nil {
			d.Exclude(ref, name, "row has no usable id", r.raw)
			continue
		}
		if reason, foreign := ownerMismatch(r, d.Project().ID, "project_id", "tenant_id"); foreign {
			d.Exclude(ref, name, reason, r.raw)
			continue
		}
		c := d.Row("floating_ip", r, id, name)
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

// listDesignateZones yields the raw rows of GET zones, following
// Designate's links.next continuation.
func listDesignateZones(ctx context.Context, client *gophercloud.ServiceClient) iter.Seq2[row, error] {
	spec := restSpec(client, "zones", "zones", "", "zones")
	spec.Paging = rest.PagePolicy[resource.RawResource]{DictionaryLinks: true, StopOnEmptyPage: true}
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

// removeZone sends one DELETE zones/{id} without X-Designate-Delete-Shares,
// as Python's delete_zone(delete_shares=False) does. Designate answers 202
// and deletes the zone asynchronously. A 404 is reported as
// resource.ErrNotFound.
func removeZone(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
	if err := resource.ID(c.ID).Validate(); err != nil {
		return err
	}
	_, err := zones.New(client).Delete(ctx, c.ID)
	if err != nil && resource.IgnoreMissing(err) == nil {
		return &resource.NotFoundError{Resource: "zones", Reference: c.ID, Cause: err}
	}
	return err
}

// unsetFloatingIPPTR sends one PATCH reverse/floatingips/{region}:{id} with
// {"ptrdname": null}, as Python's unset_floating_ip does. Designate answers
// 200 or 202; a 404 is reported as resource.ErrNotFound.
func unsetFloatingIPPTR(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
	_, err := floatingips.New(client).Unset(ctx, c.ID)
	return err
}

// dnsPTRDependencies maps a Neutron floating IP ID to the DNS floating_ip
// candidates of earlier services whose "<region>:<floatingip-id>" ID names
// it, so that the floating IP is not deleted while its PTR record could not
// be unset.
func dnsPTRDependencies(d *discovery) map[string][]Ref {
	out := map[string][]Ref{}
	for ref := range d.previous {
		if ref.Service != "dns" || ref.Kind != "floating_ip" {
			continue
		}
		if i := strings.LastIndex(ref.ID, ":"); i >= 0 && i+1 < len(ref.ID) {
			out[ref.ID[i+1:]] = append(out[ref.ID[i+1:]], ref)
		}
	}
	for _, refs := range out {
		sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	}
	return out
}
