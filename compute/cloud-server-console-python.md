# Cloud Server Console Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--server", default="vm")
    parser.add_argument("--server-json", default="")
    parser.add_argument("--length-json", default="")
    args = parser.parse_args()
    server = json.loads(args.server_json) if args.server_json else args.server
    length = json.loads(args.length_json) if args.length_json else None
    with openstack.connect(cloud=args.cloud) as conn:
        output = conn.get_server_console(server, length=length)
        print(json.dumps(output, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_string` in [example_cloud_server_console1_test.go](example_cloud_server_console1_test.go), compiled by `go test`.

```sh
python main.py --cloud devstack --server vm --length-json 50
go run main.go --cloud devstack --server vm --length-json 50
python main.py --cloud devstack --server-json '{"id":"s1"}' --length-json 0
go run main.go --cloud devstack --server-json '{"id":"s1"}' --length-json 0
```

See `ExampleService_GetServerConsole` in [example_servers_test.go](example_servers_test.go), compiled by `go test`.
