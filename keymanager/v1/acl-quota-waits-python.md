# Acl Quota Waits Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack
from openstack.key_manager.v1.secret_acl import SecretACL
from openstack.key_manager.v1.quota import Quota

class PollableSecretACL(SecretACL):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

class PollableQuota(Quota):
    def fetch(self, session, **kwargs):
        return super().fetch(session, requires_id=False, **kwargs)

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=["acl", "quota"], default="acl")
parser.add_argument("--secret-id")
parser.add_argument("--operation", choices=["status", "delete"], default="status")
parser.add_argument("--attribute", default="acl_ref")
parser.add_argument("--state")
parser.add_argument("--null-state", action="store_true")
parser.add_argument("--interval", type=float, default=2)
parser.add_argument("--timeout", type=float)
parser.add_argument("--correct-singleton-fetch", action="store_true")
args = parser.parse_args()
if args.kind == "acl" and not args.secret_id:
    parser.error("--secret-id is required for ACL")
if args.operation == "status" and args.state is None and not args.null_state:
    parser.error("--state or --null-state is required")
conn = openstack.connect(cloud=args.cloud)
if args.kind == "acl":
    cls = PollableSecretACL if args.correct_singleton_fetch else SecretACL
    res = cls.existing(secret_id=args.secret_id)
else:
    cls = PollableQuota if args.correct_singleton_fetch else Quota
    res = cls.existing()
options = {"interval": args.interval, "callback": lambda progress: print("progress", progress)}
if args.timeout is not None:
    options["wait"] = args.timeout
if args.operation == "delete":
    result = conn.key_manager.wait_for_delete(res, **options)
else:
    state = None if args.null_state else args.state
    result = conn.key_manager.wait_for_status(res, state, attribute=args.attribute, **options)
print(result.to_dict())
```

See `Example_aclQuotaWaits` in [example_acl_quota_waits_test.go](example_acl_quota_waits_test.go), compiled by `go test`.

## References

- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1275-L1336)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Install](../../docs/install.md)
- [Waits Python](waits-python.md)
- [Barbican ACL controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/acls.py#L153-L207)
- [effective quota controller](https://github.com/openstack/barbican/blob/21.0.0/barbican/api/controllers/quotas.py#L47-L59)
- [Policy](https://docs.openstack.org/barbican/2025.2/configuration/policy.html)
- [Acls](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/acls.py#L20-L88)
- [Quotas](https://github.com/openstack/barbican/blob/21.0.0/barbican/common/policies/quotas.py#L18-L74)
- [Python Project Quota](waits/python-project-quota.md)
- [Python Project Records](quotas/python-project-records.md)
