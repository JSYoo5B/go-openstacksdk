package projectcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// blockStorageCleanup follows openstack/block_storage/v3/_proxy.py
// _service_cleanup: backups (dependent backups first, waiting for each
// round's deletions), then snapshots (waiting for their deletion), then
// volumes. Python declares no cross-service dependency for block storage;
// compute is discovered and deleted first by the shared service order.
type blockStorageCleanup struct{}

func (blockStorageCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "backup", createdField: "created_at", updatedField: "updated_at", wait: true,
			remove: restRemove("backups", "backup", 202), absent: restAbsent("backups", "backup")},
		{kind: "snapshot", createdField: "created_at", updatedField: "updated_at", wait: true,
			remove: restRemove("snapshots", "snapshot", 202), absent: restAbsent("snapshots", "snapshot")},
		// Python does not wait for volume deletion.
		{kind: "volume", createdField: "created_at", updatedField: "updated_at",
			remove: restRemove("volumes", "volume", 202)},
	}
}

func (blockStorageCleanup) discover(ctx context.Context, d *discovery) error {
	if !d.Skipped("backup") {
		if err := discoverBackups(ctx, d); err != nil {
			return err
		}
	}
	// snapshotsOf maps a volume ID to its snapshot candidates.
	snapshotsOf := map[string][]Ref{}
	if !d.Skipped("snapshot") {
		project := d.Project().ID
		for r, err := range listRows(ctx, d.Client(), "snapshots/detail", "snapshots", nil) {
			if err != nil {
				return fmt.Errorf("list snapshots: %w", err)
			}
			id, name, ok := rowIdentity(d, "snapshot", r)
			if !ok {
				continue
			}
			if reason, foreign := ownerMismatch(r, project, "os-extended-snapshot-attributes:project_id", "project_id"); foreign {
				d.Exclude(Ref{Kind: "snapshot", ID: id}, name, reason, r.raw)
				continue
			}
			c := d.Row("snapshot", r, id, name)
			keep, err := d.Evaluate(c)
			if err != nil {
				return err
			}
			if !keep {
				continue
			}
			c.Reason = selectionReason(d)
			d.Add(c, 0)
			if volume, ok := r.str("volume_id"); ok && volume != "" {
				snapshotsOf[volume] = append(snapshotsOf[volume], Ref{Service: "block_storage", Kind: "snapshot", ID: id})
			}
		}
	}
	if d.Skipped("volume") {
		return nil
	}
	project := d.Project().ID
	for r, err := range listRows(ctx, d.Client(), "volumes/detail", "volumes", nil) {
		if err != nil {
			return fmt.Errorf("list volumes: %w", err)
		}
		id, name, ok := rowIdentity(d, "volume", r)
		if !ok {
			continue
		}
		if reason, foreign := ownerMismatch(r, project, "os-vol-tenant-attr:tenant_id", "project_id"); foreign {
			d.Exclude(Ref{Kind: "volume", ID: id}, name, reason, r.raw)
			continue
		}
		c := d.Row("volume", r, id, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if !keep {
			continue
		}
		c.Reason = selectionReason(d)
		// Python deletes every volume without checking snapshots or
		// attachments. A volume is not sent while one of its snapshot
		// candidates or attached server candidates did not succeed, because
		// Cinder would reject it then.
		c.DependsOn = append(c.DependsOn, snapshotsOf[id]...)
		for _, server := range attachedServers(r) {
			ref := Ref{Service: "compute", Kind: "server", ID: server}
			if _, ok := d.Candidate(ref); ok {
				c.DependsOn = append(c.DependsOn, ref)
			}
		}
		d.Add(c, 0)
	}
	return nil
}

func attachedServers(r row) []string {
	var attachments []struct {
		ServerID string `json:"server_id"`
	}
	raw, ok := r.fields["attachments"]
	if !ok || json.Unmarshal(raw, &attachments) != nil {
		return nil
	}
	var servers []string
	for _, a := range attachments {
		if a.ServerID != "" {
			servers = append(servers, a.ServerID)
		}
	}
	return servers
}

// backupRow is a listed backup of the plan's project.
type backupRow struct {
	id, name    string
	row         row
	candidate   *Candidate
	volume      string
	incremental bool
	dependents  bool
	created     time.Time
	createdOK   bool
}

// backup resolution states.
const (
	backupPending = iota
	backupResolving
	backupReady
	backupKept
)

// discoverBackups lists backups as Python's deleting rounds do (detailed,
// newest first) and assigns levels instead of rounds: a backup without
// dependent backups is level 0, and a backup with dependent backups runs
// one level after all of its possible dependents. Cinder does not expose
// the parent of an incremental backup, so the possible dependents of a
// backup are the incremental backups of the same volume created after it.
func discoverBackups(ctx context.Context, d *discovery) error {
	project := d.Project().ID
	var backups []*backupRow
	query := url.Values{"sort_key": {"created_at"}, "sort_dir": {"desc"}}
	for r, err := range listRows(ctx, d.Client(), "backups/detail", "backups", query) {
		if err != nil {
			return fmt.Errorf("list backups: %w", err)
		}
		id, name, ok := rowIdentity(d, "backup", r)
		if !ok {
			continue
		}
		if reason, foreign := ownerMismatch(r, project, "os-backup-project-attr:project_id", "project_id"); foreign {
			d.Exclude(Ref{Kind: "backup", ID: id}, name, reason, r.raw)
			continue
		}
		b := &backupRow{id: id, name: name, row: r}
		b.volume, _ = r.str("volume_id")
		b.incremental = rowBool(r, "is_incremental")
		b.dependents = rowBool(r, "has_dependent_backups")
		if text, ok := r.str("created_at"); ok {
			if t, err := parseISOTime(text); err == nil {
				b.created, b.createdOK = t, true
			}
		}
		backups = append(backups, b)
		c := d.Row("backup", r, id, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if keep {
			c.Reason = selectionReason(d)
			b.candidate = &c
		}
	}
	state := map[*backupRow]int{}
	levels := map[*backupRow]int{}
	var resolve func(b *backupRow) bool
	resolve = func(b *backupRow) bool {
		switch state[b] {
		case backupReady:
			return true
		case backupKept, backupResolving:
			return false
		}
		if b.candidate == nil {
			state[b] = backupKept
			return false
		}
		if !b.dependents {
			state[b] = backupReady
			return true
		}
		state[b] = backupResolving
		keep := func(reason string) bool {
			state[b] = backupKept
			d.Exclude(Ref{Kind: "backup", ID: b.id}, b.name, reason, b.row.raw)
			return false
		}
		if !b.createdOK {
			return keep("backup has dependent backups, but its created_at cannot order them")
		}
		var possible []*backupRow
		for _, other := range backups {
			if other != b && b.volume != "" && other.volume == b.volume && other.incremental && (!other.createdOK || other.created.After(b.created)) {
				possible = append(possible, other)
			}
		}
		if len(possible) == 0 {
			return keep("backup has dependent backups that are not listed in the project")
		}
		level := 0
		var blocked []string
		for _, other := range possible {
			if !resolve(other) {
				blocked = append(blocked, other.id)
				continue
			}
			level = max(level, levels[other]+1)
			b.candidate.DependsOn = append(b.candidate.DependsOn, Ref{Service: "block_storage", Kind: "backup", ID: other.id})
		}
		if len(blocked) > 0 {
			b.candidate.DependsOn = nil
			return keep("backup has dependent backups, and these possible dependents are not cleanup candidates: " + strings.Join(blocked, ", "))
		}
		b.candidate.Reason += "; after its possible dependent backups"
		levels[b] = level
		state[b] = backupReady
		return true
	}
	for _, b := range backups {
		resolve(b)
	}
	for _, b := range backups {
		if state[b] == backupReady {
			d.Add(*b.candidate, levels[b])
		}
	}
	return nil
}

// rowBool reads a boolean field; absent, null and non-boolean values are
// false, as Python's truthiness check treats a missing attribute.
func rowBool(r row, key string) bool {
	var value bool
	if raw, ok := r.fields[key]; ok {
		_ = json.Unmarshal(raw, &value)
	}
	return value
}
