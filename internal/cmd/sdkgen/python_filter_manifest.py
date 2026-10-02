#!/usr/bin/env python3
"""Extract audited Subnet, Secret or Container filters without importing OpenStack.

Only Python's standard-library AST is used. The source checkout is data, never
executed; unexpected expression shapes fail rather than becoming guessed fields.
AST hashes are CPython-minor-version dependent, as recorded in the manifest.
"""

import argparse
import ast
import hashlib
import json
import pathlib
import sys


PIN = "ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe"
FILES = (
    "openstack/network/v2/subnet.py",
    "openstack/network/v2/_base.py",
    "openstack/common/tag.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/network/v2/_proxy.py",
)
RESOURCE = "openstack.network.v2.subnet.Subnet"
ANCHORS = (
    ("openstack/network/v2/subnet.py", "Subnet"),
    ("openstack/network/v2/subnet.py", "Subnet._query_mapping"),
    ("openstack/common/tag.py", "TagMixin._tag_query_parameters"),
    ("openstack/common/tag.py", "TagMixin.tags"),
    ("openstack/network/v2/_base.py", "NetworkResource.revision_number"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/network/v2/_proxy.py", "Proxy.subnets"),
)
SECRET_RESOURCE = "openstack.key_manager.v1.secret.Secret"
SECRET_FILES = (
    "openstack/key_manager/v1/secret.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/key_manager/v1/_proxy.py",
    "openstack/key_manager/v1/_format.py",
    "openstack/format.py",
)
SECRET_ANCHORS = (
    ("openstack/key_manager/v1/secret.py", "Secret"),
    ("openstack/key_manager/v1/secret.py", "Secret._query_mapping"),
    ("openstack/key_manager/v1/secret.py", "Secret.secret_id"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "Resource._alternate_id"),
    ("openstack/resource.py", "Resource._get_id"),
    ("openstack/resource.py", "Resource.__init__"),
    ("openstack/resource.py", "Resource._attributes_iterator"),
    ("openstack/resource.py", "Resource._attr_to_dict"),
    ("openstack/resource.py", "Resource.to_dict"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/key_manager/v1/_proxy.py", "Proxy.secrets"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID.deserialize"),
    ("openstack/format.py", "Formatter"),
)
CONTAINER_RESOURCE = "openstack.key_manager.v1.container.Container"
CONTAINER_FILES = (
    "openstack/key_manager/v1/container.py",
    "openstack/resource.py",
    "openstack/fields.py",
    "openstack/proxy.py",
    "openstack/key_manager/v1/_proxy.py",
    "openstack/key_manager/v1/_format.py",
    "openstack/format.py",
)
CONTAINER_ANCHORS = (
    ("openstack/key_manager/v1/container.py", "Container"),
    # Container has no own query mapping: prove the effective inherited
    # QueryParameters() declaration and its limit/marker defaults separately.
    ("openstack/resource.py", "Resource._query_mapping"),
    ("openstack/key_manager/v1/container.py", "Container.container_id"),
    ("openstack/resource.py", "Resource.id"),
    ("openstack/resource.py", "Resource.name"),
    ("openstack/resource.py", "Resource.__getattribute__"),
    ("openstack/resource.py", "Resource._alternate_id"),
    ("openstack/resource.py", "Resource._get_id"),
    ("openstack/resource.py", "Resource.__init__"),
    ("openstack/resource.py", "Resource._attributes_iterator"),
    ("openstack/resource.py", "Resource._attr_to_dict"),
    ("openstack/resource.py", "Resource.to_dict"),
    ("openstack/resource.py", "QueryParameters.__init__"),
    ("openstack/resource.py", "QueryParameters._validate"),
    ("openstack/resource.py", "QueryParameters._transpose"),
    ("openstack/resource.py", "Resource.list"),
    ("openstack/fields.py", "_BaseComponent.__get__"),
    ("openstack/fields.py", "_convert_type"),
    ("openstack/proxy.py", "Proxy._list"),
    ("openstack/key_manager/v1/_proxy.py", "Proxy.containers"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID"),
    ("openstack/key_manager/v1/_format.py", "HREFToUUID.deserialize"),
    ("openstack/format.py", "Formatter"),
)
TARGETS = {
    "subnet": (RESOURCE, FILES, ANCHORS, "gophercloudsdk/network/v2/subnets"),
    "secret": (SECRET_RESOURCE, SECRET_FILES, SECRET_ANCHORS,
               "gophercloudsdk/keymanager/v1/secrets"),
    "container": (CONTAINER_RESOURCE, CONTAINER_FILES, CONTAINER_ANCHORS,
                  "gophercloudsdk/keymanager/v1/containers"),
}


def assignment(node):
    if isinstance(node, ast.Assign) and len(node.targets) == 1:
        if isinstance(node.targets[0], ast.Name):
            return node.targets[0].id, node.value
    if isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
        return node.target.id, node.value
    return None


class Source:
    def __init__(self, root, files=FILES):
        self.raw = {}
        self.trees = {}
        self.modules = {}
        self.imports = {}
        self.classes = {}
        self.attributes = {}
        for path in files:
            raw = (root / path).read_bytes()
            module = path[:-3].replace("/", ".")
            tree = ast.parse(raw, filename=path)
            self.raw[path] = raw
            self.trees[path] = tree
            self.modules[module] = tree
            aliases = {}
            for node in tree.body:
                if isinstance(node, ast.ImportFrom) and node.level == 0:
                    for alias in node.names:
                        aliases[alias.asname or alias.name] = (
                            node.module + "." + alias.name
                        )
                elif isinstance(node, ast.Import):
                    for alias in node.names:
                        aliases[alias.asname or alias.name.split(".")[0]] = (
                            alias.name if alias.asname else alias.name.split(".")[0]
                        )
                elif isinstance(node, ast.ClassDef):
                    qualified = module + "." + node.name
                    self.classes[qualified] = node
                    attrs = {}
                    for member in node.body:
                        pair = assignment(member)
                        if pair:
                            attrs[pair[0]] = (module, pair[1], member)
                        elif isinstance(member, (ast.FunctionDef, ast.AsyncFunctionDef)):
                            attrs[member.name] = (module, member, member)
                    self.attributes[qualified] = attrs
            self.imports[module] = aliases

    def resolve(self, module, node):
        if isinstance(node, ast.Subscript):
            return self.resolve(module, node.value)
        if isinstance(node, ast.Name):
            if node.id in self.imports[module]:
                result = self.imports[module][node.id]
            elif node.id in {"dict", "list", "str", "int", "bool", "object"}:
                result = "builtins." + node.id
            else:
                result = module + "." + node.id
        elif isinstance(node, ast.Attribute):
            result = self.resolve(module, node.value) + "." + node.attr
        else:
            raise ValueError("unsupported reference: " + ast.dump(node))
        parent, _, name = result.rpartition(".")
        if parent in self.imports and name in self.imports[parent]:
            return self.imports[parent][name]
        return result

    def bases(self, qualified):
        if qualified not in self.classes:
            if qualified not in {"builtins.dict", "typing.Protocol"}:
                raise ValueError("unreviewed class base: " + qualified)
            return []
        module = qualified.rpartition(".")[0]
        return [self.resolve(module, base) for base in self.classes[qualified].bases]

    def mro(self, qualified):
        bases = self.bases(qualified)
        result = [qualified]
        sequences = [self.mro(base) for base in bases] + [bases[:]]
        while any(sequences):
            sequences = [sequence for sequence in sequences if sequence]
            candidate = next(
                (sequence[0] for sequence in sequences
                 if all(sequence[0] not in other[1:] for other in sequences)),
                None,
            )
            if candidate is None:
                raise ValueError("inconsistent class MRO")
            result.append(candidate)
            for sequence in sequences:
                if sequence[0] == candidate:
                    sequence.pop(0)
        return result

    def effective_attributes(self, qualified):
        attrs = {}
        for base in reversed(self.mro(qualified)):
            attrs.update(self.attributes.get(base, {}))
        return attrs

    def literal(self, module, node):
        if isinstance(node, (ast.Name, ast.Attribute)):
            reference = self.resolve(module, node)
            parent, _, member = reference.rpartition(".")
            if parent in self.classes:
                mod, value, _ = self.effective_attributes(parent)[member]
                return self.literal(mod, value)
        return ast.literal_eval(node)

    def anchor(self, path, symbol):
        nodes = self.trees[path].body
        for name in symbol.split("."):
            found = []
            for node in nodes:
                pair = assignment(node)
                if getattr(node, "name", None) == name or (pair and pair[0] == name):
                    if isinstance(node, ast.FunctionDef) and any(
                        isinstance(decorator, ast.Name) and decorator.id == "overload"
                        for decorator in node.decorator_list
                    ):
                        continue
                    found.append(node)
            if len(found) != 1:
                raise ValueError("missing/ambiguous source anchor: " + symbol)
            node = found[0]
            nodes = getattr(node, "body", [])
        return node


def control_arguments(node, excluded):
    args = node.args.posonlyargs + node.args.args + node.args.kwonlyargs
    return [arg.arg for arg in args if arg.arg not in excluded]


def implementation_policies(source):
    listing = source.anchor("openstack/resource.py", "Resource.list")
    validates = [node for node in ast.walk(listing) if isinstance(node, ast.Call)
                 and isinstance(node.func, ast.Attribute) and node.func.attr == "_validate"]
    if len(validates) != 1:
        raise ValueError("unsupported list query validation")
    unknown = [kw.value for kw in validates[0].keywords if kw.arg == "allow_unknown_params"]
    if len(unknown) != 1 or not isinstance(unknown[0], ast.Constant) or unknown[0].value is not True:
        raise ValueError("unsupported unknown filter policy")
    transposing = source.anchor("openstack/resource.py", "QueryParameters._transpose")
    def membership(node, key):
        return (isinstance(node, ast.Compare) and isinstance(node.left, ast.Name)
                and node.left.id == key and len(node.ops) == 1
                and isinstance(node.ops[0], ast.In) and len(node.comparators) == 1
                and isinstance(node.comparators[0], ast.Name) and node.comparators[0].id == "query")
    def selects(nodes, key):
        if len(nodes) != 1 or not isinstance(nodes[0], ast.Assign):
            return False
        value = nodes[0].value
        return (isinstance(value, ast.Subscript) and isinstance(value.value, ast.Name)
                and value.value.id == "query" and isinstance(value.slice, ast.Name)
                and value.slice.id == key)
    canonical = [node for node in ast.walk(transposing) if isinstance(node, ast.If)
                 and membership(node.test, "client_side")]
    if len(canonical) != 1 or not selects(canonical[0].body, "client_side"):
        raise ValueError("unsupported canonical query precedence")
    fallback = canonical[0].orelse
    if (len(fallback) != 1 or not isinstance(fallback[0], ast.If)
            or not membership(fallback[0].test, "name") or not selects(fallback[0].body, "name")):
        raise ValueError("unsupported wire query precedence")
    return "discard", "canonical_client_name_wins"


def keymanager_body_accessors(source, attrs, body, label, alternate_id, ref):
    """Keep Resource.id separate from the formatted alternate-ID descriptor."""
    if body.get("id") != {"field": "id", "response_type": None}:
        raise ValueError("unsupported " + label + " literal ID descriptor")
    module, descriptor, _ = attrs[alternate_id]
    alternate = [keyword.value for keyword in descriptor.keywords
                 if keyword.arg == "alternate_id"]
    if len(alternate) != 1 or source.literal(module, alternate[0]) is not True:
        raise ValueError("unsupported " + label + " alternate ID descriptor")
    formatter = "openstack.key_manager.v1._format.HREFToUUID"
    if body.get(alternate_id) != {"field": ref, "response_type": formatter}:
        raise ValueError("unsupported " + label + " alternate ID formatter")
    if body.get(ref) != {"field": ref, "response_type": None}:
        raise ValueError("unsupported " + label + " reference descriptor")
    formatter_class = source.classes[formatter]
    if (len(formatter_class.bases) != 1
            or source.resolve(formatter.rpartition(".")[0], formatter_class.bases[0])
            != "openstack.format.Formatter"):
        raise ValueError("unsupported " + label + " formatter base")
    # Resource.__getattribute__ reads the stored literal ID before consulting
    # the alternate wire field. It does not apply HREFToUUID to that fallback.
    # Separate AST anchors prove both access paths and the to_dict projection.
    body["id"]["response_accessor"] = "resource_id"
    body[alternate_id]["formatter"] = formatter


def extract(root, target="subnet"):
    resource, files, anchors, sdk_package = TARGETS[target]
    source = Source(root, files)
    attrs = source.effective_attributes(resource)
    module, query_call, _ = attrs["_query_mapping"]
    if not isinstance(query_call, ast.Call) or source.resolve(module, query_call.func) != "openstack.resource.QueryParameters":
        raise ValueError("unsupported query mapping declaration")
    init = source.anchor("openstack/resource.py", "QueryParameters.__init__")
    pagination = dict(zip(
        (arg.arg for arg in init.args.kwonlyargs), init.args.kw_defaults
    ))["include_pagination_defaults"]
    include_defaults = source.literal("openstack.resource", pagination)
    keywords = {}
    for keyword in query_call.keywords:
        if keyword.arg is None:
            expansion = source.literal(module, keyword.value)
            if not isinstance(expansion, dict):
                raise ValueError("non-dictionary query expansion")
            keywords.update(expansion)
        else:
            keywords[keyword.arg] = source.literal(module, keyword.value)
    include_defaults = keywords.pop("include_pagination_defaults", include_defaults)
    if not isinstance(include_defaults, bool):
        raise ValueError("non-boolean pagination default")
    defaults = [node for node in ast.walk(init) if isinstance(node, ast.If)
                and isinstance(node.test, ast.Name) and node.test.id == "include_pagination_defaults"]
    if len(defaults) != 1 or len(defaults[0].body) != 1:
        raise ValueError("unsupported pagination defaults implementation")
    update = defaults[0].body[0]
    if (not isinstance(update, ast.Expr) or not isinstance(update.value, ast.Call)
            or len(update.value.args) != 1 or not isinstance(update.value.func, ast.Attribute)
            or update.value.func.attr != "update"):
        raise ValueError("unsupported pagination defaults mapping")
    mappings = source.literal("openstack.resource", update.value.args[0]) if include_defaults else {}
    for name in query_call.args:
        value = source.literal(module, name)
        if not isinstance(value, str):
            raise ValueError("non-string query key")
        mappings[value] = value
    mappings.update(keywords)
    query, formats = {}, {}
    for key, value in mappings.items():
        if isinstance(value, dict):
            if set(value) - {"name", "format"}:
                raise ValueError("unsupported query formatting policy")
            wire = value.get("name", key)
            if value.get("format") is not None:
                formats[key] = value["format"]
        else:
            wire = value
        if not isinstance(key, str) or not isinstance(wire, str):
            raise ValueError("non-string query mapping")
        query[key] = wire
    body, uri = {}, {}
    for name, (module, value, _) in attrs.items():
        if not isinstance(value, ast.Call):
            continue
        kind = source.resolve(module, value.func)
        if kind not in {"openstack.fields.Body", "openstack.fields.URI", "openstack.resource.Body", "openstack.resource.URI"}:
            continue
        if kind.endswith(".Body") and name in query:
            continue
        if len(value.args) != 1:
            raise ValueError("unsupported field name declaration")
        field = source.literal(module, value.args[0])
        typed = next((kw.value for kw in value.keywords if kw.arg == "type"), None)
        response_type = source.resolve(module, typed).removeprefix("builtins.") if typed else None
        (body if kind.endswith(".Body") else uri)[name] = {
            "field": field, "response_type": response_type
        }
    if target == "secret":
        keymanager_body_accessors(source, attrs, body, "Secret", "secret_id", "secret_ref")
    elif target == "container":
        keymanager_body_accessors(source, attrs, body, "Container", "container_id", "container_ref")
    resource_controls = control_arguments(
        source.anchor("openstack/resource.py", "Resource.list"), {"cls"}
    )
    proxy_controls = control_arguments(
        source.anchor("openstack/proxy.py", "Proxy._list"), {"self"}
    )
    unknown_filters, query_collision = implementation_policies(source)
    proof = []
    for path, symbol in anchors:
        node = source.anchor(path, symbol)
        proof.append({
            "source": path, "symbol": symbol,
            "line": node.lineno, "end_line": node.end_lineno,
            "ast_sha256": hashlib.sha256(ast.dump(
                node, annotate_fields=True, include_attributes=False
            ).encode("utf-8")).hexdigest(),
        })
    return {
        "schema_version": 1, "source_pin": PIN, "resource": resource,
        "sdk_package": sdk_package,
        "base_path": source.literal(*attrs["base_path"][:2]),
        "envelope": source.literal(*attrs["resources_key"][:2]),
        "class_bases": [ast.unparse(base) for base in source.classes[resource].bases],
        "mro": source.mro(resource),
        "query": query, "query_formats": formats, "body": body, "uri": uri,
        "unknown_filters": unknown_filters,
        "query_collision": query_collision,
        "counts": {
            "canonical_query": len(query),
            "accepted_query": len(set(query) | set(query.values())),
            "local_body": len(body),
        },
        "source_controls": {"resource_list": resource_controls, "proxy_list": proxy_controls},
        "reserved": sorted(set(resource_controls) | set(proxy_controls)),
        "proof": {
            "ast_algorithm": 'sha256(ast.dump(node, annotate_fields=True, include_attributes=False).encode("utf-8"))',
            "python_parser": ".".join(map(str, sys.version_info[:2])),
            "nodes": proof,
            "files": {path: hashlib.sha256(raw).hexdigest() for path, raw in source.raw.items()},
        },
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path)
    parser.add_argument("--target", choices=tuple(TARGETS))
    parser.add_argument("--resource", choices=tuple(spec[0] for spec in TARGETS.values()))
    args = parser.parse_args()
    target = args.target
    if args.resource:
        resource_target = next(name for name, spec in TARGETS.items()
                               if spec[0] == args.resource)
        if target is not None and target != resource_target:
            parser.error("--resource and --target select different resources")
        target = resource_target
    try:
        data = json.dumps(extract(args.source, target or "subnet"), indent=2, sort_keys=True) + "\n"
    except (OSError, SyntaxError, ValueError, KeyError, TypeError, AttributeError) as error:
        parser.exit(1, "Python filter source: " + str(error) + "\n")
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(data, encoding="utf-8")
    else:
        sys.stdout.write(data)


if __name__ == "__main__":
    main()
