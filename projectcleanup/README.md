# Project cleanup

`projectcleanup` deletes the resources owned by the current Keystone project. It is the Go counterpart of Python's `project_cleanup` (`openstack/cloud/openstackcloud.py`), split into a discovery step and an execution step:

- `Connection.PlanProjectCleanup(ctx, options...)` lists resources and returns a `*projectcleanup.Plan`. It sends only reads.
- `Connection.ExecuteProjectCleanup(ctx, plan, options...)` deletes exactly the plan's candidates in stage order and returns a `*projectcleanup.Result`. It does not discover again.

## Usage

```go
plan, err := conn.PlanProjectCleanup(ctx,
	projectcleanup.WithCreatedBefore(time.Now().Add(-30*24*time.Hour)),
	projectcleanup.WithSkip("compute.server"),
)
if err != nil {
	// A discovery failure of one service still returns the partial plan;
	// err names the service (errors.As with *projectcleanup.ServiceError).
	return err
}
for _, stage := range plan.Stages {
	for _, c := range stage.Candidates {
		fmt.Println(stage.Service, stage.Kind, c.ID, c.Name, c.Reason, c.DependsOn)
	}
}
for _, e := range plan.Excluded {
	fmt.Println("kept", e.Kind, e.ID, e.Reason)
}
for _, u := range plan.Unavailable {
	fmt.Println("not examined", u.Service, u.Reason)
}

result, err := conn.ExecuteProjectCleanup(ctx, plan,
	projectcleanup.WithDeletionWait(5*time.Minute),
	projectcleanup.WithProgress(func(e projectcleanup.Event) error {
		log.Println(e.Outcome.Candidate.Ref, e.Outcome.State)
		return nil
	}),
)
for _, o := range result.Outcomes {
	fmt.Println(o.Candidate.Ref, o.State, o.Reason, o.Err)
}
```

`PlanProjectCleanup` fails with `ErrNoSelection` when no filter and no selector is given, unless `WithAllResources()` acknowledges that every deletable resource of the project becomes a candidate.

### Plan options

