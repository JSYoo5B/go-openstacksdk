# Core API permission branches

SDK catalog coverage is separate from server authorization. The reference policies are OpenStack 2025.2: Cinder 27.0.0, Nova 32.0.0, and Neutron 27.0.0. Actual deployments decide policy overrides and ownership.

| API | Project scope | Additional conditions |
|---|---|---|
| Cinder attachment CreateRecord/ListRecords | Project member creation and project-scoped lists | Source documents service-internal use; this is not an ordinary user workflow guarantee. |
| Cinder connector initialize/terminate | Project member | Low-level connector operations are not classified as stock administrator-only. |
| Cinder type FindRecord | Project reader | Normal type-read policy. |
| Cinder encryption create/update and QoS FindRecord | Administrative | Stock administrator APIs. |
| Nova attachment UpdateRecord | Same-volume delete_on_termination changes at 2.85+ | Earlier versions or volume swaps require service/admin policy. |
| Nova UnshelveWithOptions | No explicit host | Nonempty host adds administrator policy. |
| Nova server-group FindRecord | Current project | all_projects fallback requires administrative policy. |
| Nova direct fixed/floating-IP actions | Through selected version 2.43 | Removed in 2.44, separately from inventory proxies removed in 2.36. |
| Neutron port forwarding | Project member/reader | Parent floating-IP ownership and server policy apply. |

Primary references: [Cinder attachments](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/attachments.py), [volume actions](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/volume_actions.py), [Nova attachment controller](https://github.com/openstack/nova/blob/32.0.0/nova/api/openstack/compute/volume_attachments.py), [Nova attachment policies](https://github.com/openstack/nova/blob/32.0.0/nova/policies/attach_volume.py), and [Neutron policy](https://docs.openstack.org/neutron/2025.2/configuration/policy.html). Static policy comparison is separate from authenticated acceptance.
