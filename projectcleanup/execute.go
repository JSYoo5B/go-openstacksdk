package projectcleanup

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Execute deletes the plan's candidates in stage order. It does not discover
// again and sends each mutation at most once.
func Execute(ctx context.Context, deps Dependencies, plan *Plan, options ...ExecuteOption) (*Result, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	o, err := parseExecuteOptions(options)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.seal == ([32]byte{}) || sealPlan(plan) != plan.seal {
		return nil, fmt.Errorf("%w: %w", resource.ErrInvalidOption, ErrPlanModified)
	}
	if len(plan.Errors) > 0 && !o.allowPartial {
		return nil, fmt.Errorf("%w: %w", ErrIncompletePlan, errors.Join(serviceErrors(plan)...))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	current, err := deps.Project(ctx)
	if err != nil {
		return nil, err
	}
	if current.ID != plan.Project.ID {
		return nil, fmt.Errorf("%w: plan project %s, current project %s", ErrProjectMismatch, plan.Project.ID, current.ID)
	}
	run := &execution{ctx: ctx, deps: deps, options: o, outcomes: map[Ref]*Outcome{}, clients: map[string]*gophercloud.ServiceClient{}, clientErrs: map[string]error{}}
	for _, stage := range plan.Stages {
		run.stage(stage)
	}
	result := &Result{Project: plan.Project}
	var failed, uncertain, local, waits int
	for _, stage := range plan.Stages {
		for _, c := range stage.Candidates {
			outcome := run.outcomes[c.Ref].clone()
			switch {
			case outcome.State == Failed:
				failed++
			case outcome.State == Uncertain:
				uncertain++
			case outcome.State == NotAttempted && outcome.Err != nil:
				local++
			case outcome.WaitErr != nil:
				waits++
			}
			result.Outcomes = append(result.Outcomes, outcome)
		}
	}
	var errs []error
	if failed+uncertain+local+waits > 0 {
		errs = append(errs, fmt.Errorf("%w: %d failed, %d uncertain, %d not attempted after a local error, %d without observed absence",
			ErrIncomplete, failed, uncertain, local, waits))
	}
	if run.stopErr != nil {
		errs = append(errs, run.stopErr)
	}
	if err := ctx.Err(); err != nil {
		errs = append(errs, err)
	}
	return result, errors.Join(errs...)
}

func serviceErrors(plan *Plan) []error {
	var errs []error
	for _, e := range plan.Errors {
		errs = append(errs, e)
	}
	return errs
}

type execution struct {
	ctx        context.Context
	deps       Dependencies
	options    *executeOptions
	mu         sync.Mutex // outcomes, stop state and clients
	progressMu sync.Mutex // serializes callbacks
	outcomes   map[Ref]*Outcome
	clients    map[string]*gophercloud.ServiceClient
	clientErrs map[string]error
	stopped    bool
	stopErr    error
}

func (e *execution) isStopped() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stopped || e.ctx.Err() != nil
}

// set records a state change and reports it. Callbacks are not called once
// execution has stopped.
func (e *execution) set(outcome Outcome) {
	e.mu.Lock()
	copy := outcome
	e.outcomes[outcome.Candidate.Ref] = &copy
	stopped := e.stopped
	e.mu.Unlock()
	if e.options.progress == nil || stopped {
		return
	}
	e.progressMu.Lock()
	defer e.progressMu.Unlock()
	if err := e.options.progress(Event{Outcome: outcome.clone()}); err != nil {
		e.mu.Lock()
		if !e.stopped {
			e.stopped = true
			e.stopErr = fmt.Errorf("project cleanup stopped by progress callback: %w", err)
		}
		e.mu.Unlock()
	}
}

func (e *execution) client(service string) (*gophercloud.ServiceClient, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if client, ok := e.clients[service]; ok {
		return client, nil
	}
	if err, ok := e.clientErrs[service]; ok {
		return nil, err
	}
	client, err := e.deps.Client(e.ctx, service)
	if err != nil {
		e.clientErrs[service] = err
		return nil, err
	}
	e.clients[service] = client
	return client, nil
}

// blocked returns why a dependency prevents sending.
func (e *execution) blocked(c Candidate) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, dep := range c.DependsOn {
		outcome := e.outcomes[dep]
		switch {
		case outcome == nil:
			return fmt.Sprintf("dependency %s has no outcome", dep)
		case outcome.State == Deleted || outcome.State == AlreadyMissing:
		case outcome.State == Accepted && outcome.WaitErr == nil:
		case outcome.State == Accepted:
			return fmt.Sprintf("absence of dependency %s was not observed", dep)
		default:
			return fmt.Sprintf("dependency %s is %s", dep, outcome.State)
		}
	}
	return ""
}

