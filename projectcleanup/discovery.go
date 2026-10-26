package projectcleanup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies connects the workflow to a Connection. Connection supplies it;
// applications call Connection.PlanProjectCleanup and
// Connection.ExecuteProjectCleanup instead of building it.
type Dependencies struct {
	// Project returns the recorded project scope of the current token, or an
	// error wrapping ErrUnscoped.
	Project func(context.Context) (Project, error)
	// ServiceEnabled reports the has_<service> configuration of a catalog type.
	ServiceEnabled func(serviceType string) bool
	// Endpoint reports whether an override or catalog endpoint exists.
	Endpoint func(serviceType string) (bool, error)
	// Client returns the SDK service client for a Python service name.
	Client func(ctx context.Context, service string) (*gophercloud.ServiceClient, error)
}

func (d Dependencies) validate() error {
	if d.Project == nil || d.ServiceEnabled == nil || d.Endpoint == nil || d.Client == nil {
		return invalid("project cleanup dependencies are incomplete")
	}
	return nil
}

type pending struct {
	candidate Candidate
	level     int
}

// discovery collects one service's results. Implementations use its methods
// and never build Plan values themselves.
type discovery struct {
	project  Project
	options  *planOptions
	service  string
	client   *gophercloud.ServiceClient
	kinds    map[string]kindSpec
	previous map[Ref]Candidate
	found    []pending
	excluded []Exclusion
	index    map[Ref]bool
}

// Project returns the plan's project.
func (d *discovery) Project() Project { return d.project }

// Client returns this service's SDK client.
func (d *discovery) Client() *gophercloud.ServiceClient { return d.client }

// Skipped reports whether WithSkip names service.kind.
func (d *discovery) Skipped(kind string) bool {
	return slices.Contains(d.options.skip, d.service+"."+kind)
}

// Candidate looks up a candidate recorded by an earlier service or earlier
// in this service's discovery.
func (d *discovery) Candidate(ref Ref) (Candidate, bool) {
	if c, ok := d.previous[ref]; ok {
		return c.Clone(), true
	}
	for _, p := range d.found {
		if p.candidate.Ref == ref {
			return p.candidate.Clone(), true
		}
	}
	return Candidate{}, false
}

// Exclude records a resource that will not be deleted.
func (d *discovery) Exclude(ref Ref, name, reason string, raw json.RawMessage) {
	ref.Service = d.service
	d.excluded = append(d.excluded, Exclusion{Ref: ref, Name: name, Reason: reason, Raw: bytes.Clone(raw)})
}

// Evaluate applies the time filters and then the selector, recording an
// exclusion when the candidate is rejected. Kinds that Python deletes as a
// consequence of another decision (for example a network's subnets) call
// Add directly instead.
func (d *discovery) Evaluate(c Candidate) (bool, error) {
	c.Service = d.service
	spec := d.kinds[c.Kind]
	check := func(attribute, field string, raw json.RawMessage, cutoff *time.Time) (bool, string) {
		if cutoff == nil {
			return true, ""
		}
		if field == "" {
			return false, attribute + " is not available for " + c.Ref.SkipName()
		}
		return timeMatches(attribute, raw, *cutoff)
	}
	if ok, reason := check("created_at", spec.createdField, c.CreatedAt, d.options.createdBefore); !ok {
		d.Exclude(c.Ref, c.Name, reason, c.Raw)
		return false, nil
	}
	if ok, reason := check("updated_at", spec.updatedField, c.UpdatedAt, d.options.updatedBefore); !ok {
		d.Exclude(c.Ref, c.Name, reason, c.Raw)
		return false, nil
	}
	if d.options.selector != nil {
		keep, err := d.options.selector(c.Clone())
		if err != nil {
			return false, &selectorError{fmt.Errorf("project cleanup selector for %s: %w", c.Ref, err)}
		}
		if !keep {
			d.Exclude(c.Ref, c.Name, "rejected by selector", c.Raw)
			return false, nil
		}
	}
	return true, nil
}

// Add records a candidate in the stage of its kind. Level orders several
// stages of the same kind (0 first).
func (d *discovery) Add(c Candidate, level int) {
	c.Service = d.service
	d.found = append(d.found, pending{candidate: c.Clone(), level: level})
}

// Row fills a candidate from a raw row using the kind's time fields.
func (d *discovery) Row(kind string, r row, id, name string) Candidate {
	spec := d.kinds[kind]
	c := Candidate{Ref: Ref{Service: d.service, Kind: kind, ID: id}, Name: name, Raw: bytes.Clone(r.raw)}
	if spec.createdField != "" {
		c.CreatedAt = bytes.Clone(r.fields[spec.createdField])
	}
	if spec.updatedField != "" {
		c.UpdatedAt = bytes.Clone(r.fields[spec.updatedField])
	}
	return c
}

