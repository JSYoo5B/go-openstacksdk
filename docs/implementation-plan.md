# Implementation priorities and coverage

Implementation proceeds through core user APIs, core administrative APIs, managed-service user APIs, and managed-service administrative APIs. Swift remains a core service. Within following services, prioritize network services, then bare metal, then the remaining services. Support decisions and tests belong to the [ledger](sdk-support-ledger.md).

## Current coverage

<!-- sdk-progress:start -->
| Metric | Current value | Interpretation |
|---|---:|---|
| Pinned source declarations | 3,362 | Gophercloud/openstacksdk declarations; inherited, descriptor and Resource surfaces tracked separately |
| Implementation target | 3,228 | All declarations excluding unsupported routes, such as retired server APIs |
| API mappings | 2,498 (77.4%) | go_mapping decisions against the target; partial verification does not increase this count |
| Source-equivalent support | 0 | supported decisions |
| Unresolved / excluded (unsupported) | 730 / 134 | Unreviewed declarations are included in unresolved totals |
| Operation reviews | 2,851 | Declarations without individual reviews 511 |
| Behavior contracts | 8,709 | contracts entries in the [review JSON](../api/sdk_reviews.json); differing granularity prevents using this as an API completion rate |
<!-- sdk-progress:end -->

Behavior-contract counts reflect review granularity and must not be used as completed API counts. Unreviewed operations remain unresolved.

## Service coverage

<!-- sdk-service-progress:start -->
| Priority group | API mappings / target | Mapping rate | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|---:|
| Core services: stages 1 and 2 | 2,055 / 2,158 | 95.2% | 103 | 0 | 134 |
| Following network: stages 3 and 4 | 92 / 254 | 36.2% | 13 | 149 | 0 |
| Following bare metal: stages 3 and 4 | 63 / 207 | 30.4% | 1 | 143 | 0 |
| Other following services: stages 3 and 4 | 278 / 580 | 47.9% | 83 | 219 | 0 |
| Shared foundation | 10 / 29 | 34.5% | 19 | 0 | 0 |
| Total | 2,498 / 3,228 | 77.4% | 219 | 511 | 134 |

**Core services**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Identity / Keystone | 373 / 379 | 6 | 0 | 10 |
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
| Load Balancer / Octavia | 60 / 144 | 7 | 77 | 0 |
| DNS / Designate | 32 / 110 | 6 | 72 | 0 |
| Bare Metal / Ironic | 57 / 189 | 0 | 132 | 0 |
| Bare Metal Introspection | 6 / 18 | 1 | 11 | 0 |
| Database / Trove | 34 / 55 | 2 | 19 | 0 |
| Container Infra / Magnum | 24 / 56 | 0 | 32 | 0 |
| Container / Zun | 4 / 4 | 0 | 0 | 0 |
| Clustering / Senlin | 52 / 69 | 17 | 0 | 0 |
| Orchestration / Heat | 29 / 66 | 10 | 27 | 0 |
| Messaging / Zaqar | 20 / 38 | 4 | 14 | 0 |
| Workflow / Mistral | 12 / 30 | 0 | 18 | 0 |
| Shared File System / Manila | 81 / 196 | 6 | 109 | 0 |
| Instance HA / Masakari | 8 / 17 | 9 | 0 | 0 |
| Metric | 9 / 9 | 0 | 0 | 0 |
| Reservation / Blazar | 5 / 5 | 0 | 0 | 0 |
| Accelerator / Cyborg | 0 / 35 | 35 | 0 | 0 |
<!-- sdk-service-progress:end -->

## Development criteria

Work in bounded functional groups. Establish inputs, defaults, permissions, source pins, result ownership, errors, paging, and waits before changing a support decision. Implement concrete SDK-owned APIs where the pinned transport library lacks a necessary route. Reuse shared engines only when each binding's behavior has been verified.

Keep implementation, source comparison, focused tests, full checks, documentation, and authenticated acceptance distinct. Shared-engine changes require affected regressions; prose-only changes do not require repeating unrelated integration runs. See [testing](testing.md), [feasibility](core-unresolved-feasibility.md), and [release criteria](release-milestones.md).
