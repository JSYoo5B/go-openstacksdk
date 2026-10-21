# Service statuses

SDK-owned binding for `GET /v2/service_statuses` and `GET /v2/service_statuses/{id}` (200 only), reached with `service.ServiceStatuses()`. `List` follows links.next like every Designate list and accepts no filters (Python `service_statuses()` has none). `Capabilities` and `Stats` keep their exact JSON. Access normally requires an administrative role.