| Option | Behavior |
|---|---|
| `WithAllResources()` | Required when no filter or selector is given. |
| `WithCreatedBefore(t)` | A resource whose `created_at` is at or before `t` passes (Python's `created_at` filter). A missing, null or empty value excludes it; a value that is not an ISO 8601 string excludes it with a reason. A value without an offset is UTC. |
| `WithUpdatedBefore(t)` | The same rule for `updated_at`. Both filters must pass when both are given. |
| `WithSkip(names...)` | Python's skip names, for example `compute.server`, `network.port`, `block_storage.volume`. Unknown names are rejected. |
| `WithSelector(fn)` | Called with a copy of each resource that passed the time filters; `false` excludes it, an error fails the plan. |

### Execute options

| Option | Behavior |
|---|---|
| `WithParallelism(n)` | Up to `n` mutations of the same stage at once. Default 1 (sequential). Stages never overlap. |
| `WithDeletionWait(d)` | Budget of each absence wait (default 120 s, Python's `wait_for_delete` default). The context bounds the whole call. |
| `WithProgress(fn)` | Synchronous callback for every candidate state change. Calls never overlap and run on the goroutine that changed the state. A returned error stops new mutations like cancellation and is returned. |
| `WithAllowIncompletePlan()` | Executes a plan that has discovery errors. Without it such a plan is refused with `ErrIncompletePlan`. |

## Outcome states

| State | Meaning |
|---|---|
| `Deleted` | The mutation was accepted and the resource was then observed absent (404). |
| `Accepted` | The mutation was accepted; absence was not observed because the stage does not wait, or because the wait failed (`WaitErr`). |
| `AlreadyMissing` | The service answered 404 to the mutation. |
| `Failed` | The service rejected the mutation. `Err`, `StatusCode`, `Header` and `Body` keep the response. |
| `Uncertain` | The request was sent and no response was received. It is never retried; the remote work may or may not have happened. |
| `NotAttempted` | Nothing was sent: a dependency did not succeed, execution stopped, or a local error occurred before sending (`Err`). |

`ExecuteProjectCleanup` returns the complete `Result` together with an error wrapping `ErrIncomplete` when any candidate is `Failed` or `Uncertain`, is `NotAttempted` because of a local error, or has a `WaitErr`. A progress callback error and context cancellation are also returned.

## Safety rules

- The project is fixed at plan time from the recorded token scope (ID, name, domain). An unscoped token fails with `ErrUnscoped`; a configured but unverified project is not used.
- `ExecuteProjectCleanup` refuses a Connection whose current token project differs from the plan's (`ErrProjectMismatch`) and a plan that was built by hand or modified after planning (`ErrPlanModified`). Use selectors or skip names to change what is deleted, then plan again.
- Only project-scoped lists are used; no `all_tenants` or `all_projects` query is sent. A row that reports another owner is excluded.
- Each mutation is sent at most once. There is no re-read before a DELETE; a 404 is `AlreadyMissing`. An uncertain request is not replayed.
- Cancellation and a progress error stop new mutations only. Mutations already sent are reported with what was observed, and the service may continue the work; nothing is rolled back.
- A candidate whose dependency is `Failed`, `Uncertain`, `NotAttempted`, or `Accepted` with a failed wait is `NotAttempted`.
- Waits follow the Gophercloud `WaitFor` convention: a polling error is returned immediately, and the wait ends at its budget or at context cancellation.

## Coverage

Services are examined in Python's dependency order. A service without an endpoint is `NotInCatalog`, a service disabled by `has_<service>` is `Disabled`, and a service that Python cleans but this package does not implement yet is `NotYetSupported`; its resources are kept. Orchestration, load balancer and DNS are reported only when they have an endpoint.

| Service | Python cleans | This package |
|---|---|---|
| compute | servers (then waits for deletion), server groups without members | servers with absence waits, then server groups. Keypairs are not cleaned, as in Python. |
| block_storage | backups (dependents first), snapshots, volumes | not yet supported |
| network | VPN resources, floating IPs, security groups except `default`, router interfaces, routers, ports, subnets, networks | not yet supported |
| image | images owned by the project | not yet supported |
| object_store | objects, then empty containers | not yet supported |
| key_manager | containers, secrets | not yet supported |
| orchestration, load_balancer, dns | stacks; load balancers; zones and floating IP PTR records | not yet supported (phase 2) |

### Compute

- Servers come from `GET /servers/detail` and are filtered on `created` and `updated`. Their stage waits for each accepted deletion to be observed absent before server groups are sent.
- Python lists server groups after its servers are deleted and skips groups that still have members. Here a group whose members are all server candidates is a candidate that depends on those servers; a group with any other member is excluded. Server groups have no `created_at` or `updated_at` in Python's resource, so any time filter excludes them.
- As in Python, the `compute.server` skip name also skips server groups.

## Differences from Python

- Python's `dry_run=True` runs the same cleanup code without deleting, and `dry_run=False` discovers again while deleting. Here discovery happens once, in `PlanProjectCleanup`, and `ExecuteProjectCleanup` acts only on that plan.
- Python cleans everything when no filter is given; here `WithAllResources()` must say so.
- Python logs per-resource exceptions and returns nothing. Here every candidate has an `Outcome`, and failures are returned as errors.
- Python's `resource_evaluation_fn` replaces the time filters. Here the time filters apply first and the selector sees only resources that passed them.
- Python skips a filter it cannot evaluate (an unparsable time is logged and ignored, so the resource can still be deleted). Here an unparsable time excludes the resource.
- Python runs services concurrently in a thread pool along its dependency graph. Here stages run sequentially, with optional parallelism inside a stage.
- Python reports progress by putting resources on a `queue.Queue`. Here a synchronous callback receives each state change.

## Limitations

- Each wait is bounded by `WithDeletionWait` and by the caller's context; a deletion that a service completes after the wait ends stays `Accepted` with `WaitErr`.
- Only the services marked above are implemented. Plans report the others as `NotYetSupported`, never as cleaned.

## Planned (not implemented)

- A channel-based progress option, in addition to the synchronous callback.

## Implementation notes

Each service is a self-contained unit implementing `serviceCleanup` (`kinds` and `discover`) in its own file and registered in `implementations` (`registry.go`). The global execution order of every kind is `stageOrder`; Python's skip names are `skipNames`. Discovery runs services in `serviceOrder`, so a later service can read earlier candidates with `discovery.Candidate` and express cross-service dependencies through `Candidate.DependsOn`. Tests: [Connection fixtures](../connection_project_cleanup_test.go) and [time filter](timefilter_test.go).
