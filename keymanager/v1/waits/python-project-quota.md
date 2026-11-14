# Barbican admin ProjectQuota waits

`ProjectQuota(projectID)` and `ForProjectQuota(record)` add the missing concrete admin target to `WaitForStatus` and `WaitForDelete`. They reuse the existing guarded reader and prepared wait policy, polling only `GET /project-quotas/{originalProjectID}`. Response id/name/href are passive. Literal targets seed Resource.id with the requested project, all other declared fields with null, and no fake receipt. Cached records snapshot actual Resource/Wire/Envelope/header/status independently.

ProjectQuota declares **no status field**. A default status wait fails unsupported before HTTP; select a declared string/null field using `WithWait(resource.WithStatusAttribute(...))`. Numeric quota values are not converted into status text. Cached matching string/null states return without polling. Status timeout is unlimited, deletion timeout120s, interval2s; explicit options preserve the existing Source-shaped timer/failure/callback policy.

WaitForDelete sends no DELETE. It ignores undeclared wire status=deleted and completes only when GET returns a clean404. A physical read/Close/context/source failure carrying404 is not absence. The result owns the original Resource snapshot plus actual absence receipt. Resetting a project override normally leaves effective/configuration GET available; observing deletion is not proof that quota settings were reset.

## Standalone Python

```python
import os
import openstack


def main():
    conn = openstack.connect()
    quota = conn.key_manager.get_project_quota("project-id")
    current = conn.key_manager.wait_for_status(
        quota, "project-id", attribute="id", wait=120
    )
    print(current.id)
    if os.environ.get("WAIT_ABSENCE") == "1":
        conn.key_manager.wait_for_delete(quota, wait=120)


if __name__ == "__main__":
    main()
```

## Standalone Go

Set `OS_KEY_MANAGER_ENDPOINT` and `OS_TOKEN`. The optional `WAIT_ABSENCE=1` demonstrates absence observation rather than issuing a configuration DELETE.

See `Example_projectQuota` in [example_project_quota_test.go](example_project_quota_test.go), compiled by `go test`.

These finite constructors now cover the known ProjectQuota gap alongside Container, Order, Secret, SecretStore, SecretACL and effective Quota. SecretConsumer has allow_fetch=False in pinned Source. This does not implement arbitrary Python Resource/runtime subclasses, mutable identity, session/location/cache, untyped callbacks or every dynamic getattr behavior. Immutable routes, typed callback errors, selected nullable strings, finite UTF-8 JSON and owned-null location remain explicit Go differences under SDK-R1/C1/S1. Local HTTP tests do not establish live cloud or release parity.
