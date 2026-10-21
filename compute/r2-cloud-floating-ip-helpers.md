# Floating IP detach and automatic attach helpers (round 2)

Available on `compute.Service` and, with lazily discovered services, on `Connection`.

## `DetachIPFromServer`

Matches `detach_ip_from_server`.

- Neutron backend: the floating IP is read through the cloud `get_floating_ip` search. An absent row or a row without `port_id` returns outcome `not_attached` with no mutation. Otherwise one `PUT floatingips/{id}` sends `{"floatingip":{"port_id":null}}` (outcome `detached`, `Detached` true). Any failure of that PUT, including 404, is a `*DetachFloatingIPError` and never falls back to Nova.
- Nova backend: a missing IP fails before any mutation; otherwise one `POST servers/{id}/action` sends `removeFloatingIp` with the row's address (outcome `nova_action_accepted`; Python returns None here, so `Detached` stays false).
- The server ID is required and validated, but the Neutron branch never checks that the IP belonged to that server, like the pinned helper. The mutation is sent at most once; cancellation never rolls back an accepted request.
- If the Neutron lookup itself falls back to Nova (404 on the list), Go continues with the Nova action, while Python's Neutron branch would act on the normalized Nova row. This difference keeps the operation unresolved.

## `AddAutoIP`

Matches `add_auto_ip` for basic use.

- Reuse (default) calls `AvailableFloatingIP` for the server; `WithAddAutoIPReuse(false)` allocates with `CreateFloatingIP` passing the server, the wait flag and the whole timeout, then spends that budget.
- If the supplied server snapshot, or a refreshed server when the IP has a `port_id`, already shows the exact floating IPv4, nothing is attached.
- Otherwise the exact address is attached through `AddIPList`. A newly allocated Neutron IP skips this step; a Nova allocation is always attached.
- With `WithAddAutoIPWait(true)` the server is polled (interval `min(5s, remaining)` by default, first poll immediate) until the address appears or the budget (default 60 seconds) ends with `*AddAutoIPTimeoutError`, which matches `context.DeadlineExceeded`. A non-positive remaining budget expires before the first poll. Plain HTTP calls are bounded only by `ctx`.
- On that timeout, a Neutron IP created by this call is deleted with `DeleteFloatingIP` on a context that ignores parent cancellation, unless `WithAddAutoIPCleanup(false)`. The delete outcome is in `Cleanup`/`CleanupError` and joins the returned error. User cancellation triggers no cleanup. Mutations are never replayed.
- `InterfaceIP` is the interface IP computed from the returned server model (refreshed only when a refresh or wait occurred); empty corresponds to Python's None.

Differences: the attach goes through the explicit-address workflow (address lookup by list), so Go sends one extra read compared with Python's direct attach; dynamic Server/Resource inputs and configuration aliases are not accepted.

Behavior is covered by `connection_r2_cloud_net_ip_test.go`.
