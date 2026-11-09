---
page_title: "lucida_flag (Resource)"
subcategory: ""
description: |-
  A feature flag, keyed by its flag key. `kind` is fixed at create.
---

# lucida_flag (Resource)

A feature flag, keyed by its flag key. `kind` is fixed at create.

Manages the `flags` collection, the same kind `lucida pull|plan|apply` calls `flags`.

## Example Usage

```terraform
resource "lucida_flag" "example" {
  key = "new-checkout"
  definition = jsonencode({ name = "New checkout", enabled = true, filters = { rollout = 25 } })
}
```

## Schema

### Required

- `key` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `name`, `description`, `kind`, `filters`, `variants`, `payloads`, `stickiness`, `tags`, `enabled`. Changing `key` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_flag.example <id>
```