// Discover builds a plan. It sends only reads.
func Discover(ctx context.Context, deps Dependencies, options ...PlanOption) (*Plan, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	o, err := parsePlanOptions(options)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	project, err := deps.Project(ctx)
	if err != nil {
		return nil, err
	}
	if project.ID == "" {
		return nil, ErrUnscoped
	}
	plan := &Plan{Project: project, Skip: slices.Clone(o.skip), CreatedBefore: o.createdBefore, UpdatedBefore: o.updatedBefore,
		AllResources: o.all, SelectorActive: o.selector != nil}
	all := map[Ref]Candidate{}
	levels := map[Ref]int{}
	var ordered []Ref
	var errs []error
	for _, info := range serviceOrder {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		present, err := servicePresent(deps, info)
		if err != nil {
			plan.Errors = append(plan.Errors, &ServiceError{Service: info.name, Err: err})
			continue
		}
		if present == "" {
			continue
		}
		if present != "present" {
			if info.core {
				plan.Unavailable = append(plan.Unavailable, Unavailable{Service: info.name, Reason: UnavailableReason(present)})
			}
			continue
		}
		impl, ok := implementations[info.name]
		if !ok {
			plan.Unavailable = append(plan.Unavailable, Unavailable{Service: info.name, Reason: NotYetSupported})
			continue
		}
		client, err := deps.Client(ctx, info.name)
		if err != nil {
			plan.Errors = append(plan.Errors, &ServiceError{Service: info.name, Err: err})
			continue
		}
		d := &discovery{project: project, options: o, service: info.name, client: client, kinds: map[string]kindSpec{}, previous: all}
		for _, spec := range impl.kinds() {
			d.kinds[spec.kind] = spec
		}
		if err := impl.discover(ctx, d); err != nil {
			var selectorErr *selectorError
			if errors.As(err, &selectorErr) || ctx.Err() != nil {
				return nil, err
			}
			plan.Errors = append(plan.Errors, &ServiceError{Service: info.name, Err: err})
			continue
		}
		plan.Excluded = append(plan.Excluded, d.excluded...)
		for _, p := range d.found {
			if _, known := d.kinds[p.candidate.Kind]; !known {
				return nil, fmt.Errorf("%w: undeclared kind %s", resource.ErrUnsupported, p.candidate.Ref.SkipName())
			}
			if _, duplicate := all[p.candidate.Ref]; duplicate {
				return nil, fmt.Errorf("%w: duplicate candidate %s", resource.ErrUnsupported, p.candidate.Ref)
			}
			all[p.candidate.Ref] = p.candidate
			levels[p.candidate.Ref] = p.level
			ordered = append(ordered, p.candidate.Ref)
		}
	}
	if err := buildStages(plan, ordered, all, levels); err != nil {
		return nil, err
	}
	plan.seal = sealPlan(plan)
	for _, e := range plan.Errors {
		errs = append(errs, e)
	}
	return plan, errors.Join(errs...)
}

// servicePresent returns "present", a core UnavailableReason, or "" for an
// absent non-core service.
func servicePresent(deps Dependencies, info serviceInfo) (string, error) {
	if !deps.ServiceEnabled(info.types[0]) {
		if info.core {
			return string(Disabled), nil
		}
		return "", nil
	}
	for _, serviceType := range info.types {
		ok, err := deps.Endpoint(serviceType)
		if err != nil {
			return "", fmt.Errorf("endpoint lookup for %s: %w", serviceType, err)
		}
		if ok {
			return "present", nil
		}
	}
	if info.core {
		return string(NotInCatalog), nil
	}
	return "", nil
}

func stageRank(name string) int { return slices.Index(stageOrder, name) }

// buildStages groups candidates by stage, keeping discovery (list) order
// within a stage.
func buildStages(plan *Plan, ordered []Ref, all map[Ref]Candidate, levels map[Ref]int) error {
	type key struct{ rank, level int }
	groups := map[key][]Candidate{}
	for _, ref := range ordered {
		c := all[ref]
		rank := stageRank(ref.SkipName())
		if rank < 0 {
			return fmt.Errorf("%w: kind %s has no stage order", resource.ErrUnsupported, ref.SkipName())
		}
		groups[key{rank, levels[ref]}] = append(groups[key{rank, levels[ref]}], c)
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].rank != keys[j].rank {
			return keys[i].rank < keys[j].rank
		}
		return keys[i].level < keys[j].level
	})
	position := map[Ref]key{}
	for _, k := range keys {
		for _, c := range groups[k] {
			position[c.Ref] = k
		}
	}
	for _, k := range keys {
		candidates := groups[k]
		first := candidates[0]
		spec := implementations[first.Service]
		var wait bool
		for _, s := range spec.kinds() {
			if s.kind == first.Kind {
				wait = s.wait
			}
		}
		for _, c := range candidates {
			for _, dep := range c.DependsOn {
				p, ok := position[dep]
				if !ok {
					return fmt.Errorf("%w: %s depends on %s, which is not a candidate", resource.ErrUnsupported, c.Ref, dep)
				}
				if p.rank > k.rank || p.rank == k.rank && p.level >= k.level {
					return fmt.Errorf("%w: %s depends on %s in a later stage", resource.ErrUnsupported, c.Ref, dep)
				}
			}
		}
		plan.Stages = append(plan.Stages, Stage{Service: first.Service, Kind: first.Kind, Level: k.level, WaitForAbsence: wait, Candidates: candidates})
	}
	return nil
}

type sealedPlan struct {
	Project                      Project
	Stages                       []Stage
	Excluded                     []Exclusion
	Unavailable                  []Unavailable
	Errors                       []string
	Skip                         []string
	CreatedBefore, UpdatedBefore *time.Time
	AllResources, SelectorActive bool
}

func sealPlan(p *Plan) [32]byte {
	value := sealedPlan{Project: p.Project, Stages: p.Stages, Excluded: p.Excluded, Unavailable: p.Unavailable, Skip: p.Skip,
		CreatedBefore: p.CreatedBefore, UpdatedBefore: p.UpdatedBefore, AllResources: p.AllResources, SelectorActive: p.SelectorActive}
	for _, e := range p.Errors {
		if e == nil {
			value.Errors = append(value.Errors, "<nil>")
			continue
		}
		value.Errors = append(value.Errors, e.Service+": "+fmt.Sprint(e.Err))
	}
	data, err := json.Marshal(value)
	if err != nil {
		return [32]byte{}
	}
	return sha256.Sum256(data)
}

type selectorError struct{ err error }

func (e *selectorError) Error() string { return e.err.Error() }
func (e *selectorError) Unwrap() error { return e.err }
