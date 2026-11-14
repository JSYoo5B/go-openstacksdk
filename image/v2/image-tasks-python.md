# Image Tasks Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--image', required=True)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
for task in conn.image.image_tasks(args.image):
    print(json.dumps(task.to_dict(), ensure_ascii=False))
```

See `Example_imageTasks` in [example_image_tasks_test.go](example_image_tasks_test.go), compiled by `go test`.

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1181-L1194)
- [`57f7dd9`](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L599-L614)
- [Glance Policy](https://docs.openstack.org/glance/2025.2/configuration/glance_policy.html)
- [`ImageTasks`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image_tasks.py#L22-L64)
- [`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2359)
- [`image.Service.ImageTasks`/`AllImageTasks`](../image-tasks.md)
