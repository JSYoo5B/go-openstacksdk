package projectcleanup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"
)

var (
	// ErrUnscoped reports authentication without a recorded project scope.
	ErrUnscoped = errors.New("project cleanup requires a project-scoped token")
	// ErrProjectMismatch reports a Connection whose current project differs
	// from the project recorded in the plan.
	ErrProjectMismatch = errors.New("current project differs from the plan's project")
	// ErrNoSelection reports a plan request with no filter and no selector
	// that did not acknowledge cleanup of all resources with WithAllResources.
	ErrNoSelection = errors.New("project cleanup without a filter or selector requires WithAllResources")
	// ErrIncompletePlan reports a plan with discovery errors. Execute refuses
	// such a plan unless WithAllowIncompletePlan is given.
	ErrIncompletePlan = errors.New("project cleanup plan has discovery errors")
	// ErrPlanModified reports a plan that was not produced by
	// PlanProjectCleanup or was changed after it was produced.
	ErrPlanModified = errors.New("project cleanup plan was not produced by PlanProjectCleanup or was modified")
	// ErrIncomplete reports an execution in which at least one candidate
	// Failed, was Uncertain, could not be attempted because of a local error,
	// or whose absence could not be observed after a required wait.
	ErrIncomplete = errors.New("project cleanup did not complete")
)

// Project is the Keystone project scope recorded at plan time.
type Project struct {
	ID, Name, DomainID, DomainName string
}

// Ref identifies one resource. Service and Kind use Python's names, for
// example "compute" and "server". Parent is set only for resources addressed
// below another resource.
type Ref struct {
	Service, Kind, ID, Parent string
}

// SkipName returns the Python skip name of the kind, for example "compute.server".
func (r Ref) SkipName() string { return r.Service + "." + r.Kind }

func (r Ref) String() string {
	if r.Parent != "" {
		return fmt.Sprintf("%s.%s %s/%s", r.Service, r.Kind, r.Parent, r.ID)
	}
	return fmt.Sprintf("%s.%s %s", r.Service, r.Kind, r.ID)
}

// Candidate is a resource selected for deletion. CreatedAt and UpdatedAt are
// the raw JSON values of the attributes the time filters read (nil when the
// row does not contain them). Raw is the complete list row as returned.
// DependsOn names candidates, possibly of other services, that must be
// handled first; a candidate is not sent when one of them did not succeed.
type Candidate struct {
	Ref
	Name                 string
	CreatedAt, UpdatedAt json.RawMessage
	Reason               string
	DependsOn            []Ref
	Raw                  json.RawMessage
}

// Clone returns an independent copy.
func (c Candidate) Clone() Candidate {
	c.CreatedAt = bytes.Clone(c.CreatedAt)
	c.UpdatedAt = bytes.Clone(c.UpdatedAt)
	c.DependsOn = slices.Clone(c.DependsOn)
	c.Raw = bytes.Clone(c.Raw)
	return c
}

// Exclusion is a discovered resource that will not be deleted, with the
// reason. Raw is the list row when one exists.
type Exclusion struct {
	Ref
	Name   string
	Reason string
	Raw    json.RawMessage
}

// UnavailableReason explains why a service was not examined.
type UnavailableReason string

const (
	// NotInCatalog: no endpoint for the service in the catalog or overrides.
	NotInCatalog UnavailableReason = "not in catalog"
	// Disabled: the service is disabled by a has_<service> flag.
	Disabled UnavailableReason = "disabled"
	// NotYetSupported: the service has an endpoint and Python cleans it, but
	// this SDK does not implement its cleanup yet. Its resources are kept.
	NotYetSupported UnavailableReason = "not yet supported"
)

// Unavailable reports a service whose resources were not examined.
type Unavailable struct {
	Service string
	Reason  UnavailableReason
}

// ServiceError is a discovery failure of one service. The plan keeps no
// candidates or exclusions of a failed service.
type ServiceError struct {
	Service string
	Err     error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("project cleanup discovery of %s: %v", e.Service, e.Err)
}
func (e *ServiceError) Unwrap() error { return e.Err }

// Stage is one ordered step of a plan. All candidates of a stage are of the
// same kind. When WaitForAbsence is set, Execute observes each accepted
// deletion of the stage as absent before the next stage starts.
type Stage struct {
	Service, Kind  string
	Level          int
	WaitForAbsence bool
	Candidates     []Candidate
}

// Plan is the immutable result of discovery. Execute refuses a plan that was
// modified after PlanProjectCleanup returned it.
type Plan struct {
	Project        Project
	Stages         []Stage
	Excluded       []Exclusion
	Unavailable    []Unavailable
	Errors         []*ServiceError
	Skip           []string
	CreatedBefore  *time.Time
	UpdatedBefore  *time.Time
	AllResources   bool
	SelectorActive bool
	seal           [32]byte
}

// Candidates returns copies of all candidates in execution order.
func (p *Plan) Candidates() []Candidate {
	if p == nil {
		return nil
	}
	var out []Candidate
	for _, stage := range p.Stages {
		for _, candidate := range stage.Candidates {
			out = append(out, candidate.Clone())
		}
	}
	return out
}

// State is the result of one candidate.
type State string

const (
	// Deleted: the mutation was accepted and absence was then observed.
	Deleted State = "deleted"
	// Accepted: the mutation was accepted; absence was not observed, either
	// because the stage does not wait or because the wait failed (WaitErr).
	Accepted State = "accepted"
	// AlreadyMissing: the service answered 404 to the mutation.
	AlreadyMissing State = "already_missing"
	// Failed: the service rejected the mutation; Err, StatusCode, Header and
	// Body keep the response.
	Failed State = "failed"
	// Uncertain: the request was sent but no response was received. It is
	// never retried and the remote work may or may not have happened.
	Uncertain State = "uncertain"
	// NotAttempted: nothing was sent, because a dependency did not succeed,
	// execution stopped, or a local error occurred before sending (Err).
	NotAttempted State = "not_attempted"
)

// Outcome is the result of one candidate.
type Outcome struct {
	Candidate  Candidate
	State      State
	Reason     string
	Err        error
	StatusCode int
	Header     http.Header
	Body       []byte
	// WaitErr is the error of a required absence wait after an accepted
	// mutation. Dependents of such a candidate are NotAttempted.
	WaitErr error
}

func (o Outcome) clone() Outcome {
	o.Candidate = o.Candidate.Clone()
	o.Header = o.Header.Clone()
	o.Body = bytes.Clone(o.Body)
	return o
}

// Event reports one candidate state change to a WithProgress callback.
type Event struct {
	Outcome Outcome
}

// Result has one Outcome per plan candidate, in plan order.
type Result struct {
	Project  Project
	Outcomes []Outcome
}

// Count returns the number of outcomes in the state.
func (r *Result) Count(state State) int {
	if r == nil {
		return 0
	}
	n := 0
	for _, outcome := range r.Outcomes {
		if outcome.State == state {
			n++
		}
	}
	return n
}
