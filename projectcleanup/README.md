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

Services are examined in Python's dependency order. A service without an endpoint is `NotInCatalog`, a service disabled by `has_<service>` is `Disabled`, and a service that Python cleans but this package does not implement would be `NotYetSupported` with its resources kept. Every service that Python's `project_cleanup` cleans is implemented, so current plans do not report `NotYetSupported`. Orchestration, load balancer and DNS are examined only when they have an endpoint and are otherwise not reported.

| Service | Python cleans | This package |
|---|---|---|
| compute | servers (then waits for deletion), server groups without members | servers with absence waits, then server groups. Keypairs are not cleaned, as in Python. |
| block_storage | backups (dependents first), snapshots, volumes | backups by dependency level (waits), snapshots (wait), then volumes after their snapshot and attached-server candidates. |
| network | VPN resources, floating IPs, security groups except `default`, router interfaces, routers, ports, subnets, networks | VPN resources, floating IPs, security groups except `default`, router routes and interfaces, routers, ports, subnets, networks, without waits. |
| image | images owned by the project | project-owned images at any visibility (owner query), one DELETE each without waits. |
| object_store | objects, then empty containers | objects, then containers whose objects are all candidates, one DELETE each without waits. |
| key_manager | containers, secrets | containers, then secrets, without waits or dependencies. Orders are not cleaned, as in Python. |
| orchestration | stacks (then waits for each deletion) | top-level stacks before compute and network, one DELETE each on the stack's name/ID route, with absence waits. |
| load_balancer | load balancers with cascade (then waits for each deletion) | load balancers after orchestration and before compute and network, one cascade DELETE each, with absence waits. |
| dns | zones, then floating IP PTR records (unset), without waits | zones, then floating IP PTR records after load balancers and before compute and network, one DELETE or PATCH each, without waits; a network floating IP depends on its PTR record candidate. |

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
- Floating IPs follow Python's `fip_cleanup_evaluation`, which replaces the time filters: without a time filter every floating IP is a candidate; with one, an attached floating IP is kept and a detached one is a candidate whatever its times. The selector is still consulted. A floating IP whose PTR record is a `dns.floating_ip` candidate depends on it (see DNS).
- Security groups named `default` are kept; the others are filtered and selected normally.
- Networks are examined only when none of `network.network`, `network.router`, `network.port` and `network.subnet` is skipped. Shared and external networks are excluded. A network is kept when one of its project ports is in use by anything other than a compute server candidate or a load balancer candidate's Octavia VIP port; DHCP ports and ports without a device owner do not count, and router interface ports are removed. A VIP port (`device_owner` `Octavia`, `device_id` `lb-<id>`) of a load balancer candidate is not planned, because Octavia deletes it with the load balancer (see Load balancer). A remaining network is filtered and selected; then, without further evaluation, its interface routers' extra routes are removed (`PUT /routers/{id}/remove_extraroutes`), its interfaces are removed (`PUT /routers/{id}/remove_router_interface`), ports without a device owner and ports of server candidates are deleted, then its subnets and the network. A router whose interface cannot be read or belongs to another project keeps the network.
- A router whose interfaces are all removed is deleted when its only other ports are gateway, HA or SNAT ports; otherwise it is excluded. Routers of the project without any port are deleted. As in Python, neither case evaluates the time filters or the selector.
- Dependencies: interfaces depend on their router's route removal; routers on their routes and interfaces; ports of servers on the compute server candidate; subnets on the network's interfaces and ports and on the load balancer candidates whose VIP ports are on the network; the network on all of them and its subnets.

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

### Orchestration

Follows `openstack/orchestration/v1/_proxy.py` `_service_cleanup`. Kind: `stack` (skip name `orchestration.stack`, which prevents listing).

