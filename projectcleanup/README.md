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
| block_storage | backups (dependents first), snapshots, volumes | backups by dependency level (waits), snapshots (wait), then volumes after their snapshot and attached-server candidates. |
| network | VPN resources, floating IPs, security groups except `default`, router interfaces, routers, ports, subnets, networks | VPN resources, floating IPs, security groups except `default`, router routes and interfaces, routers, ports, subnets, networks, without waits. |
| image | images owned by the project | project-owned images at any visibility (owner query), one DELETE each without waits. |
| object_store | objects, then empty containers | objects, then containers whose objects are all candidates, one DELETE each without waits. |
| key_manager | containers, secrets | containers, then secrets, without waits or dependencies. Orders are not cleaned, as in Python. |
| orchestration, load_balancer, dns | stacks; load balancers; zones and floating IP PTR records | not yet supported (phase 2) |

### Compute

- Servers come from `GET /servers/detail` and are filtered on `created` and `updated`. Their stage waits for each accepted deletion to be observed absent before server groups are sent.
- Python lists server groups after its servers are deleted and skips groups that still have members. Here a group whose members are all server candidates is a candidate that depends on those servers; a group with any other member is excluded. Server groups have no `created_at` or `updated_at` in Python's resource, so any time filter excludes them.
- As in Python, the `compute.server` skip name also skips server groups.

### Block storage

Follows `openstack/block_storage/v3/_proxy.py` `_service_cleanup`. Kinds, in execution order: `block_storage.backup`, `block_storage.snapshot`, `block_storage.volume`. Each skip name skips only its own kind and its list.

- Lists are project-scoped (`backups/detail` sorted by `created_at` descending as Python requests, `snapshots/detail`, `volumes/detail`); a row whose `os-backup-project-attr:project_id`, `os-extended-snapshot-attributes:project_id` or `os-vol-tenant-attr:tenant_id` names another project is excluded.
- Time filters read `created_at` and `updated_at` for all three kinds.
- Backups: Python deletes backups without dependent backups in up to ten rounds. The plan uses levels instead. A backup whose `has_dependent_backups` is false (or absent) is level 0. Cinder does not expose an incremental backup's parent, so the possible dependents of a backup with dependent backups are the incremental backups of the same volume created after it; the backup runs one level after all of them and depends on them. It is excluded when one of them is not a candidate, when none is listed, or when its own `created_at` cannot order them. Every backup level waits for absence before the next level starts.
- Snapshots: deleted after all backup levels; the stage waits for absence.
- Volumes: deleted last without a wait, as in Python. A volume depends on its snapshot candidates and on compute server candidates named in its `attachments`, so it is not sent while one of them did not succeed. Python attempts such volumes anyway; other volumes, including attached or snapshotted ones whose blockers are not candidates, are attempted as in Python and Cinder decides.
- Deletions send one `DELETE` each (202 expected); a 404 is AlreadyMissing. Wait polling errors are reported immediately, whereas Python ignores them.

### Network

- Order: VPN ipsec site connections, VPN services, endpoint groups, IKE policies, IPsec policies, floating IPs, security groups, router routes, router interfaces, routers, ports, subnets, networks. No stage waits; Neutron deletes are synchronous.
- VPN resources are examined only when `GET /service-providers?service_type=VPN` returns a provider. Python's VPN resources have no `created_at`/`updated_at`, so a time filter excludes them. A VPN service, endpoint group or policy still used by a site connection that is not a candidate is excluded; one used by candidate connections depends on them.
- Floating IPs follow Python's `fip_cleanup_evaluation`, which replaces the time filters: without a time filter every floating IP is a candidate; with one, an attached floating IP is kept and a detached one is a candidate whatever its times. The selector is still consulted.
- Security groups named `default` are kept; the others are filtered and selected normally.
- Networks are examined only when none of `network.network`, `network.router`, `network.port` and `network.subnet` is skipped. Shared and external networks are excluded. A network is kept when one of its project ports is in use by anything other than a compute server candidate; DHCP ports and ports without a device owner do not count, and router interface ports are removed. A remaining network is filtered and selected; then, without further evaluation, its interface routers' extra routes are removed (`PUT /routers/{id}/remove_extraroutes`), its interfaces are removed (`PUT /routers/{id}/remove_router_interface`), ports without a device owner and ports of server candidates are deleted, then its subnets and the network. A router whose interface cannot be read or belongs to another project keeps the network.
- A router whose interfaces are all removed is deleted when its only other ports are gateway, HA or SNAT ports; otherwise it is excluded. Routers of the project without any port are deleted. As in Python, neither case evaluates the time filters or the selector.
- Dependencies: interfaces depend on their router's route removal; routers on their routes and interfaces; ports of servers on the compute server candidate; subnets on the network's interfaces and ports; the network on all of them and its subnets.

