# Python openstacksdk comparison inventory

This inventory reads the AST of [pinned openstacksdk source](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe). It records 1,834 directly declared service Proxy methods, 397 cloud-module methods, and five directly declared Connection methods. Inherited or dynamically added behavior is outside these declaration counts.

Entries preserve arguments/defaults, referenced resources, and source locations. Candidate Go packages are name/service/model matches, not equivalence decisions. Generated review markers remain pending; durable decisions belong to [sdk_reviews.json](../sdk_reviews.json) and are validated by [paritycheck](../../internal/cmd/paritycheck/README.md).

The manifest records service inventories; [cloud.json](cloud.json) and [connection.json](connection.json) record shared helpers. [resource_inventory.json](../resource_inventory.json) describes Go collection bindings. Candidate matching may use upstream_model for SDK-owned models; model names record actual collection results.

Regenerate with `internal/cmd/parity/inventory.py`, selecting the pinned related-source checkout through its `--source` argument. The tool rejects a different source commit. Python source is comparison data, not a product runtime dependency.
