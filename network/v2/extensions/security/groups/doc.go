// Package groups exposes Gophercloud v2.15.0's
// openstack/networking/v2/extensions/security/groups calls through API, with
// concrete options, the SDK's request call categories and typed results.
// Connection.NetworkV2 returns the network/v2 service, whose SecurityGroups
// field provides this API.
//
// `SecurityGroups.Resources.List/All` classifies
// `resource.WithFilter`/`WithFilters` into 17 queries, 21 accepted names and
// three local Body fields. It maps is_shared/tag aliases;
// revision_number/tenant_id/project_id/stateful/name/id remain server
// conditions. created_at/updated_at/security_group_rules use original JSON,
// retaining unknown rule fields, large numbers and null elements. Native
// Group/nested Rule/timestamp decoding applies to the whole page before
// filtering, returning typed SecGroup. An SDK-owned raw pager preserves
// repeated/nil/extension queries unsupported by native ListOpts. Existing
// WithName local comparison and native typed List/FindIdentity/Get/Delete/rule
// mutations remain separate. WithStatus is unsupported because native Status is
// absent; semantic status is discarded, raw status remains a query.
// `Network(ctx).API.SecurityGroups.Resources` and
// `NetworkV2(ctx).SecurityGroups.Resources` share the cached client.
package groups
