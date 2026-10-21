# Security group mutation helpers

`SecurityGroupCloud` (see `cloud-security-group-reads-python.md`) also carries
the cloud-level security group mutations. Each helper first performs the
cloud lookup: it lists every group without a server query and selects one row
locally by exact name or ID (or glob), so duplicate matches fail with
`resource.ErrAmbiguous` before any mutation. The backend is the one chosen by
the read dependencies: Neutron by default, the legacy Nova
`/os-security-groups` API when only Nova is configured, and
`ErrUnavailableSecurityGroups` (no request) when the feature is disabled.

| Method | Python operation | Behavior |
| --- | --- | --- |
| `DeleteSecurityGroup(ctx, nameOrID)` | `delete_security_group` | Returns `false, nil` without a mutation when nothing matches; otherwise one DELETE that must succeed (a 404 after the lookup is an error) and returns `true`. |
| `UpdateSecurityGroup(ctx, nameOrID, SecurityGroupUpdate)` | `update_security_group` | Missing group is `resource.ErrNotFound` before any mutation. Neutron sends only the supplied `Name`, `Description` and `Stateful` (empty string and `false` are sent; no fields sends no request). Nova fills a missing name and description from the found group, and passes `stateful` through. |
| `CreateSecurityGroupRule(ctx, nameOrID, SecurityGroupRuleSpec)` | `create_security_group_rule` | Neutron always sends `security_group_id`, `port_range_min/max` (`-1` becomes null), `protocol`, `remote_ip_prefix`, `remote_group_id`, `remote_address_group_id` (falsy values are dropped by the native rule mutation), `direction` (default ingress) and `ethertype` (default IPv4), with `tenant_id` and `description` only when supplied. Nova requires a protocol, rejects egress, maps ICMP missing ports to `-1` and TCP/UDP with both ports missing to 1-65535, and ignores `ethertype`, `remote_address_group_id` and `description`. |

Mutations are never replayed automatically. Nova responses are returned as the
raw `security_group` / `security_group_rule` JSON in `NovaBody`; the Python
munch normalization of those payloads is not applied. Neutron results use the
native `groups.SecGroup` and `rules.SecGroupRule` models.
