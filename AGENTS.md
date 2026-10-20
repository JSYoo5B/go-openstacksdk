# Project instructions

## Product and design

- Provide a high-level Go OpenStack SDK with Connection, service, resource, and workflow layers over pinned Gophercloud v2.15.0. The module minimum is Go 1.25.
- Compare Python behavior against the pinned openstacksdk source in api/openstacksdk/manifest.json. Native transport bindings and full SDK behavior are separate support claims.
- Use concrete request types and operation-specific options. The library owns builders, serialization, snapshots, defaults, and protected-field validation.
- Preserve omission, null, false, zero, and empty values when the operation distinguishes them. Keep declared Resource views separate from actual Wire and HTTP evidence.
- Make context cancellation, reference identities, duplicate/missing policies, paging, microversions, permissions, waits, partial success, and ownership explicit per operation.
- Never automatically replay an uncertain mutation or imply that local cancellation rolls back remote work. Do not infer asynchronous completion from acknowledgement.
- Update api/sdk_reviews.json, behavior tests, and usage documentation together. Preserve source pins/fingerprints; leave partial contracts unresolved when required behavior remains.
- Prioritize core user APIs, core administration, following-service user APIs, then following-service administration. Within following services, prioritize networking, bare metal, then remaining services.
- Preserve source provenance, licenses, and attribution for dependencies, ports, excerpts, and data.

## Privacy review


Before publishing source, documentation, fixtures, commit messages, release notes,
pull requests, issues or GitHub discussions, search for private
execution-environment details. Review matches in context, including hidden CI
files and generated reports. These rules apply to internal code as well as
pull request and issue descriptions, reviews and comments.

- Omit actual personal hostnames, private domains and addresses, account-specific
  filesystem paths, remote-access helpers, checkout locations and private log names.
- When describing paths, use only file paths within this repository or related
  repositories. Do not expose unrelated filesystem locations or unnecessary
  arbitrary paths.
- Software versions, architecture and hardware specifications may remain when they
  explain reproducibility, compatibility or performance. Remove their association
  with an identifiable private host or access path.
- Preserve functional platform strings, synthetic test inputs, public CI runner
  information, general product requirements and public author attribution or image
  namespaces. Do not globally replace operating-system or architecture strings.
- When logs or other diagnostic information must be shown as examples, mask
  information unrelated to the point being demonstrated. Retain only the details
  needed to explain or reproduce that point.

## Documentation and validation

- Write documentation, code comments, commit messages, release notes, and issue/PR descriptions in English. Preserve intentional non-ASCII test inputs.
- Describe current features, usage, exact behavior, and explicit limitations. Keep planned support separate from implemented and independently verified behavior.
- Keep README.md focused on installation, usage, and feature boundaries. Put detailed contracts beside APIs and shared rules under docs.
- Maintain English generator templates as well as generated documentation. Use Go 1.27.2 for reference formatting while retaining the Go 1.25 module minimum.
- Run make check and relevant focused checks for implementation changes. Supply OPENSTACKSDK_SOURCE and GOPHERCLOUD_METADATA for source-backed generator checks; use the pinned inputs and Python 3.14 for AST fingerprints.
- Run applicable checks locally unless the user selects another execution environment. Unit/HTTP fixtures, source comparisons, external consumer builds, authenticated deployment acceptance, and CI are distinct evidence.
