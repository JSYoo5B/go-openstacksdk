# Implementation priorities and coverage

Implementation proceeds through core user APIs, core administrative APIs, managed-service user APIs, and managed-service administrative APIs. Swift remains a core service. Within following services, prioritize network services, then bare metal, then the remaining services. Support decisions and tests belong to the [ledger](sdk-support-ledger.md).

## Current coverage

<!-- sdk-progress:start -->
| Metric | Current value | Interpretation |
|---|---:|---|
| Pinned source declarations | 3,362 | Gophercloud/openstacksdk declarations; inherited, descriptor and Resource surfaces tracked separately |
| Implementation target | 3,216 | All declarations excluding unsupported routes, such as retired server APIs |
| API mappings | 3,192 (99.3%) | go_mapping decisions against the target; partial verification does not increase this count |
| Source-equivalent support | 0 | supported decisions |
| Unresolved / excluded (unsupported) | 24 / 146 | Unreviewed declarations are included in unresolved totals |
| Operation reviews | 3,362 | Declarations without individual reviews 0 |
| Behavior contracts | 9,926 | contracts entries in the [review JSON](../api/sdk_reviews.json); differing granularity prevents using this as an API completion rate |
<!-- sdk-progress:end -->

Behavior-contract counts reflect review granularity and must not be used as completed API counts. Unreviewed operations remain unresolved.

## Service coverage

<!-- sdk-service-progress:start -->
| Priority group | API mappings / target | Mapping rate | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|---:|
| Core services: stages 1 and 2 | 2,131 / 2,154 | 98.9% | 23 | 0 | 138 |
| Following network: stages 3 and 4 | 254 / 254 | 100.0% | 0 | 0 | 0 |
| Following bare metal: stages 3 and 4 | 207 / 207 | 100.0% | 0 | 0 | 0 |
| Other following services: stages 3 and 4 | 578 / 578 | 100.0% | 0 | 0 | 2 |
| Shared foundation | 22 / 23 | 95.7% | 1 | 0 | 6 |
| Total | 3,192 / 3,216 | 99.3% | 24 | 0 | 146 |

**Core services**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Identity / Keystone | 379 / 379 | 0 | 0 | 10 |
| Compute / Nova | 326 / 333 | 7 | 0 | 0 |
| Placement | 71 / 71 | 0 | 0 | 0 |
| Network / Neutron | 718 / 728 | 10 | 0 | 30 |
| Image / Glance | 109 / 109 | 0 | 0 | 11 |
| Block Storage / Cinder | 398 / 398 | 0 | 0 | 82 |
| Key Manager / Barbican | 67 / 67 | 0 | 0 | 0 |
| Object Storage / Swift | 63 / 69 | 6 | 0 | 5 |

**Following services: network, bare metal, others**

| Service | API mappings / target | Partial/unresolved reviews | Unreviewed | Excluded (unsupported) |
|---|---:|---:|---:|---:|
| Load Balancer / Octavia | 144 / 144 | 0 | 0 | 0 |
| DNS / Designate | 110 / 110 | 0 | 0 | 0 |
| Bare Metal / Ironic | 189 / 189 | 0 | 0 | 0 |
| Bare Metal Introspection | 18 / 18 | 0 | 0 | 0 |
| Database / Trove | 55 / 55 | 0 | 0 | 0 |
| Container Infra / Magnum | 56 / 56 | 0 | 0 | 0 |
| Container / Zun | 4 / 4 | 0 | 0 | 0 |
| Clustering / Senlin | 69 / 69 | 0 | 0 | 0 |
| Orchestration / Heat | 66 / 66 | 0 | 0 | 0 |
| Messaging / Zaqar | 37 / 37 | 0 | 0 | 1 |
| Workflow / Mistral | 30 / 30 | 0 | 0 | 0 |
| Shared File System / Manila | 196 / 196 | 0 | 0 | 0 |
| Instance HA / Masakari | 17 / 17 | 0 | 0 | 0 |
| Metric | 9 / 9 | 0 | 0 | 0 |
| Reservation / Blazar | 5 / 5 | 0 | 0 | 0 |
| Accelerator / Cyborg | 34 / 34 | 0 | 0 | 1 |
<!-- sdk-service-progress:end -->

## Development criteria

Work in bounded functional groups. Establish inputs, defaults, permissions, source pins, result ownership, errors, paging, and waits before changing a support decision. Implement concrete SDK-owned APIs where the pinned transport library lacks a necessary route. Reuse shared engines only when each binding's behavior has been verified.

Keep implementation, source comparison, focused tests, full checks, documentation, and authenticated acceptance distinct. Shared-engine changes require affected regressions; prose-only changes do not require repeating unrelated integration runs. See [testing](testing.md), [feasibility](core-unresolved-feasibility.md), and [release criteria](release-milestones.md).
