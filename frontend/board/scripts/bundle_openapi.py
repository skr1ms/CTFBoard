"""Bundle an OpenAPI spec with multi-file $refs into a single self-contained JSON."""

import json
import os
import sys

import yaml

_cache: dict[str, object] = {}


def load_yaml(path: str) -> object:
    path = os.path.normpath(path)
    if path in _cache:
        return _cache[path]
    with open(path) as f:
        data = yaml.safe_load(f)
    _cache[path] = data
    return data


def resolve_fragment(doc: object, fragment: str) -> object:
    if not fragment:
        return doc
    parts = fragment.lstrip("/").split("/")
    node = doc
    for p in parts:
        p = p.replace("~1", "/").replace("~0", "~")
        if isinstance(node, dict):
            node = node[p]  # type: ignore[index]
        elif isinstance(node, list):
            node = node[int(p)]  # type: ignore[index]
        else:
            raise KeyError(f"Cannot index {type(node)} with {p!r}")
    return node


def normalize_local_ref(ref: str) -> str:
    """Rewrite #/schemas/pkg.Name or #/schemas/Name -> #/components/schemas/ShortName."""
    if not ref.startswith("#/schemas/"):
        return ref
    name = ref[len("#/schemas/") :]
    # strip dotted prefix (e.g. request.LoginRequest -> LoginRequest)
    if "." in name:
        name = name.split(".", 1)[1]
    return f"#/components/schemas/{name}"


def resolve_refs(node: object, base_path: str) -> object:
    if isinstance(node, dict):
        if "$ref" in node and isinstance(node["$ref"], str):
            ref: str = node["$ref"]
            if ref.startswith("#/schemas/"):
                return {"$ref": normalize_local_ref(ref)}
            if ref.startswith("#"):
                return node
            file_part, _, fragment = ref.partition("#")
            ref_path = os.path.normpath(
                os.path.join(os.path.dirname(base_path), file_part)
            )
            ref_doc = load_yaml(ref_path)
            resolved = resolve_fragment(ref_doc, fragment)
            return resolve_refs(resolved, ref_path)
        return {k: resolve_refs(v, base_path) for k, v in node.items()}
    if isinstance(node, list):
        return [resolve_refs(item, base_path) for item in node]
    return node


if __name__ == "__main__":
    spec_path, out_path = sys.argv[1], sys.argv[2]
    spec = load_yaml(spec_path)
    resolved = resolve_refs(spec, spec_path)
    with open(out_path, "w") as f:
        json.dump(resolved, f)
    print(f"Bundled {spec_path} -> {out_path}")
