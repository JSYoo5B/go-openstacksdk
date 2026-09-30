#!/usr/bin/env python3
"""Record declared public methods in the pinned Python proxies and cloud modules.

Candidates are discovery hints, never proof of equivalent behavior. Generating
an inventory always leaves reviews pending and never marks a method implemented.
"""
import argparse
import ast
import json
from pathlib import Path
import subprocess

PIN = "ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe"
SERVICES = {
    "block_storage": "blockstorage", "baremetal_introspection": "baremetalintrospection",
    "container_infrastructure_management": "containerinfra", "database": "db",
    "key_manager": "keymanager", "load_balancer": "loadbalancer", "message": "messaging",
    "object_store": "objectstorage", "shared_file_system": "sharedfilesystems",
}


def imported_names(tree):
    names = {}
    for node in tree.body:
        if isinstance(node, ast.ImportFrom) and node.module:
            for value in node.names:
                names[value.asname or value.name] = node.module + "." + value.name
        elif isinstance(node, ast.Import):
            for value in node.names:
                names[value.asname or value.name] = value.name
    return names


def targets(method, imports):
    result = set()
    for node in ast.walk(method):
        if isinstance(node, ast.Attribute) and isinstance(node.value, ast.Name):
            module = imports.get(node.value.id)
            if module and module.startswith("openstack.") and node.attr[0].isupper():
                result.add(module + "." + node.attr)
    return sorted(result)


def parameters(method):
    args = method.args
    positional = args.posonlyargs + args.args
    start = len(positional) - len(args.defaults)
    defaults = {arg.arg: ast.unparse(value) for arg, value in zip(positional[start:], args.defaults)}
    defaults.update({arg.arg: ast.unparse(value) for arg, value in zip(args.kwonlyargs, args.kw_defaults) if value is not None})
    result = []
    for arg in positional + args.kwonlyargs:
        if arg.arg in {"self", "cls"}:
            continue
        value = {"name": arg.arg, "required": arg.arg not in defaults}
        if arg.arg in defaults:
            value["default"] = defaults[arg.arg]
        result.append(value)
    if args.vararg:
        result.append({"name": args.vararg.arg, "variadic": True})
    if args.kwarg:
        result.append({"name": args.kwarg.arg, "keyword_attributes": True})
    return result


def methods_in(path, source, classes=None):
    tree = ast.parse(path.read_text())
    imports = imported_names(tree)
    result = {}
    for cls in tree.body:
        if not isinstance(cls, ast.ClassDef) or (classes and cls.name not in classes):
            continue
        for method in cls.body:
            if not isinstance(method, (ast.FunctionDef, ast.AsyncFunctionDef)) or method.name.startswith("_"):
                continue
            key = cls.name + "." + method.name
            result[key] = {
                "name": method.name,
                "class": cls.name,
                "source": str(path.relative_to(source)),
                "line": method.lineno,
                "parameters": parameters(method),
                "resources": targets(method, imports),
            }
    return list(result.values())


def write(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", default=Path("api/openstacksdk"), type=Path)
    parser.add_argument("--bindings", default=Path("api/resource_inventory.json"), type=Path)
    args = parser.parse_args()
    source = args.source.resolve()
    revision = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip()
    if revision != PIN:
        parser.error("source must be pinned to " + PIN)
    bindings = json.loads(args.bindings.read_text())
    groups = []
    total = 0
    for path in sorted((source / "openstack").glob("*/v*/_proxy.py")):
        service, version = path.parts[-3:-1]
        operations = methods_in(path, source, {"Proxy"})
        for operation in operations:
            operation["id"] = f"{service}/{version}/{operation['name']}"
            operation["kind"] = "proxy"
            models = {resource.split(".")[-1] for resource in operation["resources"]}
            family = SERVICES.get(service, service)
            operation["candidates"] = [binding["package"] for binding in bindings if binding.get("model") in models and binding["package"].startswith(f"gophercloudsdk/{family}/{version}/")]
            operation["review"] = "pending"
        name = service + "/" + version + ".json"
        write(args.output / name, {"revision": revision, "operations": operations})
        groups.append({"service": service, "version": version, "file": name, "methods": len(operations)})
        total += len(operations)
    workflows = []
    for path in sorted((source / "openstack/cloud").glob("*.py")):
        for operation in methods_in(path, source):
            operation["id"] = f"cloud/{path.stem}/{operation['class']}/{operation['name']}"
            operation["kind"] = "workflow"
            operation["review"] = "pending"
            workflows.append(operation)
    write(args.output / "cloud.json", {"revision": revision, "operations": workflows})
    connection = methods_in(source / "openstack/connection.py", source, {"Connection"})
    for operation in connection:
        operation["id"] = "connection/" + operation["name"]
        operation["kind"] = "connection"
        operation["review"] = "pending"
    write(args.output / "connection.json", {"revision": revision, "operations": connection})
    manifest = {"revision": revision, "source_url": "https://github.com/openstack/openstacksdk/tree/" + revision, "proxy_methods": total, "cloud_workflows": len(workflows), "connection_methods": len(connection), "proxies": groups}
    write(args.output / "manifest.json", manifest)
    print(f"Recorded {total} proxy methods, {len(workflows)} workflows, {len(connection)} connection methods. Reviews remain pending.")


if __name__ == "__main__":
    main()