func (e *execution) spec(c Candidate) (kindSpec, bool) {
	impl, ok := implementations[c.Service]
	if !ok {
		return kindSpec{}, false
	}
	for _, spec := range impl.kinds() {
		if spec.kind == c.Kind {
			return spec, true
		}
	}
	return kindSpec{}, false
}

func (e *execution) parallel(items []Candidate, work func(Candidate)) {
	if e.options.parallelism <= 1 || len(items) <= 1 {
		for _, c := range items {
			work(c)
		}
		return
	}
	slots := make(chan struct{}, e.options.parallelism)
	var wg sync.WaitGroup
	for _, c := range items {
		slots <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			work(c)
		}()
	}
	wg.Wait()
}

func (e *execution) stage(stage Stage) {
	e.parallel(stage.Candidates, e.send)
	if !stage.WaitForAbsence {
		return
	}
	var accepted []Candidate
	e.mu.Lock()
	for _, c := range stage.Candidates {
		if o := e.outcomes[c.Ref]; o != nil && o.State == Accepted {
			accepted = append(accepted, c)
		}
	}
	e.mu.Unlock()
	e.parallel(accepted, e.wait)
}

func (e *execution) send(c Candidate) {
	outcome := Outcome{Candidate: c.Clone()}
	if e.isStopped() {
		outcome.State, outcome.Reason = NotAttempted, "execution stopped before sending"
		e.set(outcome)
		return
	}
	if reason := e.blocked(c); reason != "" {
		outcome.State, outcome.Reason = NotAttempted, reason
		e.set(outcome)
		return
	}
	spec, ok := e.spec(c)
	if !ok || spec.remove == nil {
		outcome.State, outcome.Reason = NotAttempted, "kind is not supported"
		outcome.Err = fmt.Errorf("%w: %s", resource.ErrUnsupported, c.Ref.SkipName())
		e.set(outcome)
		return
	}
	client, err := e.client(c.Service)
	if err != nil {
		outcome.State, outcome.Reason, outcome.Err = NotAttempted, "service client unavailable", err
		e.set(outcome)
		return
	}
	if err := e.ctx.Err(); err != nil {
		outcome.State, outcome.Reason = NotAttempted, "execution stopped before sending"
		e.set(outcome)
		return
	}
	err = spec.remove(e.ctx, client, c)
	classify(&outcome, err)
	e.set(outcome)
}

func (e *execution) wait(c Candidate) {
	if e.isStopped() {
		return
	}
	spec, _ := e.spec(c)
	client, err := e.client(c.Service)
	if err == nil && spec.absent == nil {
		err = fmt.Errorf("%w: absence of %s cannot be observed", resource.ErrUnsupported, c.Ref.SkipName())
	}
	if err == nil {
		err = spec.absent(e.ctx, client, c, e.options.wait, e.options.pollInterval)
	}
	e.mu.Lock()
	outcome := *e.outcomes[c.Ref]
	e.mu.Unlock()
	if err != nil {
		outcome.WaitErr = err
		e.mu.Lock()
		e.outcomes[c.Ref] = &outcome
		e.mu.Unlock()
		return
	}
	outcome.State = Deleted
	e.set(outcome)
}

// classify maps the single mutation's error to a state. A rejection keeps
// the response; an error without any response is Uncertain.
func classify(outcome *Outcome, err error) {
	outcome.Err = err
	if err == nil {
		outcome.State = Accepted
		return
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &unexpected) {
		outcome.StatusCode, outcome.Header, outcome.Body = unexpected.Actual, unexpected.ResponseHeader.Clone(), append([]byte(nil), unexpected.Body...)
	}
	var reauth gophercloud.ErrUnableToReauthenticate
	var response *resource.ResponseError
	switch {
	case errors.Is(err, resource.ErrNotFound):
		outcome.State = AlreadyMissing
	case outcome.StatusCode != 0:
		outcome.State = Failed
	case errors.As(err, &reauth):
		outcome.State = Failed
	case errors.As(err, &response) && response.StatusCode != 0:
		outcome.StatusCode, outcome.Header, outcome.Body = response.StatusCode, response.Header.Clone(), append([]byte(nil), response.Body...)
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			outcome.State = Accepted
		} else {
			outcome.State = Failed
		}
	case errors.Is(err, resource.ErrInvalidOption):
		outcome.State, outcome.Reason = NotAttempted, "rejected locally before sending"
	default:
		outcome.State = Uncertain
	}
}
