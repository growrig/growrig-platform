# GrowRig catalog schemas

JSON Schema (draft 2020-12) definitions for every file in a GrowRig **catalog** —
the content packages that provide devices, integrations, species, inventory and
vendors. See the [catalog repository](https://github.com/growrig/growrig-catalog)
for the canonical example and [growrig.dev/docs](https://growrig.dev/docs/) for
the authoring guides.

The schemas are authored here in YAML and published as JSON at
`https://growrig.dev/schema/catalog/<name>.json` (emitted by the docs site
build). Point an editor at them for autocomplete and inline validation with a
header comment on any catalog file:

```yaml
# yaml-language-server: $schema=https://growrig.dev/schema/catalog/device.json
```

| File in this dir | Applies to | `$id` |
|---|---|---|
| `manifest.schema.yaml` | `catalog.yaml` | `…/catalog/manifest.json` |
| `device.schema.yaml` | `devices/<category>/<id>/device.yaml` | `…/catalog/device.json` |
| `integration.schema.yaml` | `integrations/<category>/<id>/integration.yaml` | `…/catalog/integration.json` |
| `species.schema.yaml` | `species/<id>/species.yaml` | `…/catalog/species.json` |
| `feedings.schema.yaml` | `species/<id>/feedings.yaml` | `…/catalog/feedings.json` |
| `inventory.schema.yaml` | `inventory/<category>/inventory.yaml` | `…/catalog/inventory.json` |
| `products.schema.yaml` | `inventory/<category>/products.yaml` | `…/catalog/products.json` |
| `vendor.schema.yaml` | `vendors/<id>/vendor.yaml` | `…/catalog/vendor.json` |

## Who enforces them

- **Grow Core** validates every content file against these schemas when it fetches
  a custom catalog source, so a malformed third-party catalog is rejected with a
  clear error instead of half-loading (`growcore/internal/catalogsource`).
- **The catalog repo's CI** validates the whole tree on every pull request.
- **Editors** surface problems live via the `$schema` header comment above.

## Conventions

Objects are **closed** (`additionalProperties: false`): an unknown key is a
likely typo and is rejected. Adding a new field therefore means updating the
corresponding schema here in the same change. Directory-derived values (a
device's category and id, a species or vendor id) are **not** part of the file
schemas — they come from the path.
