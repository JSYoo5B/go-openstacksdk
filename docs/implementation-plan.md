# Implementation priorities and coverage

Implementation proceeds through core user APIs, core administrative APIs, managed-service user APIs, and managed-service administrative APIs. Swift remains a core service. Within following services, prioritize network services, then bare metal, then the remaining services. Support decisions and tests belong to the [ledger](sdk-support-ledger.md).

## Current coverage

<!-- sdk-progress:start -->
| Metric | Current value | Interpretation |
|---|---:|---|
| Pinned source declarations | 3,362 | Gophercloud/openstacksdk declarations; inherited, descriptor and Resource surfaces tracked separately |
| Implementation target | 3,228 | All declarations excluding unsupported routes, such as retired server APIs |
| API mappings | 2,135 (66.1%) | go_mapping decisions against the target; partial verification does not increase this count |
| Source-equivalent support | 0 | supported decisions |
| Unresolved / excluded (unsupported) | 1,093 / 134 | Unreviewed declarations are included in unresolved totals |
| Operation reviews | 2,490 | Declarations without individual reviews 872 |
| Behavior contracts | 7,780 | contracts entries in the [review JSON](../api/sdk_reviews.json); differing granularity prevents using this as an API completion rate |
<!-- sdk-progress:end -->

Behavior-contract counts reflect review granularity and must not be used as completed API counts. Unreviewed operations remain unresolved.

## Service coverage

<!-- sdk-service-progress:start -->
| Priority group | API mappings / target | Mapping rate | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|---:|
| Core services: stages 1 and 2 | 2,054 / 2,158 | 95.2% | 103 | 1 | 134 |
| Following network: stages 3 and 4 | 5 / 254 | 2.0% | 13 | 236 | 0 |
| Following bare metal: stages 3 and 4 | 4 / 207 | 1.9% | 1 | 202 | 0 |
| Other following services: stages 3 and 4 | 68 / 580 | 11.7% | 85 | 427 | 0 |
| Shared foundation | 4 / 29 | 13.8% | 19 | 6 | 0 |
| Total | 2,135 / 3,228 | 66.1% | 221 | 872 | 134 |

**Core services**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Identity / Keystone | 372 / 379 | 6 | 1 | 10 |
| Compute / Nova | 300 / 333 | 33 | 0 | 0 |
| Placement | 68 / 71 | 3 | 0 | 0 |
| Network / Neutron | 689 / 728 | 39 | 0 | 30 |
| Image / Glance | 108 / 109 | 1 | 0 | 11 |
| Block Storage / Cinder | 389 / 398 | 9 | 0 | 82 |
| Key Manager / Barbican | 66 / 67 | 1 | 0 | 0 |
| Object Storage / Swift | 62 / 73 | 11 | 0 | 1 |

**Following services: network, bare metal, others**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Load Balancer / Octavia | 3 / 144 | 7 | 134 | 0 |
| DNS / Designate | 2 / 110 | 6 | 102 | 0 |
| Bare Metal / Ironic | 3 / 189 | 0 | 186 | 0 |
| Bare Metal Introspection | 1 / 18 | 1 | 16 | 0 |
| Database / Trove | 1 / 55 | 2 | 52 | 0 |
| Container Infra / Magnum | 1 / 56 | 0 | 55 | 0 |
| Container / Zun | 0 / 4 | 0 | 4 | 0 |
| Clustering / Senlin | 52 / 69 | 17 | 0 | 0 |
| Orchestration / Heat | 6 / 66 | 23 | 37 | 0 |
| Messaging / Zaqar | 0 / 38 | 4 | 34 | 0 |
| Workflow / Mistral | 0 / 30 | 0 | 30 | 0 |
| Shared File System / Manila | 0 / 196 | 6 | 190 | 0 |
| Instance HA / Masakari | 8 / 17 | 9 | 0 | 0 |
| Metric | 0 / 9 | 0 | 9 | 0 |
| Reservation / Blazar | 0 / 5 | 0 | 5 | 0 |
| Accelerator / Cyborg | 0 / 35 | 24 | 11 | 0 |
<!-- sdk-service-progress:end -->

## Development criteria

Work in bounded functional groups. Establish inputs, defaults, permissions, source pins, result ownership, errors, paging, and waits before changing a support decision. Implement concrete SDK-owned APIs where the pinned transport library lacks a necessary route. Reuse shared engines only when each binding's behavior has been verified.

Keep implementation, source comparison, focused tests, full checks, documentation, and authenticated acceptance distinct. Shared-engine changes require affected regressions; prose-only changes do not require repeating unrelated integration runs. See [testing](testing.md), [feasibility](core-unresolved-feasibility.md), and [release criteria](release-milestones.md).
