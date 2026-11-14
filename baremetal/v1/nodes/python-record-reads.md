# Python Record Reads

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import sys
import openstack

def main():
    conn = openstack.connect(cloud='dev', baremetal_api_version='1.109')
    action = sys.argv[1] if len(sys.argv) > 1 else 'list'
    identity = sys.argv[2] if len(sys.argv) > 2 else 'node-a'
    if action == 'list':
        values = conn.baremetal.nodes(details=True, fields=['id', 'name'])
        print(json.dumps([value.to_dict() for value in values], indent=2))
    elif action == 'get':
        value = conn.baremetal.get_node(identity, fields=['id', 'name'])
        print(json.dumps(value.to_dict(), indent=2))
    elif action == 'find':
        value = conn.baremetal.find_node(identity, ignore_missing=True, details=True)
        print(json.dumps(value.to_dict() if value is not None else None, indent=2))
    else:
        raise ValueError('expected list, get or find')

if __name__ == '__main__':
    main()
```

See `Example_recordReads` in [example_record_reads_test.go](example_record_reads_test.go), compiled by `go test`.
