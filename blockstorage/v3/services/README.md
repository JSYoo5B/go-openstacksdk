# Cinder services

Native `List` preserves Gophercloud's typed service transport. Owned actions
provide service enable/disable/freeze/thaw/failover and log operations.

[Source-shaped lookup and worker cleanup](python-find-cleanup.md) documents
`FindRecord`, `CleanupServiceWorkers`, Python comparisons and remaining kwargs
boundaries. These helpers own list matching, duplicate/missing behavior,
version selection, defaults, and response evidence.
