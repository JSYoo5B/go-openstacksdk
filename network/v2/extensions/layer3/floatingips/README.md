# Floatingips

Floating-IP operations distinguish native typed calls, owned record APIs, and higher-level Connection workflows. Explicit parent IDs and actual wire IDs remain separate. Available-IP selection must check association state; allocation, attachment, deletion, and waiting have distinct success/partial-failure contracts. Read the operation-specific record and available-IP guides before relying on automatic selection.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../../../docs/sdk-support-ledger.md) and [operation reviews](../../../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [python_list_records.go](python_list_records.go)
- [python_mutations.go](python_mutations.go)
- [resources_generated.go](resources_generated.go)
- [get_fields_generated.go](get_fields_generated.go)
- [delete_options_generated.go](delete_options_generated.go)
- [python_available.go](python_available.go)

## References

- [Python List Records](python-list-records.md)
- [Python Available](python-available.md)
- [Python Mutations](python-mutations.md)