### Image

Follows `openstack/image/v2/_proxy.py` `_service_cleanup`. Kind: `image` (skip name `image.image`).

- Discovery sends `GET /v2/images?owner=<project ID>`, the only query Python sends, and follows Glance's `next` link. Glance therefore lists only images whose owner is the project, at any visibility (private, shared, community or public). It does not list images that other projects own or hidden images (`os_hidden=true`).
- A listed row whose `owner` is not the plan's project (another project, missing or null) is excluded with a reason. Rows without a usable ID are excluded.
- Each remaining image goes through the `created_at`/`updated_at` filters and then the selector.
- Execution runs after all compute, block storage and network stages and before object store. It sends one `DELETE /v2/images/{id}` per candidate and does not wait for absence: Python does not wait, and Glance deletes synchronously. A 404 is `already_missing`. A refusal such as 403 for a protected image is `failed` with the response evidence. A lost response is `uncertain` and is never retried.
- Image candidates have no dependencies; Python declares only `before: identity`. An image still used by a server or volume is not held back; Glance decides whether the DELETE succeeds.

### Object store

Kinds, in stage order: `object_store.object`, then `object_store.container`. Neither stage waits for absence.

Discovery lists the containers of the token's account and, for each container, its objects (`format=json`, marker paging that ends when the `X-Account-Container-Count` / `X-Container-Object-Count` header is reached or on an empty page). Every object is evaluated by the filters and the selector. A container is evaluated only when every object in it became a candidate (or it has none); otherwise it is excluded with the number of objects it keeps. A selected container depends on all of its object candidates, so it is not sent when any of them Failed, was Uncertain or was not attempted.

Filters follow Python: an object's `updated_at` is the listing's `last_modified` (read as UTC when it has no offset), and objects have no `created_at`. Containers have neither attribute, so any time filter keeps every container.

Either skip name, `object_store.object` or `object_store.container`, skips the whole service, as in Python. Object refs carry the container in `Ref.Parent`; container and object names are path-escaped. Each object and container receives one DELETE (Python's bulk-delete is not used), and a 404 is reported as AlreadyMissing. Swift listings have no owner field: ownership comes from the token's account endpoint.

### Key manager

- Python's `_service_cleanup` deletes every container, then every secret, and declares no cleanup dependencies. Here containers are one stage and secrets the next. Neither stage waits for absence, because Barbican deletes synchronously and Python does not wait. No secret depends on a container, so a failed or uncertain container deletion does not hold back any secret, as in Python.
- Containers come from `GET /v1/containers` and secrets from `GET /v1/secrets`. The time filters use `created` and `updated`. Lists follow Barbican's top-level `next` offset links. No `acl_only` or all-projects query is sent, so only the token project's containers and secrets are listed. A row that reports another `project_id` is excluded.
- The ID is the last path component of `container_ref` or `secret_ref`, which is Python's `HREFToUUID` formatter behind `container_id` and `secret_id`. The DELETE goes to `containers/<id>` or `secrets/<id>` on the key manager endpoint. A row whose reference is missing, null, or not an absolute URL with a usable last segment is excluded. (Python's `Resource.id` is the raw href and its delete URL appends the whole href to the base path.)
- Orders are not cleaned, as in Python. The skip names are `key_manager.container` and `key_manager.secret`.

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
