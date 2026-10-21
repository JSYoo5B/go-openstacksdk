# Implementation priorities and coverage

Implementation proceeds through core user APIs, core administrative APIs, managed-service user APIs, and managed-service administrative APIs. Swift remains a core service. Within following services, prioritize network services, then bare metal, then the remaining services. Support decisions and tests belong to the [ledger](sdk-support-ledger.md).

## Current coverage

<!-- sdk-progress:start -->
| Metric | Current value | Interpretation |
|---|---:|---|
| Pinned source declarations | 3,362 | Gophercloud/openstacksdk declarations; inherited, descriptor and Resource surfaces tracked separately |
| Implementation target | 3,216 | All declarations excluding unsupported routes, such as retired server APIs |
| API mappings | 2,893 (90.0%) | go_mapping decisions against the target; partial verification does not increase this count |
| Source-equivalent support | 0 | supported decisions |
| Unresolved / excluded (unsupported) | 323 / 146 | Unreviewed declarations are included in unresolved totals |
| Operation reviews | 3,362 | Declarations without individual reviews 0 |
| Behavior contracts | 9,434 | contracts entries in the [review JSON](../api/sdk_reviews.json); differing granularity prevents using this as an API completion rate |
<!-- sdk-progress:end -->

Behavior-contract counts reflect review granularity and must not be used as completed API counts. Unreviewed operations remain unresolved.

## Service coverage

<!-- sdk-service-progress:start -->
| Priority group | API mappings / target | Mapping rate | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|---:|
| Core services: stages 1 and 2 | 2,076 / 2,154 | 96.4% | 78 | 0 | 138 |
| Following network: stages 3 and 4 | 193 / 254 | 76.0% | 61 | 0 | 0 |
| Following bare metal: stages 3 and 4 | 115 / 207 | 55.6% | 92 | 0 | 0 |
| Other following services: stages 3 and 4 | 495 / 578 | 85.6% | 83 | 0 | 2 |
| Shared foundation | 14 / 23 | 60.9% | 9 | 0 | 6 |
| Total | 2,893 / 3,216 | 90.0% | 323 | 0 | 146 |

**Core services**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Identity / Keystone | 377 / 379 | 2 | 0 | 10 |
| Compute / Nova | 308 / 333 | 25 | 0 | 0 |
| Placement | 68 / 71 | 3 | 0 | 0 |
| Network / Neutron | 695 / 728 | 33 | 0 | 30 |
| Image / Glance | 108 / 109 | 1 | 0 | 11 |
| Block Storage / Cinder | 392 / 398 | 6 | 0 | 82 |
| Key Manager / Barbican | 66 / 67 | 1 | 0 | 0 |
| Object Storage / Swift | 62 / 69 | 7 | 0 | 5 |

**Following services: network, bare metal, others**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Load Balancer / Octavia | 131 / 144 | 13 | 0 | 0 |
| DNS / Designate | 62 / 110 | 48 | 0 | 0 |
| Bare Metal / Ironic | 103 / 189 | 86 | 0 | 0 |
| Bare Metal Introspection | 12 / 18 | 6 | 0 | 0 |
| Database / Trove | 50 / 55 | 5 | 0 | 0 |
| Container Infra / Magnum | 50 / 56 | 6 | 0 | 0 |
| Container / Zun | 4 / 4 | 0 | 0 | 0 |
| Clustering / Senlin | 67 / 69 | 2 | 0 | 0 |
| Orchestration / Heat | 50 / 66 | 16 | 0 | 0 |
| Messaging / Zaqar | 37 / 37 | 0 | 0 | 1 |
| Workflow / Mistral | 30 / 30 | 0 | 0 | 0 |
| Shared File System / Manila | 149 / 196 | 47 | 0 | 0 |
| Instance HA / Masakari | 10 / 17 | 7 | 0 | 0 |
| Metric | 9 / 9 | 0 | 0 | 0 |
| Reservation / Blazar | 5 / 5 | 0 | 0 | 0 |
| Accelerator / Cyborg | 34 / 34 | 0 | 0 | 1 |
<!-- sdk-service-progress:end -->

## Development criteria

Work in bounded functional groups. Establish inputs, defaults, permissions, source pins, result ownership, errors, paging, and waits before changing a support decision. Implement concrete SDK-owned APIs where the pinned transport library lacks a necessary route. Reuse shared engines only when each binding's behavior has been verified.

Keep implementation, source comparison, focused tests, full checks, documentation, and authenticated acceptance distinct. Shared-engine changes require affected regressions; prose-only changes do not require repeating unrelated integration runs. See [testing](testing.md), [feasibility](core-unresolved-feasibility.md), and [release criteria](release-milestones.md).
