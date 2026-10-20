# Unresolved core API feasibility

The [operation reviews](../api/sdk_reviews.json) are authoritative for current per-operation status. Feasibility categories describe implementation strategies and do not promote an operation to supported.

| Category | Implementation approach |
|---|---|
| Review evidence | Verify matching requests, defaults, responses, and error policies before assigning go_mapping. |
| Missing transport binding | Implement SDK-owned concrete calls when a supported server route is absent from pinned Gophercloud. |
| Common helpers | Bind exact finding/fallback, extension checks, local filters, paging, selective fields, and waits per resource. |
| Input representation | Distinguish omission/null/false/zero/empty values and repeated query keys with owned options/builders. |
| Response models | Preserve extension fields, nullable values, exact numbers, raw bodies, and actual response evidence. |
| Retired server APIs | Record unsupported scope against the actual server release; do not invent replacement routes. |
| Python-specific behavior or source defects | State an explicit Go mapping or unresolved boundary and test the observable contract. |

Potential shared work includes SDK-R1/C1/S1 Resource/session/cache lifecycles, descriptor conversion, full encoding profiles, and cross-service workflow policies. A helper's existence does not prove that all service bindings use it correctly. Avoid estimating current completion from older category totals; use [generated coverage](implementation-plan.md).