- Python's cleanup dependencies run orchestration before compute, network and identity. Orchestration is discovered first and its stage runs before every compute and network stage, so Heat removes the resources it owns before those services are cleaned.
- Discovery sends `GET /stacks` without a query, as Python does. Heat's default list contains only the project's top-level stacks; nested stacks are deleted by Heat with their parent and never become candidates. A row without a usable `stack_name` and `id` is excluded, and a row whose `project` names another project is excluded.
- The time filters read `creation_time` and `updated_time` (Python's `created_at` and `updated_at`). Heat reports `updated_time` as null for a stack that was never updated, so `WithUpdatedBefore` excludes such a stack. The selector sees the stack name as `Name`.
- Execution sends one `DELETE /stacks/{stack_name}/{id}` per candidate, the canonical route that Python reaches through Heat's redirect from `/stacks/{id}`. A 404 is `already_missing`; a refusal such as 409 is `failed` with the response evidence.
- After all DELETEs of the stage, each accepted stack is polled with `GET /stacks/{stack_name}/{id}` through the shared wait engine, as Python's `wait_for_delete` does. A 404 or `DELETE_COMPLETE` is absence (`deleted`). `DELETE_FAILED` ends the wait at once with an `ErrFailedState` wait error. Any other state, including the earlier action's `*_FAILED` state before Heat reports `DELETE_IN_PROGRESS`, keeps polling until `WithDeletionWait` ends.
- Compute and network candidates do not depend on stacks. Python's dependency walk also runs compute and network after a failed orchestration cleanup. A resource that Heat already removed with its stack is reported as `already_missing`.

### Load balancer

Follows `openstack/load_balancer/v2/_proxy.py` `_service_cleanup`. Kind: `load_balancer` (skip name `load_balancer.load_balancer`, which prevents listing).

- Python's cleanup dependencies run load_balancer before compute and network. The load balancer is discovered after orchestration and its stage runs after the stack stage and before every compute and network stage. Python does not order load_balancer against orchestration; stacks run first, so a load balancer that Heat removes with its stack is reported as `already_missing`.
- Discovery sends `GET /v2.0/lbaas/loadbalancers` without a query, as Python does, and follows `loadbalancers_links`. Octavia lists only the token project's load balancers unless the token may list every project; a row whose `project_id` or `tenant_id` names another project is excluded, and so is a row without `project_id` or without a usable `id`. Python deletes every listed row, including other projects' load balancers for such a token.
- The time filters read `created_at` and `updated_at`. Octavia reports `updated_at` as null for a load balancer that was never updated, so `WithUpdatedBefore` excludes it. Load balancers in any provisioning status are evaluated, as in Python.
- Execution sends one `DELETE /v2.0/lbaas/loadbalancers/{id}?cascade=true` per candidate, as Python's `delete_load_balancer(cascade=True)` does, so Octavia also removes the listeners, pools, members, health monitors and L7 policies. A 404 is `already_missing`; a refusal such as 409 for a load balancer in a `PENDING_*` state is `failed` with the response evidence; a lost response is `uncertain` and is never retried.
- After all DELETEs of the stage, each accepted load balancer is polled with `GET /v2.0/lbaas/loadbalancers/{id}` through the shared wait engine, as Python's `wait_for_delete` does. A 404 or `provisioning_status` `DELETED` is absence (`deleted`). `ERROR` ends the wait at once with an `ErrFailedState` wait error. `PENDING_DELETE` and other states keep polling until `WithDeletionWait` ends. Python treats only 404 as absence and only logs a failed wait.
- Compute candidates do not depend on load balancers, and network candidates depend on one only through its VIP port. Python's dependency walk also runs compute and network after a failed load balancer cleanup.
- Octavia creates the VIP port in the load balancer's project with `device_owner` `Octavia` and `device_id` `lb-<load balancer id>`, and deletes that port itself while deleting the load balancer. Python's real run lists networks only after its load balancers are deleted and waited for, so such a network can be cleaned in the same run. Discovery here runs once before any deletion, so the VIP port of a load balancer candidate is neither planned nor counted as a port in use; instead the network and its subnets depend on the load balancer and are `not_attempted` unless it is `deleted` or `already_missing` (for example after a failed DELETE or a wait that ended while it was still `PENDING_DELETE`). The VIP port of a load balancer that is not a candidate (another project's, filtered out, rejected by the selector or skipped) keeps its network, as in Python's dry run. Amphora VRRP ports are created in the Octavia service project, so the project-scoped port list does not return them.

### DNS

Follows `openstack/dns/v2/_proxy.py` `_service_cleanup`. Kinds, in stage order: `dns.zone`, then `dns.floating_ip`. Each skip name prevents its own list.

- Python's cleanup dependencies run dns before network ("DNS may depend on floating ip"). DNS is discovered after load balancer and before compute and network, and its stages run after the load balancer stage and before every compute and network stage. Python does not order dns against orchestration, load_balancer or compute.
- Zones come from `GET /v2/zones` without a query or scope header, following Designate's `links.next`. No `X-Auth-All-Projects` or `X-Auth-Sudo-Project-ID` header is ever sent, so Designate answers for the token project. Designate also lists zones that other projects share with this one; a row whose `project_id` names another project, a row without `project_id` and a row without a usable `id` are excluded. Python deletes every listed zone, including shared ones.
- Zone time filters read `created_at` and `updated_at`. Designate reports `updated_at` as null for a zone that was never updated, so `WithUpdatedBefore` excludes it. Zones in any status are evaluated, as in Python.
- Execution sends one `DELETE /v2/zones/{id}` per zone candidate without `X-Designate-Delete-Shares`, as Python's `delete_zone` does. Designate deletes the zone's recordsets with it, so recordsets are not separate candidates (Python does not list them either). Designate answers 202 and deletes asynchronously; the stage does not wait, as in Python, so a zone stays `accepted`.
- Floating IP PTR records come from `GET /v2/reverse/floatingips` without a query or scope header. Their ID is Designate's `<region>:<floatingip-id>`, and the selector sees the floating IP `address` as `Name`. Python's resource has no `created_at` or `updated_at`, so any time filter excludes every PTR record, matching Python's note that floating IPs are not cleaned when filters are set. A row that reports another `project_id` or has no usable `id` is excluded.
- Execution sends one `PATCH /v2/reverse/floatingips/{region}:{id}` with `{"ptrdname": null}` per PTR candidate, as Python's `unset_floating_ip` does, without a wait. Like Python, a listed row whose `ptrdname` is already null is still sent; Designate's answer decides the outcome. 200 and 202 are `accepted`, 404 is `already_missing`, a refusal is `failed` with the response evidence and a lost response is `uncertain`.
- A `network.floating_ip` candidate depends on the `dns.floating_ip` candidate whose ID ends with `:<floating IP ID>`. When the PTR unset is `failed`, `uncertain` or not attempted, the floating IP is `not_attempted`, so the PTR record is not orphaned by deleting its floating IP. Python unsets the PTR records first but deletes the floating IPs whatever the result. Zones have no dependents.

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
- Only the services and kinds listed above are cleaned. A service that Python cleans and this package did not implement would be reported as `NotYetSupported`, never as cleaned; no such service remains.

## Planned (not implemented)

- A channel-based progress option, in addition to the synchronous callback.

## Implementation notes

Each service is a self-contained unit implementing `serviceCleanup` (`kinds` and `discover`) in its own file and registered in `implementations` (`registry.go`). The global execution order of every kind is `stageOrder`; Python's skip names are `skipNames`. Discovery runs services in `serviceOrder`, so a later service can read earlier candidates with `discovery.Candidate` and express cross-service dependencies through `Candidate.DependsOn`. Tests: [Connection fixtures](../connection_project_cleanup_test.go) and [time filter](timefilter_test.go).
