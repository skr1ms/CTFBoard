#!/usr/bin/env python3
"""Merge all schema files into components/schemas.yml"""

import re
from pathlib import Path

import yaml

SCHEMAS_DIR = Path("internal/openapi/components/schemas")
SCHEMAS_FILE = Path("internal/openapi/components/schemas.yml")

SCHEMA_FILES = [
    "common_schemas.yml",
    "auth_schemas.yml",
    "email_schemas.yml",
    "user_schemas.yml",
    "team_schemas.yml",
    "challenge_schemas.yml",
    "comment_schemas.yml",
    "submission_schemas.yml",
    "settings_schemas.yml",
    "backup_schemas.yml",
    "competition_params_schemas.yml",
    "file_schemas.yml",
    "competition_schemas.yml",
    "websocket_schemas.yml",
    "api_token_schemas.yml",
    "notification_schemas.yml",
    "page_schemas.yml",
    "bracket_schemas.yml",
    "field_schemas.yml",
    "tag_schemas.yml",
    "award_schemas.yml",
    "avatar_schemas.yml",
    "statistics_schemas.yml",
    "scoreboard_schemas.yml",
    "ban_appeal_schemas.yml",
    "storage_schemas.yml",
]


def normalize_schema_name(name):
    """Remove prefixes like response., request., v1., jwt., entity. from schema names"""
    # Remove common prefixes
    prefixes = ["response.", "request.", "v1.", "jwt.", "entity."]
    for prefix in prefixes:
        if name.startswith(prefix):
            return name[len(prefix) :]
    return name


def convert_refs(obj, name_mapping):
    """Recursively convert #/schemas/ refs to #/ refs and normalize schema names"""
    if isinstance(obj, dict):
        result = {}
        for key, value in obj.items():
            if key == "$ref" and isinstance(value, str):
                # Handle internal refs: #/schemas/response.X -> #/X
                if value.startswith("#/schemas/"):
                    old_name = value.replace("#/schemas/", "")
                    new_name = normalize_schema_name(old_name)
                    if old_name in name_mapping:
                        result[key] = f"#/{name_mapping[old_name]}"
                    else:
                        result[key] = f"#/{new_name}"
                # Handle external refs: ../components/schemas.yml#/schemas/response.X -> ../components/schemas.yml#/X
                elif "../components/schemas.yml#/schemas/" in value:
                    old_name = value.split("#/schemas/")[-1]
                    new_name = normalize_schema_name(old_name)
                    if old_name in name_mapping:
                        result[key] = (
                            f"../components/schemas.yml#/{name_mapping[old_name]}"
                        )
                    else:
                        result[key] = f"../components/schemas.yml#/{new_name}"
                else:
                    result[key] = value
            else:
                result[key] = convert_refs(value, name_mapping)
        return result
    elif isinstance(obj, list):
        return [convert_refs(item, name_mapping) for item in obj]
    else:
        return obj


all_schemas = {}
name_mapping = {}  # old_name -> new_name

# First pass: collect all schemas and build name mapping
for schema_file in SCHEMA_FILES:
    file_path = SCHEMAS_DIR / schema_file
    if file_path.exists():
        with open(file_path, "r") as f:
            content = yaml.safe_load(f)
            if content and "schemas" in content and content["schemas"]:
                file_schemas = content["schemas"]
                for old_name, schema_def in file_schemas.items():
                    new_name = normalize_schema_name(old_name)
                    if new_name in all_schemas:
                        print(
                            f"Warning: Duplicate normalized schema name '{new_name}' (from '{old_name}')"
                        )
                    name_mapping[old_name] = new_name
                    all_schemas[new_name] = schema_def

# Second pass: convert all refs using the mapping
all_schemas_converted = {}
for new_name, schema_def in all_schemas.items():
    all_schemas_converted[new_name] = convert_refs(schema_def, name_mapping)

# Write without the root "schemas:" key - schemas should be at the root level
with open(SCHEMAS_FILE, "w") as f:
    yaml.dump(
        all_schemas_converted,
        f,
        default_flow_style=False,
        sort_keys=False,
        allow_unicode=True,
        width=1000,
    )

print(f"Merged {len(all_schemas_converted)} schemas into {SCHEMAS_FILE}")
print("Normalized schema names (removed prefixes)")

# Fix references in route files
print("\nFixing references in route files...")
ROUTES_DIR = Path("internal/openapi/routes")


def fix_refs_in_file(file_path):
    """Fix schema references in a route file"""
    with open(file_path, "r") as f:
        content = f.read()

    # Pattern: ../components/schemas.yml#/schemas/response.X or request.X etc.
    pattern = r"('../components/schemas\.yml#/schemas/)([^']+)'"

    def replace_ref(match):
        old_name = match.group(2)  # 'response.X' or 'request.X' etc.
        new_name = normalize_schema_name(old_name)
        # Replace #/schemas/response.X with #/X
        return f"'../components/schemas.yml#/{new_name}'"

    new_content = re.sub(pattern, replace_ref, content)

    if new_content != content:
        with open(file_path, "w") as f:
            f.write(new_content)
        return True
    return False


fixed_count = 0
for route_file in ROUTES_DIR.glob("*.yml"):
    if fix_refs_in_file(route_file):
        print(f"Fixed refs in {route_file.name}")
        fixed_count += 1

if fixed_count > 0:
    print(f"Fixed references in {fixed_count} route files")
else:
    print("Route files already up to date")
