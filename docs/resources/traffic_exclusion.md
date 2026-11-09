---
page_title: "lucida_traffic_exclusion (Resource)"
subcategory: ""
description: |-
  A traffic exclusion rule. The note is its identity; only `enabled` can change in place.
---

# lucida_traffic_exclusion (Resource)

A traffic exclusion rule. The note is its identity; only `enabled` can change in place.

Manages the `traffic-exclusions` collection, the same kind `lucida pull|plan|apply` calls `traffic-exclusions`.

## Example Usage

```terraform
resource "lucida_traffic_exclusion" "example" {
  note = "Office network"
  definition = jsonencode({ kind = "ip", value = "203.0.113.0/24", enabled = true })
}
```

## Schema

### Required

- `note` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `kind`, `key`, `value`, `enabled`. Changing `kind`, `key`, `value` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_traffic_exclusion.example <id>
```
