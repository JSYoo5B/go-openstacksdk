# Floating IP detach and automatic attach helpers (round 2)

Available on `compute.Service` and, with lazily discovered services, on `Connection`.

## `DetachIPFromServer`

Matches `detach_ip_from_server`.

- Neutron backend: the floating IP is read through the cloud `get_floating_ip` search. An absent row or a row without `port_id` returns outcome `not_attached` with no mutation. Otherwise one `PUT floatingips/{id}` sends `{"floatingip":{"port_id":null}}` (outcome `detached`, `Detached` true). Any failure of that PUT, including 404, is a `*DetachFloatingIPError` and never falls back to Nova.
- Nova backend: a missing IP fails before any mutation; otherwise one `POST servers/{id}/action` sends `removeFloatingIp` with the row's address (outcome `nova_action_accepted`; Python returns None here, so `Detached` stays false).
- The server ID is required and validated, but the Neutron branch never checks that the IP belonged to that server, like the pinned helper. The mutation is sent at most once; cancellation never rolls back an accepted request.
- The branch follows the configured floating IP source (Python `_use_neutron_floating`), not the backend of the looked-up row. When the Neutron list returns 404 and the lookup falls back to normalized Nova rows, the Neutron branch still runs on that row: a falsy `port_id` returns `not_attached` with no mutation, a truthy one sends the Neutron PUT (its failure is a `*DetachFloatingIPError`), and a strict-mode row without `port_id` fails before any mutation. `Backend` is `neutron` and `Lookup.Backend`/`Lookup.FallbackError` show the fallback.
- A clean Neutron 404 from the lookup itself (for example a direct member GET) is recorded in `NeutronLookup`/`NeutronNotFound`; like Python, the helper then runs the Nova branch, which repeats the lookup into `Lookup`.

## `AddAutoIP`

Matches `add_auto_ip` for basic use.

- Reuse (default) calls `AvailableFloatingIP` for the server; `WithAddAutoIPReuse(false)` allocates with `CreateFloatingIP` passing the server, the wait flag and the whole timeout, then spends that budget.
- If the supplied server snapshot, or a refreshed server when the IP has a `port_id`, already shows the exact floating IPv4, nothing is attached.
- Otherwise the exact address is attached through `AddIPList`. With the Neutron floating source (`NeutronFloating`), a newly allocated IP skips this step, including an allocation that fell back to Nova after a Neutron 404 (`Backend` is then `nova`); with the Nova source an allocation is always attached.
- A reused row that `AvailableFloatingIP` obtained from Nova after a Neutron 404 under the Neutron source is attached as Python does, through the Neutron branch of `_attach_ip_to_server`: `network.FloatingIPs.AttachByID` lists the server's ports, narrows several ports to the NAT destination network (an error when none is known), takes the newest port's first IPv4 address and sends one `PUT floatingips/{id}` with that `port_id` and `fixed_ip_address`. A 404 from the port list or the PUT is ignored and recorded in `NeutronNotFound`; no Nova `addFloatingIp` follows. A server without ports, a missing NAT destination or any other PUT failure is returned. The update is in `NeutronAttach`. As in Python, the preceding Neutron allocation attempt whose 404 selected Nova is part of `AvailableFloatingIP`.
- With `WithAddAutoIPWait(true)` the server is polled (interval `min(5s, remaining)` by default, first poll immediate) until the address appears or the budget (default 60 seconds) ends with `*AddAutoIPTimeoutError`, which matches `context.DeadlineExceeded`. A non-positive remaining budget expires before the first poll. Plain HTTP calls are bounded only by `ctx`.
- On that timeout, an IP created by this call under the Neutron source (including the Nova fallback, deleted through the configured Neutron delete, where a 404 is a not-deleted result) is deleted with `DeleteFloatingIP` on a context that ignores parent cancellation, unless `WithAddAutoIPCleanup(false)`. The delete outcome is in `Cleanup`/`CleanupError` and joins the returned error. User cancellation triggers no cleanup. Mutations are never replayed.
- `InterfaceIP` is the interface IP computed from the returned server model (refreshed only when a refresh or wait occurred); empty corresponds to Python's None.

Differences: other attaches go through the explicit-address workflow (address lookup by list), so Go sends one extra read compared with Python's direct attach; dynamic Server/Resource inputs and configuration aliases are not accepted.

With a Connection, `WithServiceEnabled("network", false)` or `has_network: false` in clouds.yaml makes the helpers use the Nova lane even with the Neutron source, as Python's `_use_neutron_floating` does.

Behavior is covered by `connection_r2_cloud_net_ip_test.go` and, for the Neutron-to-Nova fallback edges, `connection_cloud_net_ip_fallback_test.go` and `connection_cloud_net_auto_ip_fallback_test.go`; the disabled network service by `connection_network_service_disable_test.go`.
