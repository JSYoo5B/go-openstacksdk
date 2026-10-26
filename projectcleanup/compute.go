package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// computeCleanup follows openstack/compute/v2/_proxy.py _service_cleanup:
// servers (waiting for their deletion), then server groups without members.
// Python does not clean keypairs, and neither does this unit.
type computeCleanup struct{}

func (computeCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "server", createdField: "created", updatedField: "updated", wait: true,
			remove: restRemove("servers", "server", 204), absent: restAbsent("servers", "server")},
		// Python's ServerGroup has no created_at or updated_at attribute.
		{kind: "server_group", remove: restRemove("os-server-groups", "server_group", 204)},
	}
}

func (computeCleanup) discover(ctx context.Context, d *discovery) error {
	// Python's compute.server skip name also skips server groups.
	if d.Skipped("server") {
		return nil
	}
	project := d.Project().ID
	for r, err := range listRows(ctx, d.Client(), "servers/detail", "servers", nil) {
		if err != nil {
			return fmt.Errorf("list servers: %w", err)
		}
		id, name, ok := rowIdentity(d, "server", r)
		if !ok {
			continue
		}
		if reason, foreign := ownerMismatch(r, project, "tenant_id", "project_id"); foreign {
			d.Exclude(Ref{Kind: "server", ID: id}, name, reason, r.raw)
			continue
		}
		c := d.Row("server", r, id, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if keep {
			c.Reason = selectionReason(d)
			d.Add(c, 0)
		}
	}
	for r, err := range listRows(ctx, d.Client(), "os-server-groups", "server_groups", nil) {
		if err != nil {
			return fmt.Errorf("list server groups: %w", err)
		}
		id, name, ok := rowIdentity(d, "server_group", r)
		if !ok {
			continue
		}
		if reason, foreign := ownerMismatch(r, project, "project_id"); foreign {
			d.Exclude(Ref{Kind: "server_group", ID: id}, name, reason, r.raw)
			continue
		}
		var members []string
		if raw, present := r.fields["members"]; present && string(raw) != "null" {
			if err := json.Unmarshal(raw, &members); err != nil {
				d.Exclude(Ref{Kind: "server_group", ID: id}, name, "members cannot be read", r.raw)
				continue
			}
		}
		// Python lists server groups after its servers are gone and skips
		// groups that still have members. A group whose members are all
		// server candidates is empty at that point; any other member keeps it.
		var depends, kept []string
		for _, member := range members {
			if _, ok := d.Candidate(Ref{Service: "compute", Kind: "server", ID: member}); ok {
				depends = append(depends, member)
			} else {
				kept = append(kept, member)
			}
		}
		if len(kept) > 0 {
			d.Exclude(Ref{Kind: "server_group", ID: id}, name, "server group has members that are not cleanup candidates: "+strings.Join(kept, ", "), r.raw)
			continue
		}
		c := d.Row("server_group", r, id, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if !keep {
			continue
		}
		c.Reason = selectionReason(d)
		for _, member := range depends {
			c.DependsOn = append(c.DependsOn, Ref{Service: "compute", Kind: "server", ID: member})
		}
		if len(depends) > 0 {
			c.Reason += "; all members are server candidates"
		}
		d.Add(c, 0)
	}
	return nil
}

// rowIdentity reads id and name, excluding rows without a usable ID.
func rowIdentity(d *discovery, kind string, r row) (string, string, bool) {
	id, ok := r.str("id")
	name, _ := r.str("name")
	if !ok || resource.ID(id).Validate() != nil {
		d.Exclude(Ref{Kind: kind, ID: id}, name, "row has no usable id", r.raw)
		return "", "", false
	}
	return id, name, true
}

func selectionReason(d *discovery) string {
	var parts []string
	if d.options.createdBefore != nil {
		parts = append(parts, "created_at at or before cutoff")
	}
	if d.options.updatedBefore != nil {
		parts = append(parts, "updated_at at or before cutoff")
	}
	if d.options.selector != nil {
		parts = append(parts, "accepted by selector")
	}
	if len(parts) == 0 {
		return "all resources"
	}
	return strings.Join(parts, ", ")
}
